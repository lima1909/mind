package sync

import (
	"sync"

	"github.com/lima1909/mind"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
)

// List is a fast in-memory collection, which is extended by Indices for fast finding Items.
//
// The List is thread safe!
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() to modify data.
//
// WARNING: The list index, the position in the list (Slice Index) can be used again.
// This means, after removing an Item and inserting a new Item, the new Item can reuse the index of the removed Item.
type List[T any] struct {
	list *mind.List[T]
	lock sync.RWMutex
}

func NewList[T any]() *List[T] {
	return &List[T]{list: mind.NewList[T]()}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - Index: a impl of the Index interface
func (l *List[T]) CreateIndex(fieldName string, idx index.Index[T]) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.CreateIndex(fieldName, idx)
}

// RemoveIndex removed the Index with the given fieldName (what the name of the Index is).
func (l *List[T]) RemoveIndex(fieldName string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.list.RemoveIndex(fieldName)
}

// InitialBulkInsert can be used for a more performant inserting of initial values.
// The List MUST be empty!
func (l *List[T]) InitialBulkInsert(values mind.FreeList[T]) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.InitialBulkInsert(values)
}

// Insert add the given Item to the list and returns the list-index of this Item,
// which stays valid, until the Item is removed.
func (l *List[T]) Insert(item T) int {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.Insert(item)
}

// Update gives the Item of the given list-index to the update function, so the caller can
// update the fields he wants, and keeps all indexes in sync.
// Returns false, if the list-index is unknown or the Item was removed.
func (l *List[T]) Update(lidx int, update func(*T)) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.Update(lidx, update)

}

// func (l *List[T]) updateAt(lidx int, update func(*T)) error {
// }

// Remove the Item of the given list-index.
// Returns false, if the list-index is unknown or the Item was already removed.
func (l *List[T]) Remove(lidx int) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.Remove(lidx)
}

// Get the Item of the given list-index, or the zero value and false,
func (l *List[T]) Get(lidx int) (T, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Get(lidx)
}

// ToValues extract all Items
func (l *List[T]) ToValues() []T {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.ToValues()
}

// Values iterate over the complete List and call the yield function, for every item
func (l *List[T]) Values(yield func(int, T) bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	l.list.Values(yield)
}

// Len the Items, which in this list exist
func (l *List[T]) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Len()
}

// QueryStr execute the given Query-string.
func (l *List[T]) QueryStr(queryStr string, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromStr(listQuery[T]{l.list.Queryable(), &l.lock}, queryStr, opts...)
}

// Query execute the given Query.
func (l *List[T]) Query(q query.Expr, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromExpr(listQuery[T]{l.list.Queryable(), &l.lock}, q, opts...)
}

// implements the Queryable interface
// ------------------------------------------
type listQuery[T any] struct {
	q    query.Queryable[T]
	lock *sync.RWMutex
}

func (l listQuery[T]) Count(q query.Query) (int, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.q.Count(q)
}

func (l listQuery[T]) Values(q query.Query, precicate func(*T) bool) ([]T, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.q.Values(q, precicate)
}

func (l listQuery[T]) Paginate(q query.Query, offset uint32, result *[]T) (query.PageInfo, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.q.Paginate(q, offset, result)
}

func (l listQuery[T]) Update(q query.Query, update func(*T)) (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.q.Update(q, update)
}

func (l listQuery[T]) Remove(q query.Query) (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.q.Remove(q)
}
