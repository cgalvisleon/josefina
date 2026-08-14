package jdb

import (
	"encoding/json"
	"slices"
	"sort"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/josefina/internal/store"
)

const bpDegree = 32 // minimum degree: each node holds between t-1 and 2t-1 keys

/**
* bpNode is an internal or leaf node of the B+ tree.
* Internal nodes: keys holds separators, children holds child pointers, vals is nil.
* Leaf nodes: vals holds primary-key sets per key, children is nil, next links to the next leaf.
**/
type bpNode struct {
	keys     []IndexKey // sorted keys
	children []*bpNode  // internal: len(keys)+1 children; leaf: nil
	vals     [][]string // leaf: set of primary keys per key; internal: nil
	next     *bpNode    // linked list of leaves (ascending)
	leaf     bool
}

/**
* BTree is a thread-safe B+ tree: IndexKey → set of string values (primary keys).
* If st != nil, every Insert/Delete is automatically persisted to the FileStore.
**/
type BTree struct {
	root *bpNode
	t    int              // minimum degree
	size int              // number of distinct keys
	st   *store.FileStore // nil = memory-only
	mu   sync.RWMutex
}

/**
* NewBTree: Creates a memory-only BTree (no persistence).
* @return *BTree
**/
func NewBTree() *BTree {
	return &BTree{
		root: &bpNode{leaf: true},
		t:    bpDegree,
	}
}

/**
* OpenBTree: Opens (or creates) a BTree backed by a FileStore at path/name.
* Call Init() to load existing data from the store.
* @param path, name string
* @return *BTree, error
**/
func OpenBTree(pathData, pathWald, name string) (*BTree, error) {
	st, err := store.Open(pathData, pathWald, "btidx_"+name, store.ReadWrite)
	if err != nil {
		return nil, err
	}
	result := &BTree{
		root: &bpNode{leaf: true},
		t:    bpDegree,
		st:   st,
	}

	err = result.initFromStore()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* initFromStore: Loads all records from the FileStore into the in-memory tree.
* Only has effect if the BTree was opened with OpenBTree.
* @return error
**/
func (bt *BTree) initFromStore() error {
	if bt.st == nil {
		return nil
	}
	bt.mu.Lock()
	defer bt.mu.Unlock()

	return bt.st.ForEach(func(storeKey string, data []byte) (bool, error) {
		key, err := decodeKey(storeKey)
		if err != nil {
			return true, nil // skip malformed entries
		}
		var pks []string
		if err := json.Unmarshal(data, &pks); err != nil {
			return true, nil
		}
		bt.insertSlice(key, pks)
		return true, nil
	}, true, 0, 0)
}

/**
* persistKey: Writes the current values of key to the FileStore.
* Must be called with bt.mu already held.
* @param key IndexKey
* @return error
**/
func (bt *BTree) persistKey(key IndexKey) error {
	if bt.st == nil {
		return nil
	}
	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i >= len(leaf.keys) || leaf.keys[i].Compare(key) != 0 {
		_, err := bt.st.Delete(encodeKey(key))
		return err
	}
	_, _, err := bt.st.Put(encodeKey(key), leaf.vals[i])
	return err
}

/**
* Len: Returns the number of distinct keys stored.
* @return int
**/
func (bt *BTree) Len() int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.size
}

/**
* childIdx: Returns the child index to follow in an internal node (upper bound).
* @param keys []IndexKey, key IndexKey
* @return int
**/
func childIdx(keys []IndexKey, key IndexKey) int {
	return sort.Search(len(keys), func(i int) bool { return key.Compare(keys[i]) < 0 })
}

