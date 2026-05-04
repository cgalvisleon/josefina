package jdb

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type JoinType int

const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
	FullJoin
)

/**
* String
* @return string
**/
func (j JoinType) String() string {
	switch j {
	case InnerJoin:
		return "inner"
	case LeftJoin:
		return "left"
	case RightJoin:
		return "right"
	case FullJoin:
		return "full"
	default:
		return ""
	}
}

type Join struct {
	To   *Model            `json:"to"`
	Keys map[string]string `json:"keys"`
	Type JoinType          `json:"type"`
}

type OrderField struct {
	Field string
	Asc   bool
}

/**
* Where
**/
type Where struct {
	db         *DB                 `json:"-"`
	from       *Model              `json:"-"`
	selects    []string            `json:"-"`
	hidden     []string            `json:"-"`
	keys       map[string][]string `json:"-"`
	orderBy    []OrderField        `json:"-"`
	joins      []Join              `json:"-"`
	offset     int                 `json:"-"`
	limit      int                 `json:"-"`
	conditions []*et.Condition     `json:"-"`
	isDebug    bool                `json:"-"`
}

/**
* newWhere
* @param owner *Model
* @return *Where
**/
func newWhere(model *Model) *Where {
	result := &Where{
		db:         model.db,
		from:       model,
		selects:    make([]string, 0, 4),
		hidden:     make([]string, 0, 4),
		keys:       make(map[string][]string, 0),
		orderBy:    make([]OrderField, 0, 2),
		joins:      make([]Join, 0, 2),
		offset:     0,
		limit:      0,
		conditions: make([]*et.Condition, 0, 4),
	}

	return result
}

/**
* IsDebug: Returns the debug mode
* @return *Where
**/
func (s *Where) IsDebug() *Where {
	s.isDebug = true
	return s
}

/**
* ToJson
* @return et.Json
**/
func (s *Where) ToJson() []et.Json {
	result := []et.Json{}
	for _, condition := range s.conditions {
		result = append(result, condition.ToJson())
	}
	return result
}

/**
* From
* @param model *Model
* @return *Where
**/
func (s *Where) From(model *Model) *Where {
	if model == nil {
		return s
	}

	s.from = model
	return s
}

/**
* Add
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) Add(condition *et.Condition) *Where {
	if len(s.conditions) > 0 && condition.Connector == et.NaC {
		condition.Connector = et.And
	}

	s.conditions = append(s.conditions, condition)
	return s
}

/**
* Where
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) Where(condition *et.Condition) *Where {
	return s.Add(condition)
}

/**
* And
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) And(condition *et.Condition) *Where {
	condition.Connector = et.And
	return s.Add(condition)
}

/**
* Or
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) Or(condition *et.Condition) *Where {
	condition.Connector = et.Or
	return s.Add(condition)
}

/**
* Selects
* @param fields ...string
* @return *Where
**/
func (s *Where) Selects(fields ...string) *Where {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.selects = append(s.selects, field)
	}

	return s
}

/**
* Hidden
* @param fields ...string
* @return *Where
**/
func (s *Where) Hidden(fields ...string) *Where {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.hidden = append(s.hidden, field)
	}

	return s
}

/**
* Asc
* @param field string
* @return *Where
**/
func (s *Where) Asc(field string) *Where {
	s.orderBy = append(s.orderBy, OrderField{Field: strings.ToLower(field), Asc: true})
	return s
}

/**
* Desc
* @param field string
* @return *Where
**/
func (s *Where) Desc(field string) *Where {
	s.orderBy = append(s.orderBy, OrderField{Field: strings.ToLower(field), Asc: false})
	return s
}

/**
* Order
* @param field string
* @return bool
**/
func (s *Where) Order(field string) bool {
	field = strings.ToLower(field)
	for _, ob := range s.orderBy {
		if ob.Field == field {
			return ob.Asc
		}
	}
	return true
}

/**
* InnerJoin: Adds an inner join — only primary records with a match in to are returned.
* @param to *Model, keys map[string]string
* @return *Where
**/
func (s *Where) InnerJoin(to *Model, keys map[string]string) *Where {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: InnerJoin})
	return s
}

