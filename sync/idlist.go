package sync

import (
	"sync"

	"github.com/lima1909/mind"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
)

// IDList is a fast in-memory collection, which is extended by Indices for fast finding Items.
//
// The IDList is thread safe!
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() or Replace() to modify data.
//
// WARNING: The list index, the position in the list (Slice Index) can be used again.
// This means, after removing an Item and inserting a new Item, the new Item can reuse the index of the removed Item.
type IDList[T any, ID comparable] struct {
	list *mind.IDList[T, ID]
	lock sync.RWMutex
}

// NewIDList create a new List with an ID-Index
func NewIDList[T any, ID comparable](fieldIDGetFn func(*T) ID) *IDList[T, ID] {
	return &IDList[T, ID]{list: mind.NewIDList(fieldIDGetFn)}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - Index: a impl of the Index interface
//
// Hint: empty field-names are not allowed and field-name 'id' is reserved!
func (l *IDList[T, ID]) CreateIndex(fieldName string, idx index.Index[T]) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.CreateIndex(fieldName, idx)
}

// RemoveIndex removed the Index with the given fieldName (what the name of the Index is).
// The reserved fieldNmae 'id' can not be removed.
func (l *IDList[T, ID]) RemoveIndex(fieldName string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.list.RemoveIndex(fieldName)
}

// InitialBulkInsert can be used for a more performant inserting of initial values.
// The List MUST be empty!
func (l *IDList[T, ID]) InitialBulkInsert(values mind.FreeList[T]) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.InitialBulkInsert(values)
}

// Insert add the given Item to the list and returns the list-index of this Item,
// which stays valid, until the Item is removed.
// An error orcurred, if the id already exist.
func (l *IDList[T, ID]) Insert(item T) (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.Insert(item)
}

// Update find the item by the given id and give the item to the update function,
// so can the caller update the fields he want. If no item found for the given ID,
// the method returns a ValueNotFoundError. Changing the ID is not allowed.
func (l *IDList[T, ID]) Update(id ID, update func(*T)) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.Update(id, update)
}

// Remove an item by the given ID.
// Returns true, if the ID exist, the list-index of the removed Item and an error, if occurred.
// errors: ID not found
func (l *IDList[T, ID]) Remove(id ID) (bool, int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.Remove(id)
}

// Get returns an item by the given ID.
// errors: ID not found
func (l *IDList[T, ID]) Get(id ID) (T, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Get(id)
}

// Contains check, is this ID found in the list.
func (l *IDList[T, ID]) Contains(id ID) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Contains(id)
}

// Len the Items, which in this list exist
func (l *IDList[T, ID]) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Len()
}

// ToValues extract all Items
func (l *IDList[T, ID]) ToValues() []T {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.ToValues()
}

// Values iterate over the complete List and call the yield function, for every item
func (l *IDList[T, ID]) Values(yield func(int, T) bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	l.list.Values(yield)
}

// QueryStr execute the given Query-string.
func (l *IDList[T, ID]) QueryStr(queryStr string, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromStr(listQuery[T]{l.list.Queryable(), &l.lock}, queryStr, opts...)
}

// Query execute the given Query.
func (l *IDList[T, ID]) Query(q query.Expr, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromExpr(listQuery[T]{l.list.Queryable(), &l.lock}, q, opts...)
}