/**
* leafSearch: Returns the lower bound of key in the sorted keys of a leaf.
* @param keys []IndexKey, key IndexKey
* @return int
**/
func leafSearch(keys []IndexKey, key IndexKey) int {
	lo, hi := 0, len(keys)
	for lo < hi {
		mid := (lo + hi) >> 1
		if key.Compare(keys[mid]) <= 0 {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}

/**
* findLeaf: Navigates from the root to the leaf that should contain key.
* @param key IndexKey
* @return *bpNode
**/
func (bt *BTree) findLeaf(key IndexKey) *bpNode {
	n := bt.root
	for !n.leaf {
		n = n.children[childIdx(n.keys, key)]
	}
	return n
}

/**
* Get: Returns the values associated with key. ok=false if key does not exist.
* @param key IndexKey
* @return []string, bool
**/
func (bt *BTree) Get(key IndexKey) ([]string, bool) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i < len(leaf.keys) && leaf.keys[i].Compare(key) == 0 {
		out := make([]string, len(leaf.vals[i]))
		copy(out, leaf.vals[i])
		return out, true
	}
	return nil, false
}

/**
* Insert: Adds value to the set stored at key and persists if a store is attached.
* @param key IndexKey, value string
* @return error
**/
func (bt *BTree) Insert(key IndexKey, value string) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	if len(bt.root.keys) == 2*bt.t-1 {
		newRoot := &bpNode{children: []*bpNode{bt.root}}
		bt.splitChild(newRoot, 0)
		bt.root = newRoot
	}
	bt.insertNonFull(bt.root, key, value)
	return bt.persistKey(key)
}

/**
* insertNonFull: Inserts key/value into subtree n (which must not be full).
* @param n *bpNode, key IndexKey, value string
**/
func (bt *BTree) insertNonFull(n *bpNode, key IndexKey, value string) {
	if n.leaf {
		i := leafSearch(n.keys, key)
		if i < len(n.keys) && n.keys[i].Compare(key) == 0 {
			if !slices.Contains(n.vals[i], value) {
				n.vals[i] = append(n.vals[i], value)
			}
			return
		}
		n.keys = append(n.keys, IndexKey{})
		copy(n.keys[i+1:], n.keys[i:])
		n.keys[i] = key
		n.vals = append(n.vals, nil)
		copy(n.vals[i+1:], n.vals[i:])
		n.vals[i] = []string{value}
		bt.size++
		return
	}

	i := childIdx(n.keys, key)
	if len(n.children[i].keys) == 2*bt.t-1 {
		bt.splitChild(n, i)
		if key.Compare(n.keys[i]) >= 0 {
			i++
		}
	}
	bt.insertNonFull(n.children[i], key, value)
}

/**
* insertSlice: Inserts all pks for a key in one shot (used by Init to avoid store writes).
* @param key IndexKey, pks []string
**/
func (bt *BTree) insertSlice(key IndexKey, pks []string) {
	if len(bt.root.keys) == 2*bt.t-1 {
		newRoot := &bpNode{children: []*bpNode{bt.root}}
		bt.splitChild(newRoot, 0)
		bt.root = newRoot
	}
	bt.insertSliceNonFull(bt.root, key, pks)
}

/**
* insertSliceNonFull: Inserts a slice of pks into subtree n (which must not be full).
* @param n *bpNode, key IndexKey, pks []string
**/
func (bt *BTree) insertSliceNonFull(n *bpNode, key IndexKey, pks []string) {
	if n.leaf {
		i := leafSearch(n.keys, key)
		if i < len(n.keys) && n.keys[i].Compare(key) == 0 {
			for _, pk := range pks {
				if !slices.Contains(n.vals[i], pk) {
					n.vals[i] = append(n.vals[i], pk)
				}
			}
			return
		}
		n.keys = append(n.keys, IndexKey{})
		copy(n.keys[i+1:], n.keys[i:])
		n.keys[i] = key
		n.vals = append(n.vals, nil)
		copy(n.vals[i+1:], n.vals[i:])
		n.vals[i] = append([]string{}, pks...)
		bt.size++
		return
	}
	i := childIdx(n.keys, key)
	if len(n.children[i].keys) == 2*bt.t-1 {
		bt.splitChild(n, i)
		if key.Compare(n.keys[i]) >= 0 {
			i++
		}
	}
	bt.insertSliceNonFull(n.children[i], key, pks)
}