/**
* LeftJoin: Adds a left join — all primary records are returned; joined fields empty when no match.
* @param to *Model, keys map[string]string
* @return *Where
**/
func (s *Where) LeftJoin(to *Model, keys map[string]string) *Where {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: LeftJoin})
	return s
}

/**
* RightJoin: Adds a right join — all records from to are returned; primary fields empty when no match.
* @param to *Model, keys map[string]string
* @return *Where
**/
func (s *Where) RightJoin(to *Model, keys map[string]string) *Where {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: RightJoin})
	return s
}

/**
* FullJoin: Adds a full join — all records from both models, matched where possible.
* @param to *Model, keys map[string]string
* @return *Where
**/
func (s *Where) FullJoin(to *Model, keys map[string]string) *Where {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: FullJoin})
	return s
}

/**
* Limit
* @param page int, rows int
* @return *Where
**/
func (s *Where) Limit(page int, rows int) *Where {
	offset := (page - 1) * rows
	s.offset = offset
	s.limit = rows
	return s
}

/**
* SetOffset: Sets raw offset and row limit without page arithmetic.
* @param offset int, rows int
* @return *Where
**/
func (s *Where) SetOffset(offset int, rows int) *Where {
	s.offset = offset
	s.limit = rows
	return s
}

/**
* AllTx: Executes the WHERE query and returns all matching records.
* Execution order: resolve sub-queries → collect via index or full scan →
* apply joins → sort → offset/limit.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Where) AllTx(tx *Tx) (et.Items, error) {
	if s.from == nil {
		return et.Items{}, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	tx, _ = GetTx(s.db, tx)
	model := s.from

	// Resolve sub-Where condition values before any index access.
	for _, con := range s.conditions {
		switch v := con.Value.(type) {
		case *Where:
			items, err := v.AllTx(tx)
			if err != nil {
				return et.Items{}, err
			}
			con.Value = items
		case Where:
			items, err := v.AllTx(tx)
			if err != nil {
				return et.Items{}, err
			}
			con.Value = items
		}
	}

	// needFullScan: joins require collecting everything before applying join logic
	// (RightJoin/FullJoin add records from j.To not present in primary results).
	// Custom order fields require sorting the full result before offset/limit.
	hasCustomOrder := false
	for _, ob := range s.orderBy {
		if ob.Field != INDEX {
			hasCustomOrder = true
			break
		}
	}
	hasJoin := len(s.joins) > 0
	needFullScan := hasCustomOrder || hasJoin

	fetchOffset := s.offset
	fetchLimit := s.limit
	if needFullScan {
		fetchOffset = 0
		fetchLimit = 0
	}

	combinedHidden := append(model.Hidden[:len(model.Hidden):len(model.Hidden)], s.hidden...)
	keys := make(map[string]int, s.limit)
	result := make([]et.Json, 0, s.limit)

	addResult := func(idx string, item et.Json) bool {
		if len(s.selects) == 0 {
			item = item.Hidden(combinedHidden)
		} else {
			item = item.Select(s.selects)
			item = item.Hidden(s.hidden)
		}
		if pos, exists := keys[idx]; exists {
			result[pos] = item
			return true
		}
		n := len(result)
		keys[idx] = n
		result = append(result, item)
		n++
		if needFullScan {
			return true
		}
		return s.limit == 0 || n < s.limit
	}

	cache := tx.Items(model)

	if len(s.conditions) == 0 {
		// No conditions: full iteration in index order.
		asc := s.Order(INDEX)
		err := model.ForEach(func(idx string, item et.Json) (bool, error) {
			return addResult(idx, item), nil
		}, asc, fetchOffset, fetchLimit)
		if err != nil {
			return et.Items{}, err
		}
		if s.limit == 0 || len(result) < s.limit || needFullScan {
			for _, item := range cache {
				idx := item.Str(INDEX)
				if idx == "" {
					continue
				}
				if !addResult(idx, item) {
					break
				}
			}
		}
	} else {
		// Determine whether btree indexes cover every "entry-point" condition.
		// Entry-points are NaC (first condition) and OR conditions — a record can
		// reach the result only through them. If any entry-point is not indexed we
		// would silently miss records that satisfy only that condition, so we must
		// fall back to a full scan evaluated against ALL conditions at once.
		canUseBtree := true
		anyBtree := false

		for _, con := range s.conditions {
			_, hasBt := model.getBtree(con.Field)
			if !hasBt {
				if con.Connector != et.And {
					canUseBtree = false
				}
				continue
			}
			anyBtree = true
		}

		if anyBtree && canUseBtree {
			// Build candidate set:
			//   AND / NaC conditions with btree → intersect (narrows the set).
			//   OR conditions with btree         → union (broadens the set).
			// Non-indexed AND conditions are handled by et.Evaluate below.
			var andCandidates []string
			var orCandidates []string
			andInit := false

			for _, con := range s.conditions {
				bt, exists := model.getBtree(con.Field)
				if !exists {
					continue
				}
				idxs := bt.ApplyCondition(con)

				if con.Connector == et.Or {
					orCandidates = append(orCandidates, idxs...)
				} else {
					if !andInit {
						andCandidates = idxs
						andInit = true
					} else {
						set := make(map[string]struct{}, len(idxs))
						for _, i := range idxs {
							set[i] = struct{}{}
						}
						n := 0
						for _, i := range andCandidates {
							if _, ok := set[i]; ok {
								andCandidates[n] = i
								n++
							}
						}
						andCandidates = andCandidates[:n]
					}
				}
			}

			// Merge AND and OR candidates without duplicates.
			seen := make(map[string]struct{}, len(andCandidates)+len(orCandidates))
			candidates := make([]string, 0, len(andCandidates)+len(orCandidates))
			for _, idx := range andCandidates {
				if _, ok := seen[idx]; !ok {
					seen[idx] = struct{}{}
					candidates = append(candidates, idx)
				}
			}
			for _, idx := range orCandidates {
				if _, ok := seen[idx]; !ok {
					seen[idx] = struct{}{}
					candidates = append(candidates, idx)
				}
			}

			// Fetch each candidate and filter through ALL conditions.
			for _, idx := range candidates {
				item, exists, err := model.Current(idx)
				if err != nil {
					return et.Items{}, err
				}
				if !exists {
					continue
				}
				if et.Evaluate(item, s.conditions) {
					addResult(idx, item)
				}
			}

			// Check pending transaction records against ALL conditions.
			for _, item := range cache {
				if et.Evaluate(item, s.conditions) {
					idx := item.Str(INDEX)
					if idx != "" {
						addResult(idx, item)
					}
				}
			}
		} else {
			// No safe btree path: single full scan evaluating ALL conditions per record.
			// One pass — no repeated scans regardless of condition count.
			err := model.ForEach(func(idx string, item et.Json) (bool, error) {
				if et.Evaluate(item, s.conditions) {
					return addResult(idx, item), nil
				}
				return true, nil
			}, true, fetchOffset, fetchLimit)
			if err != nil {
				return et.Items{}, err
			}

			for _, item := range cache {
				if et.Evaluate(item, s.conditions) {
					idx := item.Str(INDEX)
					if idx != "" {
						addResult(idx, item)
					}
				}
			}
		}
	}

	// Apply joins before sort so all records participate in ordering.
	if hasJoin {
		var err error
		result, err = applyJoins(result, s.joins, s.from, tx)
		if err != nil {
			return et.Items{}, err
		}
	}

	if hasCustomOrder {
		sortKeys := make([][]any, len(result))
		for i, item := range result {
			values := make([]any, len(s.orderBy))
			for j, ob := range s.orderBy {
				values[j] = item[ob.Field]
			}
			sortKeys[i] = values
		}
		sort.Slice(result, func(i, j int) bool {
			for f, ob := range s.orderBy {
				cmp := compareJsonValue(sortKeys[i][f], sortKeys[j][f])
				if cmp == 0 {
					continue
				}
				return cmp < 0 == ob.Asc
			}
			return false
		})
	}

	if needFullScan {
		if s.offset > 0 {
			if s.offset >= len(result) {
				return et.NewItems([]et.Json{}), nil
			}
			result = result[s.offset:]
		}
		if s.limit > 0 && len(result) > s.limit {
			result = result[:s.limit]
		}
	}

	return et.NewItems(result), nil
}

/**
* All: Executes the WHERE query and returns all matching records.
* @return et.Items, error
**/
func (s *Where) All() (et.Items, error) {
	var tx *Tx
	return s.AllTx(tx)
}

