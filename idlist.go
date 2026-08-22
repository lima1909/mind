package mind

import (
	"fmt"

	"github.com/lima1909/mind/errr"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
)

// IDList is a fast in-memory store, which is extended by Indices for fast finding Items.
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() or Replace() to modify data.
//
// WARNING: The list index, the position in the list (Slice Index) can be used again.
// This means, after removing an Item and inserting a new Item, the new Item can reuse the index of the removed Item.

type IDList[T any, ID comparable] struct {
	store[T]

	// idIndex is additionally registered in the IndexMap of the store, under the reserved
	// field-name "id". So every Insert/Update/Delete of the store keeps it in sync like any
	// other Index and a query `id = ...` needs no special case.
	// The typed reference is kept here, for the lookups ID -> slot position.
	idIndex index.IdIndex[T, ID]
}

// NewIDList create a new List with an ID-Index
func NewIDList[T any, ID comparable](fieldIDGetFn func(*T) ID) *IDList[T, ID] {
	l := &IDList[T, ID]{
		store:   newStore[T](),
		idIndex: index.NewIDMapIndex(fieldIDGetFn),
	}

	// can not fail: the IndexMap of a new store is empty
	_ = l.indexMap.AddIndex(query.IDIndexFieldName, l.idIndex)
	l.validateUpdate = l.checkIDHasNotChanged

	return l
}

// checkIDHasNotChanged is the update-check of an IDList: the ID is the identity of an Item,
// so an update is not allowed to change it.
func (l *IDList[T, ID]) checkIDHasNotChanged(oldItem, newItem *T) error {
	oldID, newID := l.idIndex.GetID(oldItem), l.idIndex.GetID(newItem)
	if oldID != newID {
		return fmt.Errorf("changing the id from: %v to %v is not allowed", oldID, newID)
	}

	return nil
}

// Get returns an item by the given ID.
// errors: ID not found
func (l *IDList[T, ID]) Get(id ID) (T, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	lidx, found := l.idIndex.GetIndex(id)
	if !found {
		var null T
		return null, errr.ValueNotFoundError{Value: id}
	}

	// not found should be NOT possible
	item, _ := l.list.Get(int(lidx))
	return item, nil
}

// Contains check, is this ID found in the list.
func (l *IDList[T, ID]) Contains(id ID) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	_, found := l.idIndex.GetIndex(id)
	return found
}

// Remove an item by the given ID.
// Returns true, if the ID exist, the list-index of the removed Item and an error, if occurred.
// errors: ID not found
func (l *IDList[T, ID]) Remove(id ID) (bool, int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	lidx, found := l.idIndex.GetIndex(id)
	if !found {
		return false, -1, errr.ValueNotFoundError{Value: id}
	}

	return l.removeAt(int(lidx)), int(lidx), nil
}

// Replace find the old item by the ID of the given item and
// replace the old item with the new item and consistently updates all registered indexes.
func (l *IDList[T, ID]) Replace(item T) (T, error) {
	id := l.idIndex.GetID(&item)

	l.lock.Lock()
	defer l.lock.Unlock()

	lidx, found := l.idIndex.GetIndex(id)
	if !found {
		var zero T
		return zero, errr.ValueNotFoundError{Value: id}
	}

	// overwrite the data in the main list
	oldItem, ok := l.list.Update(int(lidx), item)
	if !ok {
		var zero T
		return zero, errr.ValueNotFoundError{Value: id}
	}

	// update all indexes: re-index
	// the ID-Index is one of them, but the ID has not changed, so it stays untouched
	l.indexMap.Update(&oldItem, &item, int(lidx))

	return oldItem, nil
}

// Update find the item by the given id and give the item to the update function,
// so can the caller update the fields he want. If no item found for the given ID,
// the method returns a ValueNotFoundError. Changing the ID is not allowed.
func (l *IDList[T, ID]) Update(id ID, update func(*T)) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	lidx, found := l.idIndex.GetIndex(id)
	if !found {
		return errr.ValueNotFoundError{Value: id}
	}

	return l.updateAt(int(lidx), update)
}