/**
* splitChild: Splits n.children[ci] (which must be full) and promotes the separator to parent.
* @param parent *bpNode, ci int
**/
func (bt *BTree) splitChild(parent *bpNode, ci int) {
	t := bt.t
	child := parent.children[ci]

	if child.leaf {
		mid := t
		right := &bpNode{
			leaf: true,
			keys: append([]IndexKey{}, child.keys[mid:]...),
			vals: append([][]string{}, child.vals[mid:]...),
			next: child.next,
		}
		child.keys = child.keys[:mid]
		child.vals = child.vals[:mid]
		child.next = right

		bpInsertKey(parent, ci, right.keys[0])
		bpInsertChild(parent, ci+1, right)
	} else {
		mid := t - 1
		right := &bpNode{
			leaf:     false,
			keys:     append([]IndexKey{}, child.keys[mid+1:]...),
			children: append([]*bpNode{}, child.children[mid+1:]...),
		}
		sep := child.keys[mid]
		child.keys = child.keys[:mid]
		child.children = child.children[:mid+1]

		bpInsertKey(parent, ci, sep)
		bpInsertChild(parent, ci+1, right)
	}
}

/**
* bpInsertKey: Inserts key at position i in node n's key slice.
* @param n *bpNode, i int, key IndexKey
**/
func bpInsertKey(n *bpNode, i int, key IndexKey) {
	n.keys = append(n.keys, IndexKey{})
	copy(n.keys[i+1:], n.keys[i:])
	n.keys[i] = key
}

/**
* bpInsertChild: Inserts child at position i in node n's children slice.
* @param n *bpNode, i int, child *bpNode
**/
func bpInsertChild(n *bpNode, i int, child *bpNode) {
	n.children = append(n.children, nil)
	copy(n.children[i+1:], n.children[i:])
	n.children[i] = child
}

/**
* Delete: Removes value from the set at key. If the set becomes empty the key is removed.
* Returns (found, error).
* @param key IndexKey, value string
* @return bool, error
**/
func (bt *BTree) Delete(key IndexKey, value string) (bool, error) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i >= len(leaf.keys) || leaf.keys[i].Compare(key) != 0 {
		return false, nil
	}

	vals := leaf.vals[i]
	n := 0
	for _, v := range vals {
		if v != value {
			vals[n] = v
			n++
		}
	}
	if n == len(vals) {
		return false, nil
	}
	vals = vals[:n]

	if len(vals) > 0 {
		leaf.vals[i] = vals
		return true, bt.persistKey(key)
	}

	leaf.keys = append(leaf.keys[:i], leaf.keys[i+1:]...)
	leaf.vals = append(leaf.vals[:i], leaf.vals[i+1:]...)
	bt.size--

	if bt.st != nil {
		if _, err := bt.st.Delete(encodeKey(key)); err != nil {
			return true, err
		}
	}
	return true, nil
}

/**
* DeleteKey: Removes a key together with all its associated values.
* @param key IndexKey
* @return bool, error
**/
func (bt *BTree) DeleteKey(key IndexKey) (bool, error) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i >= len(leaf.keys) || leaf.keys[i].Compare(key) != 0 {
		return false, nil
	}

	leaf.keys = append(leaf.keys[:i], leaf.keys[i+1:]...)
	leaf.vals = append(leaf.vals[:i], leaf.vals[i+1:]...)
	bt.size--

	if bt.st != nil {
		if _, err := bt.st.Delete(encodeKey(key)); err != nil {
			return true, err
		}
	}
	return true, nil
}

/**
* Equal: Returns all values of keys equal to key.
* @param key IndexKey
* @return []string
**/
func (bt *BTree) Equal(key IndexKey) []string {
	result, ok := bt.Get(key)
	if !ok {
		return []string{}
	}
	return result
}

/**
* NotEqual: Returns all values of keys != key.
* @param key IndexKey
* @return []string
**/
func (bt *BTree) NotEqual(key IndexKey) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var result []string
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		for j := 0; j < len(leaf.keys); j++ {
			if leaf.keys[j].Compare(key) != 0 {
				result = append(result, leaf.vals[j]...)
			}
		}
	}
	return result
}