/**
* One
* @param tx *Tx, idx int
* @return et.Json, error
**/
func (s *Where) OneTx(tx *Tx, idx int) (et.Item, error) {
	items, err := s.AllTx(tx)
	if err != nil {
		return et.Item{}, err
	}

	return items.One(idx)
}

/**
* One: Executes the WHERE query and returns the item at the specified index.
* @param idx int
* @return et.Item, error
**/
func (s *Where) One(idx int) (et.Item, error) {
	var tx *Tx
	return s.OneTx(tx, idx)
}

/**
* First
* @param tx *Tx
* @return et.Json, error
**/
func (s *Where) FirstTx(tx *Tx) (et.Item, error) {
	return s.OneTx(tx, 0)
}

/**
* First: Executes the WHERE query and returns the first item.
* @return et.Item, error
**/
func (s *Where) First() (et.Item, error) {
	var tx *Tx
	return s.FirstTx(tx)
}

/**
* LastTx
* @param tx *Tx
* @return et.Item, error
**/
func (s *Where) LastTx(tx *Tx) (et.Item, error) {
	return s.OneTx(tx, -1)
}

/**
* Last: Executes the WHERE query and returns the last item.
* @return et.Item, error
**/
func (s *Where) Last() (et.Item, error) {
	var tx *Tx
	return s.LastTx(tx)
}

