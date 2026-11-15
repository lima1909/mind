package mind

import (
	"errors"

	"github.com/lima1909/mind/errr"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
)

// List is a fast in-memory collection, which is extended by Indices for fast finding Items.
//
// WARNING: The List is NOT thread safe!
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() to modify data.
//
// WARNING: The list index, the position in the list (Slice Index) can be used again.
// This means, after removing an Item and inserting a new Item, the new Item can reuse the index of the removed Item.
type List[T any] struct {
	list     FreeList[T]
	indexMap IndexMap[T]

	// save the ref to the FilterByName func to save allocations
	filterByName query.FilterByName
}

// NewList create a new List
func NewList[T any]() *List[T] {
	l := newList[T]()
	return &l
}

func newList[T any]() List[T] {
	indexMap := NewIndexMap[T]()
	return List[T]{
		list:         NewFreeList[T](),
		indexMap:     indexMap,
		filterByName: indexMap.FilterByName,
	}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - Index: a impl of the Index interface
func (l *List[T]) CreateIndex(fieldName string, idx index.Index[T]) error {
	if err := l.indexMap.AddIndex(fieldName, idx); err != nil {
		return err
	}

	for pos, item := range l.list.Iter() {
		idx.Set(item, uint32(pos))
	}

	return nil
}

// RemoveIndex removed the Index with the given fieldName (what the name of the Index is).
func (l *List[T]) RemoveIndex(fieldName string) { l.indexMap.RemoveIndex(fieldName) }

// InitialBulkInsert can be used for a more performant inserting of initial values.
// The List MUST be empty!
func (l *List[T]) InitialBulkInsert(values FreeList[T]) error {
	if l.list.len > 0 {
		return errors.New("can not execute bulk insert for a non empty list")
	}

	// update all indexes (including the ID-Index of an IDList)
	l.indexMap.bulkInsert(values.Iter())
	l.list = values

	return nil
}

// Insert add the given Item to the list and returns the list-index of this Item,
// which stays valid, until the Item is removed.
func (l *List[T]) Insert(item T) int {
	lidx := l.list.Insert(item)
	l.indexMap.Insert(&item, lidx)
	return lidx
}

// Update gives the Item of the given list-index to the update function, so the caller can
// update the fields he wants, and keeps all indexes in sync.
// Returns false, if the list-index is unknown or the Item was removed.
func (l *List[T]) Update(lidx int, update func(*T)) error { return l.updateAt(lidx, update) }

func (l *List[T]) updateAt(lidx int, update func(*T)) error {
	item, found := l.list.Get(lidx)
	if !found {
		return errr.ValueNotFoundError{Value: lidx}
	}

	// save old value, the item MUST BE copyable (not a map, slice, ...)
	oldItem := item
	update(&item)

	l.list.Update(lidx, item)
	l.indexMap.Update(&oldItem, &item, lidx)
	return nil
}

// Remove the Item of the given list-index.
// Returns false, if the list-index is unknown or the Item was already removed.
func (l *List[T]) Remove(lidx int) bool {
	if item, found := l.list.Get(lidx); found {
		l.indexMap.Remove(&item, lidx)
		_, removed := l.list.Remove(lidx)
		return removed
	}

	return false
}

// Get the Item of the given list-index, or the zero value and false,
func (l *List[T]) Get(lidx int) (T, bool) { return l.list.Get(lidx) }

// ToValues extract all Items
func (l *List[T]) ToValues() []T { return l.list.ToValues() }

// Values iterate over the complete List and call the yield function, for every item
func (l *List[T]) Values(yield func(int, T) bool) { l.list.Values(yield) }

// Len the Items, which in this list exist
func (l *List[T]) Len() int { return l.list.len }

func (l *List[T]) Queryable() query.Queryable[T] { return listQuery[T, *List[T]]{l, l} }

// QueryStr execute the given Query-string.
func (l *List[T]) QueryStr(queryStr string, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromStr(l.Queryable(), queryStr, opts...)
}

// Query execute the given Query.
func (l *List[T]) Query(q query.Expr, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromExpr(l.Queryable(), q, opts...)
}

// implements the Queryable interface
// ------------------------------------------
type updater[T any] interface {
	updateAt(lidx int, update func(*T)) error
}

type listQuery[T any, U updater[T]] struct {
	*List[T]
	updater U
}

func (l listQuery[T, U]) Count(q query.Query) (int, error) {
	rids, _, err := q(l.filterByName, l.indexMap.allIDs)
	if err != nil {
		return 0, err
	}

	return rids.Count(), nil
}

func (l listQuery[T, U]) Values(q query.Query, precicate func(*T) bool) ([]T, error) {
	rids, _, err := q(l.filterByName, l.indexMap.allIDs)
	if err != nil {
		return nil, err
	}

	sl := rids.ToSlice()
	result := make([]T, 0, len(sl))
	if precicate != nil {
		l.list.FilterUnChecked(sl, &result, precicate)
	} else {
		l.list.GetManyUnChecked(sl, &result)
	}
	return result, nil
}

func (l listQuery[T, U]) Paginate(q query.Query, offset uint32, result *[]T) (query.PageInfo, error) {
	var pi query.PageInfo

	rids, _, err := q(l.filterByName, l.indexMap.allIDs)
	if err != nil {
		return pi, err
	}

	total := uint32(rids.Count())
	limit := uint32(cap(*result))

	// early exit, if limit = 0
	if limit == 0 {
		pi.Offset = offset
		pi.Total = total
		return pi, nil
	}

	pi = query.Paginate{Offset: offset, Limit: limit}.ComputePageInfo(total)

	idxs := rids.ValuesBetween(int(pi.Offset), int(pi.Limit))
	l.list.GetManyUnChecked(idxs, result)
	return pi, nil
}

func (l listQuery[T, U]) Update(q query.Query, update func(*T)) (int, error) {
	rids, canMutate, err := q(l.filterByName, l.indexMap.allIDs)
	if err != nil {
		return 0, err
	}

	if !canMutate {
		rids = rids.Copy()
	}

	count := 0
	var errr error
	rids.Values(func(idx uint32) bool {

		errr = l.updater.updateAt(int(idx), update)
		if errr != nil {
			return false
		}

		count++
		return true
	})

	if errr != nil {
		return 0, errr
	}

	return count, err
}

func (l listQuery[T, U]) Remove(q query.Query) (int, error) {
	rids, canMutate, err := q(l.filterByName, l.indexMap.allIDs)
	if err != nil {
		return 0, err
	}

	if !canMutate {
		rids = rids.Copy()
	}

	count := 0
	rids.Values(func(idx uint32) bool {
		if l.List.Remove(int(idx)) {
			count++
		}
		return true
	})

	return count, nil
}