/**
* Between: Returns all values of keys in [from, to] inclusive.
* Pass zero (IndexKey{}) in from or to to indicate an open bound.
* @param from, to IndexKey, asc bool
* @return []string
**/
func (bt *BTree) Between(from, to IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var start *bpNode
	var startIdx int

	if from == zero {
		start = bt.leftmostLeaf()
		startIdx = 0
	} else {
		start = bt.findLeaf(from)
		startIdx = leafSearch(start.keys, from)
	}

	var result []string
	for leaf := start; leaf != nil; leaf = leaf.next {
		begin := 0
		if leaf == start {
			begin = startIdx
		}
		for j := begin; j < len(leaf.keys); j++ {
			if to != zero && leaf.keys[j].Compare(to) > 0 {
				if !asc {
					bpReverse(result)
				}
				return result
			}
			result = append(result, leaf.vals[j]...)
		}
	}

	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* More: Returns all values of keys strictly greater than key.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) More(key IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	start := bt.findLeaf(key)
	startIdx := leafSearch(start.keys, key)
	if startIdx < len(start.keys) && start.keys[startIdx].Compare(key) == 0 {
		startIdx++
	}

	var result []string
	for leaf := start; leaf != nil; leaf = leaf.next {
		begin := 0
		if leaf == start {
			begin = startIdx
		}
		for j := begin; j < len(leaf.keys); j++ {
			result = append(result, leaf.vals[j]...)
		}
	}
	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* MoreEq: Returns all values of keys >= key.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) MoreEq(key IndexKey, asc bool) []string {
	return bt.Between(key, zero, asc)
}

/**
* Less: Returns all values of keys strictly less than key.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) Less(key IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var result []string
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		for j := 0; j < len(leaf.keys); j++ {
			if leaf.keys[j].Compare(key) >= 0 {
				goto done
			}
			result = append(result, leaf.vals[j]...)
		}
	}
done:
	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* LessEq: Returns all values of keys <= key.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) LessEq(key IndexKey, asc bool) []string {
	return bt.Between(zero, key, asc)
}

/**
* Like: Returns all values of KtString keys matching the SQL LIKE pattern.
* % matches any sequence of characters, _ matches any single character.
* Non-string keys are skipped.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) Like(key IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	pattern := []rune(key.String())
	var result []string
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		for j := 0; j < len(leaf.keys); j++ {
			if leaf.keys[j].tp != KtString {
				continue
			}
			if matchLike([]rune(leaf.keys[j].str), pattern) {
				result = append(result, leaf.vals[j]...)
			}
		}
	}
	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* In: Returns all values of keys present in the given set.
* Duplicate primary keys across multiple index entries are deduplicated.
* @param keys []IndexKey, asc bool
* @return []string
**/
func (bt *BTree) In(keys []IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	seen := map[string]bool{}
	var result []string
	for _, key := range keys {
		leaf := bt.findLeaf(key)
		i := leafSearch(leaf.keys, key)
		if i < len(leaf.keys) && leaf.keys[i].Compare(key) == 0 {
			for _, v := range leaf.vals[i] {
				if !seen[v] {
					seen[v] = true
					result = append(result, v)
				}
			}
		}
	}
	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* NotIn: Returns all values of keys NOT present in the given set.
* Scans all leaves; results are deduplicated.
* @param keys []IndexKey, asc bool
* @return []string
**/
func (bt *BTree) NotIn(keys []IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	excluded := make(map[string]bool, len(keys))
	for _, key := range keys {
		excluded[key.String()] = true
	}

	seen := map[string]bool{}
	var result []string
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		for j := 0; j < len(leaf.keys); j++ {
			if excluded[leaf.keys[j].String()] {
				continue
			}
			for _, v := range leaf.vals[j] {
				if !seen[v] {
					seen[v] = true
					result = append(result, v)
				}
			}
		}
	}
	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* Is: Returns all values of keys that strictly match key (same KeyType and value).
* Unlike Equal, cross-type string fallback is not used — string must match string,
* number must match number, etc.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) Is(key IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i < len(leaf.keys) && leaf.keys[i].tp == key.tp && leaf.keys[i].Compare(key) == 0 {
		out := make([]string, len(leaf.vals[i]))
		copy(out, leaf.vals[i])
		if !asc {
			bpReverse(out)
		}
		return out
	}
	return []string{}
}