/**
* From
* @param model *Model
* @return *Where
**/
func From(model *Model) *Where {
	return newWhere(model)
}

/**
* Eq
* @param field string, value interface{}
* @return *et.Condition
**/
func Eq(field string, value interface{}) *et.Condition {
	return et.Eq(field, value)
}

/**
* Neg
* @param field string, value interface{}
* @return *et.Condition
**/
func Neg(field string, value interface{}) *et.Condition {
	return et.Neg(field, value)
}

/**
* Less
* @param field string, value interface{}
* @return *et.Condition
**/
func Less(field string, value interface{}) *et.Condition {
	return et.Less(field, value)
}

/**
* LessEq
* @param field string, value interface{}
* @return *et.Condition
**/
func LessEq(field string, value interface{}) *et.Condition {
	return et.LessEq(field, value)
}

/**
* More
* @param field string, value interface{}
* @return *et.Condition
**/
func More(field string, value interface{}) *et.Condition {
	return et.More(field, value)
}

/**
* MoreEq
* @param field string, value interface{}
* @return *et.Condition
**/
func MoreEq(field string, value interface{}) *et.Condition {
	return et.MoreEq(field, value)
}

/**
* Like
* @param field string, value interface{}
* @return *et.Condition
**/
func Like(field string, value interface{}) *et.Condition {
	return et.Like(field, value)
}

/**
* In
* @param field string, value []interface{}
* @return *et.Condition
**/
func In(field string, value []interface{}) *et.Condition {
	return et.In(field, value)
}

/**
* NotIn
* @param field string, value []interface{}
* @return *et.Condition
**/
func NotIn(field string, value []interface{}) *et.Condition {
	return et.NotIn(field, value)
}

/**
* Is
* @param field string, value interface{}
* @return *et.Condition
**/
func Is(field string, value interface{}) *et.Condition {
	return et.Is(field, value)
}

/**
* IsNot
* @param field string, value interface{}
* @return *et.Condition
**/
func IsNot(field string, value interface{}) *et.Condition {
	return et.IsNot(field, value)
}

/**
* Null
* @param field string
* @return *et.Condition
**/
func Null(field string) *et.Condition {
	return et.Null(field)
}

