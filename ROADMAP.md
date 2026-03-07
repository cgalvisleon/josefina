# ROADMAP.md

## `internal/store` — Production Readiness Evaluation

The design follows the **Bitcask** model: append-only WAL segments, a fully in-memory hash index, tombstone deletes, periodic compaction, and snapshots for faster restarts. The concept is sound. The implementation has several correctness bugs that would make it unsafe for production as written.

---

### Critical Bugs (data loss / corruption risk)

**1. Write errors are silently discarded** — `segment.go:97-99`
```go
func (s *segment) loop() {
    for data := range s.ch {
        s.file.Write(data)  // error is ignored
    }
}
```
If the disk is full or an I/O error occurs, the write silently fails. The caller receives no error. `s.size` is already updated in `WriteRecord`, so the in-memory index points to a byte range that was never written. On the next read or restart, that record is corrupted or missing.

**2. `s.size` is updated before the write actually happens** — `segment.go:197-215`
```go
s.Write(header)       // sends to channel → async
s.size += h.RecordSize()  // updated immediately
```
`Write` pushes bytes to an unbuffered channel and returns only when `loop()` accepts them — not when they are written to disk. `s.size` is already wrong if the write fails (see above), and the offset returned to the caller is based on a size that doesn't reflect what's on disk.

**3. `writeMu` does not serialize actual disk writes** — `store.go:226-264`
`appendRecord` holds `writeMu`, but `WriteRecord` just enqueues onto the channel. The actual `file.Write` call happens in `loop()` with no lock. Multiple goroutines writing to different segments simultaneously is safe today only because each segment has its own goroutine, but the ordering guarantee expected from a WAL is not enforced.

**4. Data race: `s.keys` mutated under `RLock`** — `store.go:396-403`, `store.go:454-457`

In `getRecords`:
```go
s.indexMu.RLock()
sort.Strings(s.keys)   // mutates shared slice under read lock
```
And in `Keys`:
```go
s.indexMu.RLock()
sort.Strings(s.keys)   // same
```
`sort.Strings` modifies the slice in place. A `RLock` allows concurrent readers, so two goroutines can race on this mutation. The Go race detector will catch this immediately.

**5. Panic on pagination bounds** — `store.go:406-413`, `store.go:462-468`

In `getRecords` and `Keys` the loop does:
```go
k := s.keys[offset]
offset++
if i >= limit { break }
```
There is no check that `offset < len(s.keys)` before indexing. If `limit` exceeds the remaining entries, this panics with an index out of range.

**6. Concurrent compaction races with active readers** — `compact.go:108-128`

```go
s.writeMu.Lock()
for _, seg := range s.segments {
    seg.file.Close()   // closes file descriptors
}
// ...
s.indexMu.Lock()
s.segments = newSegments
```
Between the `file.Close()` calls and the `s.segments` swap, any concurrent `Get` that already acquired `indexMu.RLock` and is inside `seg.ReadAt` will call `ReadAt` on a closed file, causing an error or undefined behavior. The two locks are taken in different order in different code paths, which is also a classic deadlock setup.

**7. Multiple concurrent compactions can be triggered** — `store.go:258-261`

```go
if s.TombStones > threshold {
    go s.Compact()
}
```
This is inside `appendRecord`, which runs on every write. Once the threshold is exceeded, every subsequent write launches another compaction goroutine. There is no guard preventing simultaneous compactions.

---

### Correctness Concerns (not immediately fatal but wrong)

**8. `buildIndex` only replays the last segment** — `store.go:368-371`
```go
func (s *FileStore) buildIndex() error {
    idx := len(s.segments) - 1
    return s.rebuildIndex(idx)
}
```
On startup, this only scans segment N-1. Segments 0 through N-2 are expected to be covered by the snapshot. If the snapshot is missing, stale, or written before compaction, records in earlier segments are invisible. There is no fallback full-scan path.

**9. Snapshot excludes the current (active) segment** — `snapshot.go:41-45`
```go
currentSegment := len(s.segments) - 1
for id, ref := range s.index {
    if ref.segment == currentSegment {
        continue
    }
}
```
This is intentional — recovery depends on `buildIndex` replaying the active segment. But if the snapshot is created at segment rotation (before the new segment has any entries) and then the process crashes before a new snapshot, a restart loads an incomplete snapshot and only partially scans. The invariant is fragile and undocumented.

**10. `WriteRecord` sends header and data as two separate channel messages** — `segment.go:204-208`
```go
s.Write(header)
if len(data) > 0 {
    s.Write(data)
}
```
Two separate `file.Write` calls with no atomicity. A crash between the two produces a torn record — the header is on disk, the data is not. `rebuildIndex` would stop at the corrupt record but would not skip it and resume, so all later records in that segment are lost too.

---

### Performance Notes

| Area | Assessment |
|---|---|
| CRC32C (Castagnoli) checksum | Good — hardware-accelerated on x86/arm, fast |
| Atomic segment files | Good — `os.Rename` for atomic swap in snapshot and compaction |
| `getRecords` re-sorts `s.keys` on every paginated call | O(n log n) per query; unnecessary if keys are kept sorted at write time |
| Compaction reads each live record with individual `ReadAt` calls | Many small syscalls; a sequential scan per segment would be faster |
| Snapshot allocated with `bytes.NewBuffer(nil)` | Grows dynamically; pre-allocating with index size would reduce allocations |
| Fully in-memory index | Correct Bitcask tradeoff — scales only to available RAM |
| No userspace read cache | Every `Get` is a syscall; fine for throughput, high latency for hot keys |

---

### Summary

| Category | Status |
|---|---|
| Conceptual design (Bitcask-style WAL + index) | Sound |
| Write durability | **Broken** — errors silently swallowed |
| Concurrent safety | **Broken** — data race on `s.keys`, unsafe compaction swap |
| Crash recovery | **Fragile** — torn records, partial snapshots, single-segment replay |
| Production readiness | **Not yet** |

### Priority Fix List

The most urgent fixes, in order of severity:

1. Propagate write errors from `loop()` back to the caller (eliminates silent data loss)
2. Merge header + data into a single `Write` call to prevent torn records
3. Guard `s.keys` mutations with `WLock` instead of `RLock` (eliminates data race)
4. Add bounds checks in `getRecords` / `Keys` loops (eliminates panic)
5. Serialize compaction with an atomic flag or dedicated mutex (eliminates concurrent compaction launches)
6. Fix compaction swap — readers must be drained before closing old segment file descriptors
7. Add a full-scan fallback path if the snapshot is missing or fails CRC validation