/**
* IsNot: Returns all values of keys that do NOT strictly match key.
* Complement of Is: excludes keys where both KeyType and value are equal.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) IsNot(key IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var result []string
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		for j := 0; j < len(leaf.keys); j++ {
			if leaf.keys[j].tp == key.tp && leaf.keys[j].Compare(key) == 0 {
				continue
			}
			result = append(result, leaf.vals[j]...)
		}
	}
	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* Null: Returns all values of keys that represent a null field value.
* The null sentinel is KeyString("<nil>"), produced by KeyFromAny(nil).
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) Null(key IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i < len(leaf.keys) && leaf.keys[i].Compare(key) == 0 {
		out := make([]string, len(leaf.vals[i]))
		copy(out, leaf.vals[i])
		if !asc {
			bpReverse(out)
		}
		return out
	}
	return []string{}
}

/**
* NotNull: Returns all values of keys that do NOT represent a null field value.
* Complement of Null: scans all leaves and skips the null sentinel key.
* @param key IndexKey, asc bool
* @return []string
**/
func (bt *BTree) NotNull(key IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var result []string
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		for j := 0; j < len(leaf.keys); j++ {
			if leaf.keys[j].Compare(key) == 0 {
				continue
			}
			result = append(result, leaf.vals[j]...)
		}
	}
	if !asc {
		bpReverse(result)
	}
	return result
}

/**
* NotBetween: Returns all values of keys outside the range [from, to] exclusive.
* Complement of Between: collects keys < from and keys > to in a single leaf scan.
* @param from, to IndexKey, asc bool
* @return []string
**/
func (bt *BTree) NotBetween(from, to IndexKey, asc bool) []string {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var result []string
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		for j := 0; j < len(leaf.keys); j++ {
			k := leaf.keys[j]
			if k.Compare(from) < 0 || k.Compare(to) > 0 {
				result = append(result, leaf.vals[j]...)
			}
		}
	}
	if !asc {
		bpReverse(result)
	}
	return result
}

// matchLike reports whether s matches pattern p using SQL LIKE semantics.
func matchLike(s, p []rune) bool {
	var match func(si, pi int) bool
	match = func(si, pi int) bool {
		if pi == len(p) {
			return si == len(s)
		}
		if p[pi] == '%' {
			for i := si; i <= len(s); i++ {
				if match(i, pi+1) {
					return true
				}
			}
			return false
		}
		if si == len(s) {
			return false
		}
		if p[pi] == '_' || p[pi] == s[si] {
			return match(si+1, pi+1)
		}
		return false
	}
	return match(0, 0)
}