/**
* NotNull
* @param field string
* @return *et.Condition
**/
func NotNull(field string) *et.Condition {
	return et.NotNull(field)
}

/**
* Between
* @param field string, min any, max any
* @return *et.Condition
**/
func Between(field string, min, max any) *et.Condition {
	return et.Between(field, min, max)
}

/**
* NotBetween
* @param field string, min any, max any
* @return *et.Condition
**/
func NotBetween(field string, min, max any) *et.Condition {
	return et.NotBetween(field, min, max)
}

/**
* Evaluate
* @param item et.Json, conditions []*et.Condition
* @return bool
**/
func Evaluate(item et.Json, conditions []*et.Condition) bool {
	return et.Evaluate(item, conditions)
}

/**
* joinLookup: Returns records from j.To that match the given primary item via j.Keys.
* Uses the btree index on the join field when available; falls back to full scan.
* Only the first key pair is used for the btree lookup; subsequent pairs are checked
* as filters on the candidates (composite key support).
* @param item et.Json, j Join, tx *Tx
* @return []et.Json, error
**/
func joinLookup(item et.Json, j Join, tx *Tx) ([]et.Json, error) {
	var candidateIdx []string
	first := true

	for fromField, toField := range j.Keys {
		fromVal := item[fromField]
		cond := et.Eq(toField, fromVal)

		bt, exists := j.To.getBtree(toField)
		if exists {
			idxs := bt.ApplyCondition(cond)
			if first {
				candidateIdx = idxs
				first = false
			} else {
				set := make(map[string]struct{}, len(idxs))
				for _, idx := range idxs {
					set[idx] = struct{}{}
				}
				filtered := candidateIdx[:0]
				for _, idx := range candidateIdx {
					if _, ok := set[idx]; ok {
						filtered = append(filtered, idx)
					}
				}
				candidateIdx = filtered
			}
		} else {
			var scanned []string
			j.To.ForEach(func(idx string, toItem et.Json) (bool, error) {
				if toItem.ApplyCondition(cond) {
					scanned = append(scanned, idx)
				}
				return true, nil
			}, true, 0, 0)
			if first {
				candidateIdx = scanned
				first = false
			} else {
				set := make(map[string]struct{}, len(scanned))
				for _, idx := range scanned {
					set[idx] = struct{}{}
				}
				filtered := candidateIdx[:0]
				for _, idx := range candidateIdx {
					if _, ok := set[idx]; ok {
						filtered = append(filtered, idx)
					}
				}
				candidateIdx = filtered
			}
		}

		if len(candidateIdx) == 0 {
			break
		}
	}

	matched := make([]et.Json, 0, len(candidateIdx))
	for _, idx := range candidateIdx {
		toItem, exists, err := j.To.Current(idx)
		if err != nil {
			return nil, err
		}
		if exists {
			matched = append(matched, toItem)
		}
	}

	// Also check transaction cache for pending writes to j.To.
	for fromField, toField := range j.Keys {
		cond := et.Eq(toField, item[fromField])
		for _, toItem := range tx.Items(j.To) {
			if toItem.ApplyCondition(cond) {
				matched = append(matched, toItem)
			}
		}
		break
	}

	return matched, nil
}

/**
* mergeJoinFields: Copies fields from src into a new Json merging with dst.
* Fields from src are prefixed with modelName to avoid collisions.
* @param dst et.Json, src et.Json, modelName string
* @return et.Json
**/
func mergeJoinFields(dst, src et.Json, modelName string) et.Json {
	result := make(et.Json, len(dst)+len(src))
	for k, v := range dst {
		result[k] = v
	}
	prefix := modelName + "."
	for k, v := range src {
		result[prefix+k] = v
	}
	return result
}

