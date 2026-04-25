package catalog

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"sync"
	"time"
)

const bpDegree = 32 // t: cada nodo tiene entre t-1 y 2t-1 keys

// KeyType identifica el tipo de dato de un IndexKey.
type KeyType uint8

const (
	KtString   KeyType = 1
	KtInt      KeyType = 2
	KtFloat    KeyType = 3
	KtBool     KeyType = 4
	KtDateTime KeyType = 5
)

// IndexKey es una clave tipada para el B+ tree.
// Garantiza orden correcto por tipo: numérico para int/float/datetime/bool,
// lexicográfico para strings.
type IndexKey struct {
	tp  KeyType
	str string  // KtString
	num float64 // KtInt (bits de int64), KtFloat, KtBool (0/1), KtDateTime (UnixNano)
}

func KeyString(v string) IndexKey { return IndexKey{tp: KtString, str: v} }
func KeyInt(v int64) IndexKey     { return IndexKey{tp: KtInt, num: float64(v)} }
func KeyFloat(v float64) IndexKey { return IndexKey{tp: KtFloat, num: v} }
func KeyDateTime(v time.Time) IndexKey {
	return IndexKey{tp: KtDateTime, num: float64(v.UnixNano())}
}
func KeyBool(v bool) IndexKey {
	if v {
		return IndexKey{tp: KtBool, num: 1}
	}
	return IndexKey{tp: KtBool, num: 0}
}

/**
* KeyFromAny construye un IndexKey a partir de cualquier valor JSON.
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
* Compare retorna -1, 0 o 1.
* @param b IndexKey
* @return int
* Tipos distintos se comparan por su representación string (fallback seguro).
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
	default: // KtInt, KtFloat, KtBool, KtDateTime — todos numéricos
		if a.num < b.num {
			return -1
		}
		if a.num > b.num {
			return 1
		}
		return 0
	}
}

// String retorna una representación legible de la key.
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

// zero es el valor centinela que indica "sin límite" en Range.
var zero IndexKey

// --- B+ tree ---

// bpNode es un nodo interno o hoja del B+ tree.
type bpNode struct {
	keys     []IndexKey // keys ordenadas
	children []*bpNode  // nodos internos: len(keys)+1 hijos; hojas: nil
	vals     [][]string // hojas: conjunto de primary-keys por key; internos: nil
	next     *bpNode    // lista enlazada de hojas (ascendente)
	leaf     bool
}

// BTree es un B+ tree thread-safe: IndexKey → conjunto de string values (primary keys).
type BTree struct {
	root *bpNode
	t    int // grado mínimo
	size int // número de keys distintas
	mu   sync.RWMutex
}

func NewBTree() *BTree {
	return &BTree{
		root: &bpNode{leaf: true},
		t:    bpDegree,
	}
}

/**
* Len retorna el número de keys distintas almacenadas.
* @return int
**/
func (bt *BTree) Len() int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.size
}

/**
* childIdx retorna el índice del hijo a seguir en un nodo interno (upper bound).
* @param keys []IndexKey, key IndexKey
* @return int
**/
func childIdx(keys []IndexKey, key IndexKey) int {
	return sort.Search(len(keys), func(i int) bool { return key.Compare(keys[i]) < 0 })
}

/**
* leafSearch retorna el lower bound de key en las keys ordenadas de una hoja.
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
* findLeaf navega desde la raíz hasta la hoja que debe contener key.
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
* Get retorna los values asociados a key. ok=false si key no existe.
* @param key IndexKey
* @return ([]string, bool)
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
* Insert agrega value al conjunto almacenado en key. No inserta duplicados.
* @param key IndexKey, value string
**/
func (bt *BTree) Insert(key IndexKey, value string) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	if len(bt.root.keys) == 2*bt.t-1 {
		newRoot := &bpNode{children: []*bpNode{bt.root}}
		bt.splitChild(newRoot, 0)
		bt.root = newRoot
	}
	bt.insertNonFull(bt.root, key, value)
}

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

// splitChild divide n.children[ci] (que debe estar lleno) y sube el separador a parent.
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

func bpInsertKey(n *bpNode, i int, key IndexKey) {
	n.keys = append(n.keys, IndexKey{})
	copy(n.keys[i+1:], n.keys[i:])
	n.keys[i] = key
}

func bpInsertChild(n *bpNode, i int, child *bpNode) {
	n.children = append(n.children, nil)
	copy(n.children[i+1:], n.children[i:])
	n.children[i] = child
}

/**
* Delete elimina value del conjunto en key.
* Si el conjunto queda vacío la key se elimina de la hoja.
* @param key IndexKey, value string
* @return bool
**/
func (bt *BTree) Delete(key IndexKey, value string) bool {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i >= len(leaf.keys) || leaf.keys[i].Compare(key) != 0 {
		return false
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
		return false
	}
	vals = vals[:n]

	if len(vals) > 0 {
		leaf.vals[i] = vals
		return true
	}

	leaf.keys = append(leaf.keys[:i], leaf.keys[i+1:]...)
	leaf.vals = append(leaf.vals[:i], leaf.vals[i+1:]...)
	bt.size--
	return true
}

/**
* DeleteKey elimina una key completa con todos sus values.
* @param key IndexKey
* @return bool
**/
func (bt *BTree) DeleteKey(key IndexKey) bool {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	leaf := bt.findLeaf(key)
	i := leafSearch(leaf.keys, key)
	if i >= len(leaf.keys) || leaf.keys[i].Compare(key) != 0 {
		return false
	}

	leaf.keys = append(leaf.keys[:i], leaf.keys[i+1:]...)
	leaf.vals = append(leaf.vals[:i], leaf.vals[i+1:]...)
	bt.size--
	return true
}

/**
* Range retorna todos los values de keys en [from, to] inclusive.
* Pasar zero (IndexKey{}) en from o to indica rango abierto.
* @param from IndexKey, to IndexKey, asc bool
* @return []string
**/
func (bt *BTree) Range(from, to IndexKey, asc bool) []string {
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
* Keys retorna las keys distintas con paginación. limit=0 retorna todo.
* @param asc bool, offset int, limit int
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

func (bt *BTree) leftmostLeaf() *bpNode {
	n := bt.root
	for !n.leaf {
		n = n.children[0]
	}
	return n
}

func bpReverse(s []string) {
	for l, r := 0, len(s)-1; l < r; l, r = l+1, r-1 {
		s[l], s[r] = s[r], s[l]
	}
}

func bpReverseKeys(s []IndexKey) {
	for l, r := 0, len(s)-1; l < r; l, r = l+1, r-1 {
		s[l], s[r] = s[r], s[l]
	}
}