/**
* Keys: Returns distinct keys with pagination. limit=0 returns all.
* @param asc bool, offset, limit int
* @return []IndexKey
**/
func (bt *BTree) Keys(asc bool, offset, limit int) []IndexKey {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var all []IndexKey
	for leaf := bt.leftmostLeaf(); leaf != nil; leaf = leaf.next {
		all = append(all, leaf.keys...)
	}

	if !asc {
		bpReverseKeys(all)
	}
	if offset >= len(all) {
		return nil
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all
}

/**
* leftmostLeaf: Returns the leftmost (smallest-key) leaf node.
* @return *bpNode
**/
func (bt *BTree) leftmostLeaf() *bpNode {
	n := bt.root
	for !n.leaf {
		n = n.children[0]
	}
	return n
}

/**
* ApplyCondition: Applies a condition to all keys and returns the matching ones.
* @param condition *et.Condition
* @return []string
**/
func (bt *BTree) ApplyCondition(condition *et.Condition) []string {
	value := condition.Value.Value
	switch condition.Operator {
	case et.EQ:
		key := KeyFromAny(value)
		return bt.Equal(key)
	case et.NEG:
		key := KeyFromAny(value)
		return bt.NotEqual(key)
	case et.LESS:
		key := KeyFromAny(value)
		return bt.Less(key, true)
	case et.LESS_EQ:
		key := KeyFromAny(value)
		return bt.LessEq(key, true)
	case et.MORE:
		key := KeyFromAny(value)
		return bt.More(key, true)
	case et.MORE_EQ:
		key := KeyFromAny(value)
		return bt.MoreEq(key, true)
	case et.LIKE:
		key := KeyFromAny(value)
		return bt.Like(key, true)
	case et.IN:
		if vals, ok := value.([]any); ok {
			keys := make([]IndexKey, len(vals))
			for i, v := range vals {
				keys[i] = KeyFromAny(v)
			}
			return bt.In(keys, true)
		}
		return bt.In([]IndexKey{KeyFromAny(value)}, true)
	case et.NOT_IN:
		if vals, ok := value.([]any); ok {
			keys := make([]IndexKey, len(vals))
			for i, v := range vals {
				keys[i] = KeyFromAny(v)
			}
			return bt.NotIn(keys, true)
		}
		return bt.NotIn([]IndexKey{KeyFromAny(value)}, true)
	case et.IS:
		key := KeyFromAny(value)
		return bt.Is(key, true)
	case et.IS_NOT:
		key := KeyFromAny(value)
		return bt.IsNot(key, true)
	case et.NULL:
		key := KeyFromAny(value)
		return bt.Null(key, true)
	case et.NOT_NULL:
		key := KeyFromAny(value)
		return bt.NotNull(key, true)
	case et.BETWEEN:
		btValues, ok := value.(et.BetweenValue)
		if ok {
			keyMin := KeyFromAny(btValues.Min)
			keyMax := KeyFromAny(btValues.Max)
			return bt.Between(keyMin, keyMax, true)
		}
	case et.NOT_BETWEEN:
		btValues, ok := value.(et.BetweenValue)
		if ok {
			keyMin := KeyFromAny(btValues.Min)
			keyMax := KeyFromAny(btValues.Max)
			return bt.NotBetween(keyMin, keyMax, true)
		}
	}

	return []string{}
}

/**
* ApplyConditions: Applies a set of conditions to all keys and returns the matching ones,
* combining each condition's result with the accumulated result according to its Connector
* (et.And intersects, et.Or unions; the first condition's connector is ignored).
* @param conditions []*et.Condition
* @return []string
**/
func (bt *BTree) ApplyConditions(conditions []*et.Condition) []string {
	if len(conditions) == 0 {
		return []string{}
	}

	result := bt.ApplyCondition(conditions[0])
	for _, condition := range conditions[1:] {
		keys := bt.ApplyCondition(condition)
		if condition.Connector == et.Or {
			result = bpUnion(result, keys)
		} else {
			result = bpIntersect(result, keys)
		}
	}

	return result
}

/**
* bpIntersect: Returns the values present in both a and b, preserving a's order.
* @param a, b []string
* @return []string
**/
func bpIntersect(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, v := range b {
		set[v] = true
	}

	var result []string
	for _, v := range a {
		if set[v] {
			result = append(result, v)
		}
	}
	return result
}

/**
* bpUnion: Returns the deduplicated values present in a or b, preserving order (a first).
* @param a, b []string
* @return []string
**/
func bpUnion(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var result []string
	for _, v := range a {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	for _, v := range b {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

/**
* bpReverse: Reverses a string slice in place.
* @param s []string
**/
func bpReverse(s []string) {
	for l, r := 0, len(s)-1; l < r; l, r = l+1, r-1 {
		s[l], s[r] = s[r], s[l]
	}
}

/**
* bpReverseKeys: Reverses an IndexKey slice in place.
* @param s []IndexKey
**/
func bpReverseKeys(s []IndexKey) {
	for l, r := 0, len(s)-1; l < r; l, r = l+1, r-1 {
		s[l], s[r] = s[r], s[l]
	}
}