/**
* applyJoins: Applies all joins in sequence to the result set.
* @param items []et.Json, joins []Join, from *Model, tx *Tx
* @return []et.Json, error
**/
func applyJoins(items []et.Json, joins []Join, from *Model, tx *Tx) ([]et.Json, error) {
	for _, j := range joins {
		var err error
		items, err = applySingleJoin(items, j, from, tx)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

/**
* applySingleJoin: Applies one join to the item set according to its type.
* InnerJoin  — keeps only primary records that have a match in j.To.
* LeftJoin   — keeps all primary records; merges right fields when matched.
* RightJoin  — keeps all j.To records; merges primary fields when matched.
* FullJoin   — union of all primary and all j.To records, merged where matched.
* @param items []et.Json, j Join, from *Model, tx *Tx
* @return []et.Json, error
**/
func applySingleJoin(items []et.Json, j Join, from *Model, tx *Tx) ([]et.Json, error) {
	switch j.Type {
	case InnerJoin:
		result := make([]et.Json, 0, len(items))
		for _, item := range items {
			matched, err := joinLookup(item, j, tx)
			if err != nil {
				return nil, err
			}
			if len(matched) > 0 {
				result = append(result, mergeJoinFields(item, matched[0], j.To.Name))
			}
		}
		return result, nil

	case LeftJoin:
		result := make([]et.Json, 0, len(items))
		for _, item := range items {
			matched, err := joinLookup(item, j, tx)
			if err != nil {
				return nil, err
			}
			if len(matched) > 0 {
				result = append(result, mergeJoinFields(item, matched[0], j.To.Name))
			} else {
				result = append(result, item)
			}
		}
		return result, nil

	case RightJoin:
		matchedToIdx := make(map[string]struct{}, len(items))
		result := make([]et.Json, 0, len(items))
		for _, item := range items {
			matched, err := joinLookup(item, j, tx)
			if err != nil {
				return nil, err
			}
			if len(matched) > 0 {
				toItem := matched[0]
				// j.To fields at root; primary fields prefixed with from.Name.
				result = append(result, mergeJoinFields(toItem, item, from.Name))
				if idx := toItem.Str(INDEX); idx != "" {
					matchedToIdx[idx] = struct{}{}
				}
			}
		}
		err := j.To.ForEach(func(idx string, toItem et.Json) (bool, error) {
			if _, seen := matchedToIdx[idx]; !seen {
				result = append(result, toItem)
			}
			return true, nil
		}, true, 0, 0)
		if err != nil {
			return nil, err
		}
		return result, nil

	case FullJoin:
		matchedToIdx := make(map[string]struct{}, len(items))
		result := make([]et.Json, 0, len(items))
		for _, item := range items {
			matched, err := joinLookup(item, j, tx)
			if err != nil {
				return nil, err
			}
			if len(matched) > 0 {
				toItem := matched[0]
				// Primary fields at root; j.To fields prefixed with j.To.Name.
				result = append(result, mergeJoinFields(item, toItem, j.To.Name))
				if idx := toItem.Str(INDEX); idx != "" {
					matchedToIdx[idx] = struct{}{}
				}
			} else {
				result = append(result, item)
			}
		}
		err := j.To.ForEach(func(idx string, toItem et.Json) (bool, error) {
			if _, seen := matchedToIdx[idx]; !seen {
				result = append(result, toItem)
			}
			return true, nil
		}, true, 0, 0)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return items, nil
}

/**
* compareJsonValue: Compares two JSON values for ordering.
* JSON numbers unmarshal as float64; strings and bools are also handled.
* Returns -1, 0, or 1.
* @param a any, b any
* @return int
**/
func compareJsonValue(a, b any) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv)
		}
	case float64:
		if bv, ok := b.(float64); ok {
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
	case int:
		if bv, ok := b.(int); ok {
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
	case int64:
		if bv, ok := b.(int64); ok {
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
	case bool:
		if bv, ok := b.(bool); ok {
			if av == bv {
				return 0
			}
			if av {
				return 1
			}
			return -1
		}
	}
	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

type DFrom struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type DQuery struct {
	From    DFrom          `json:"from"`
	Join    []Join         `json:"join"`
	Where   []et.Condition `json:"where"`
	Selects []string       `json:"selects"`
	Hidden  []string       `json:"hidden"`
	OrderBy []OrderField   `json:"order_by"`
	Limit   int            `json:"limit"`
	Page    int            `json:"page"`
}
