package jdb

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/store"
)

const bpDegree = 32 // minimum degree: each node holds between t-1 and 2t-1 keys

/**
* KeyType identifies the data type of an IndexKey.
**/
type KeyType uint8

const (
	KtString   KeyType = 1
	KtInt      KeyType = 2
	KtFloat    KeyType = 3
	KtBool     KeyType = 4
	KtDateTime KeyType = 5
)

/**
* IndexKey is a typed key for the B+ tree.
* Guarantees correct ordering per type: numeric for int/float/datetime/bool,
* lexicographic for strings.
**/
type IndexKey struct {
	tp  KeyType
	str string  // KtString
	num float64 // KtInt (int64 bits), KtFloat, KtBool (0/1), KtDateTime (UnixNano)
}

/**
* KeyString: Creates an IndexKey from a string value.
* @param v string
* @return IndexKey
**/
func KeyString(v string) IndexKey {
	return IndexKey{tp: KtString, str: v}
}

/**
* KeyInt: Creates an IndexKey from an int64 value.
* @param v int64
* @return IndexKey
**/
func KeyInt(v int64) IndexKey {
	return IndexKey{tp: KtInt, num: float64(v)}
}

/**
* KeyFloat: Creates an IndexKey from a float64 value.
* @param v float64
* @return IndexKey
**/
func KeyFloat(v float64) IndexKey {
	return IndexKey{tp: KtFloat, num: v}
}

/**
* KeyDateTime: Creates an IndexKey from a time.Time value.
* @param v time.Time
* @return IndexKey
**/
func KeyDateTime(v time.Time) IndexKey {
	return IndexKey{tp: KtDateTime, num: float64(v.UnixNano())}
}

/**
* KeyBool: Creates an IndexKey from a bool value.
* @param v bool
* @return IndexKey
**/
func KeyBool(v bool) IndexKey {
	if v {
		return IndexKey{tp: KtBool, num: 1}
	}
	return IndexKey{tp: KtBool, num: 0}
}

/**
* KeyFromAny: Builds an IndexKey from any JSON-unmarshalled value.
* Auto-detects type: whole float64 → KeyInt, RFC3339 string → KeyDateTime.
* @param v any
* @return IndexKey
**/
func KeyFromAny(v any) IndexKey {
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return KeyDateTime(t)
		}
		return KeyString(val)
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) {
			return KeyInt(int64(val))
		}
		return KeyFloat(val)
	case int:
		return KeyInt(int64(val))
	case int32:
		return KeyInt(int64(val))
	case int64:
		return KeyInt(val)
	case bool:
		return KeyBool(val)
	case time.Time:
		return KeyDateTime(val)
	default:
		return KeyString(fmt.Sprintf("%v", val))
	}
}

/**
* Compare: Returns -1, 0 or 1.
* Keys of different types compare by their string representation (safe fallback).
* @param b IndexKey
* @return int
**/
func (a IndexKey) Compare(b IndexKey) int {
	if a.tp != b.tp {
		as, bs := a.String(), b.String()
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	}
	switch a.tp {
	case KtString:
		if a.str < b.str {
			return -1
		}
		if a.str > b.str {
			return 1
		}
		return 0
	default: // KtInt, KtFloat, KtBool, KtDateTime — all numeric
		if a.num < b.num {
			return -1
		}
		if a.num > b.num {
			return 1
		}
		return 0
	}
}

/**
* String: Returns a human-readable representation of the key.
* @return string
**/
func (k IndexKey) String() string {
	switch k.tp {
	case KtString:
		return k.str
	case KtInt:
		return fmt.Sprintf("%d", int64(k.num))
	case KtFloat:
		return fmt.Sprintf("%g", k.num)
	case KtBool:
		if k.num != 0 {
			return "true"
		}
		return "false"
	case KtDateTime:
		return time.Unix(0, int64(k.num)).UTC().Format(time.RFC3339Nano)
	}
	return ""
}

/**
* encodeKey: Serializes an IndexKey as a FileStore key: "<prefix>/<value>".
* @param key IndexKey
* @return string
**/
func encodeKey(key IndexKey) string {
	switch key.tp {
	case KtString:
		return "s/" + key.str
	case KtInt:
		return "i/" + strconv.FormatInt(int64(key.num), 10)
	case KtFloat:
		return "f/" + strconv.FormatFloat(key.num, 'g', -1, 64)
	case KtBool:
		if key.num != 0 {
			return "b/1"
		}
		return "b/0"
	case KtDateTime:
		return "d/" + strconv.FormatInt(int64(key.num), 10)
	}
	return "s/"
}

/**
* decodeKey: Reconstructs an IndexKey from a FileStore key string.
* @param s string
* @return IndexKey, error
**/
func decodeKey(s string) (IndexKey, error) {
	if len(s) < 2 || s[1] != '/' {
		return IndexKey{}, fmt.Errorf("invalid store key: %q", s)
	}
	tp, val := s[0], s[2:]
	switch tp {
	case 's':
		return KeyString(val), nil
	case 'i':
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return IndexKey{}, err
		}
		return KeyInt(n), nil
	case 'f':
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return IndexKey{}, err
		}
		return KeyFloat(f), nil
	case 'b':
		return KeyBool(val == "1"), nil
	case 'd':
		ns, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return IndexKey{}, err
		}
		return KeyDateTime(time.Unix(0, ns)), nil
	}
	return IndexKey{}, fmt.Errorf("unknown key type prefix: %c", tp)
}

// zero is the sentinel value that means "no bound" in Between/MoreEq/LessEq.
var zero IndexKey

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
func OpenBTree(path, name string) (*BTree, error) {
	st, err := store.Open(path, "btidx_"+name, store.ReadWrite)
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
	}, true, 0, 0, 1)
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
	return bt.st.Put(encodeKey(key), leaf.vals[i])
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
* @return []string, bool
**/
func (bt *BTree) Equal(key IndexKey) ([]string, bool) {
	return bt.Get(key)
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
	var result []string

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
