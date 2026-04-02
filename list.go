package mind

import (
	"fmt"
	"strings"
	"sync"
)

// List is a fast in-memory store, which is extended by Indices for fast finding Items.
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() to modify data.
type List[T any, ID comparable] struct {
	list     FreeList[T]
	indexMap indexMap[T, ID]

	lock sync.RWMutex
}

// NewList create a new List
func NewList[T any]() *List[T, struct{}] {
	return &List[T, struct{}]{
		list:     NewFreeList[T](),
		indexMap: newIndexMap[T, struct{}](nil),
	}
}

// NewList create a new List with an ID-Index
func NewListWithID[T any, ID comparable](fieldIDGetFn func(*T) ID) *List[T, ID] {
	return &List[T, ID]{
		list:     NewFreeList[T](),
		indexMap: newIndexMap(newIDMapIndex(fieldIDGetFn)),
	}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - fieldGetFn: a function, which returns the value of an field
//   - Index: a impl of the Index interface
//
// Hint: empty field-name or the field-name ID are not allowed!
func (l *List[T, ID]) CreateIndex(fieldName string, index Index[T]) error {
	if fieldName == "" {
		return fmt.Errorf("empty fieldName is not allowed")
	}
	if strings.ToLower(fieldName) == IDIndexFieldName {
		return fmt.Errorf("ID is a reserved field name")
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	if _, exist := l.indexMap.index[fieldName]; exist {
		return fmt.Errorf("field-name: %s already exists", fieldName)
	}

	for idx, item := range l.list.Iter() {
		index.Set(&item, uint32(idx))
	}

	l.indexMap.index[fieldName] = index
	return nil
}

// RemoveIndex removed a the Index with the given field-name (what the name of the Index is)
// With the field-name: ID you can remove the ID-Index
func (l *List[T, ID]) RemoveIndex(fieldName string) {
	if fieldName == "" {
		return
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	if strings.ToUpper(fieldName) == "ID" {
		l.indexMap.idIndex = nil
		return
	}

	delete(l.indexMap.index, fieldName)
}

// Insert add the given Item to the list,
// There is NO check, for existing this Item in the list, it will ALWAYS inserting!
func (l *List[T, ID]) Insert(item T) int {
	l.lock.Lock()
	defer l.lock.Unlock()

	idx := l.list.Insert(item)
	l.indexMap.Set(&item, idx)

	return idx
}

// Update replaces an item and consistently updates all registered indexes.
func (l *List[T, ID]) Update(item T) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	id, idx, err := l.indexMap.getIDByItem(&item)
	if err != nil {
		return err
	}

	// overwrite the data in the main list
	oldItem, ok := l.list.Set(idx, item)
	if !ok {
		return ValueNotFoundError{id}
	}

	// update all indexes: re-index
	l.indexMap.ReIndex(&oldItem, &item, idx)
	return nil
}

// Remove an item by the given ID.
// This works ONLY, if an ID is defined (with calling: NewListWithID)
// errors:
// - wrong datatype
// - ID not found
// - no ID defined
func (l *List[T, ID]) Remove(id ID) (bool, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	idx, err := l.indexMap.getIndexByID(id)
	if err != nil {
		return false, err
	}

	return l.removeByIdxNoLock(idx), nil
}

//go:inline
func (l *List[T, ID]) removeByIdxNoLock(index int) (removed bool) {
	item, found := l.list.Get(index)
	if !found {
		return found
	}

	removed = l.list.Remove(index)
	l.indexMap.UnSet(&item, index)

	return removed
}

// Get returns an item by the given ID.
// This works ONLY, if an ID is defined (with calling: NewListWithID)
// errors:
// - wrong datatype
// - ID not found
// - no ID defined
func (l *List[T, ID]) Get(id ID) (T, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	idx, err := l.indexMap.getIndexByID(id)
	if err != nil {
		var null T
		return null, err
	}

	// not found should be possible
	item, _ := l.list.Get(idx)
	return item, nil
}

// ContainsID check, is this ID found in the list.
func (l *List[T, ID]) Contains(id ID) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	_, err := l.indexMap.getIndexByID(id)
	return err == nil
}

// Count the Items, which in this list exist
func (l *List[T, ID]) Count() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Count()
}

type QueryOption struct {
	WithOptimizer bool
	WithTracer    *Tracer
}

type Opion func(*QueryOption)

func NoOptimizer() Opion         { return func(o *QueryOption) { o.WithOptimizer = false } }
func WithTracer(t *Tracer) Opion { return func(o *QueryOption) { o.WithTracer = t } }

// QueryStr execute the given Query-string.
func (l *List[T, ID]) QueryStr(queryStr string, opts ...Opion) *QueryResult[T, ID] {
	ast, err := Parse(queryStr)
	if err != nil {
		var query Query
		return &QueryResult[T, ID]{list: l, query: query, err: err}
	}

	return l.Query(ast, opts...)
}

// Query execute the given Query.
func (l *List[T, ID]) Query(query Expr, opts ...Opion) *QueryResult[T, ID] {
	opt := QueryOption{WithOptimizer: true}
	for _, o := range opts {
		o(&opt)
	}

	if opt.WithOptimizer {
		query = query.Optimize()
	}

	return &QueryResult[T, ID]{list: l, query: query.Compile(opt.WithTracer)}
}

type QueryResult[T any, ID comparable] struct {
	list  *List[T, ID]
	query Query
	err   error
}

// Count of the Query result
func (qr *QueryResult[T, ID]) Count() (int, error) {
	if qr.err != nil {
		return 0, qr.err
	}

	qr.list.lock.RLock()
	defer qr.list.lock.RUnlock()

	bs, _, err := qr.query(qr.list.indexMap.FilterByName, qr.list.indexMap.allIDs)
	if err != nil {
		return 0, err
	}

	return bs.Count(), nil
}

// Values the result values of the Query
func (qr *QueryResult[T, ID]) Values() ([]T, error) {
	result, _, err := qr.exec(Paginate{})
	return result, err
}

// Paginate the result values of the Query, but in Pagination
func (qr *QueryResult[T, ID]) Paginate(offset, limit uint32) ([]T, PageInfo, error) {
	return qr.exec(Paginate{Offset: offset, Limit: limit})
}

type PageInfo struct {
	Offset uint32
	Limit  uint32
	Count  uint32
	Total  uint32
}

type Paginate struct {
	Offset uint32
	Limit  uint32
}

// QueryPage executes the query with optional pagination.
// If opts is nil, it returns all matching results.
func (qr *QueryResult[T, ID]) exec(p Paginate) ([]T, PageInfo, error) {
	if qr.err != nil {
		return nil, PageInfo{}, qr.err
	}

	qr.list.lock.RLock()
	defer qr.list.lock.RUnlock()

	rids, _, err := qr.query(qr.list.indexMap.FilterByName, qr.list.indexMap.allIDs)
	if err != nil {
		return nil, PageInfo{}, err
	}

	total := uint32(rids.Count())
	offset := p.Offset
	limit := total // default to "all"
	// if limit is provided and not zero, use it; otherwise stay at "total"
	if p.Limit > 0 {
		limit = p.Limit
	}
	pi := PageInfo{Offset: offset, Limit: limit, Total: total}

	// bound check
	if offset >= total {
		return []T{}, pi, nil
	}

	// adjust limit if it exceeds the remaining items
	if offset+limit > total {
		limit = total - offset
	}
	pi.Count = limit

	startIndex := uint32(0)
	if offset > 0 {
		idx, found := rids.ValueOnIndex(offset)
		if !found {
			return []T{}, pi, nil
		}
		startIndex = idx
	}

	// the theoretical maximum bit index for the "to" parameter
	result := make([]T, 0, limit)

	maxBitIndex := uint32(rids.Max())
	rids.Range(startIndex, maxBitIndex, func(idx uint32) bool {
		item, _ := qr.list.list.Get(int(idx))
		result = append(result, item)

		// run only until reach the limit
		return uint32(len(result)) < limit
	})

	return result, pi, nil
}
