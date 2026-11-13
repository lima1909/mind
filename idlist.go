package mind

import (
	"fmt"
	"strings"

	"github.com/lima1909/mind/errr"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
)

// IDList is a fast in-memory store, which is extended by Indices for fast finding Items.
//
// WARNING: The IDList ist NOT thread safe!
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() or Replace() to modify data.
//
// WARNING: The list index, the position in the list (Slice Index) can be used again.
// This means, after removing an Item and inserting a new Item, the new Item can reuse the index of the removed Item.
type IDList[T any, ID comparable] struct {
	list List[T]

	// idIndex is additionally registered in the IndexMap of the store, under the reserved
	// field-name "id". So every Insert/Update/Delete of the store keeps it in sync like any
	// other Index and a query `id = ...` needs no special case.
	// The typed reference is kept here, for the lookups ID -> slot position.
	idIndex index.IdIndex[T, ID]
}

// NewIDList create a new List with an ID-Index
func NewIDList[T any, ID comparable](fieldIDGetFn func(*T) ID) *IDList[T, ID] {
	list := newList[T]()
	idIndex := index.NewIDMapIndex(fieldIDGetFn)
	// can not fail: the IndexMap of a new store is empty
	_ = list.indexMap.AddIndex(query.IDIndexFieldName, idIndex)

	return &IDList[T, ID]{list: list, idIndex: idIndex}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - Index: a impl of the Index interface
//
// Hint: empty field-names are not allowed and field-name 'id' is reserved!
func (l *IDList[T, ID]) CreateIndex(fieldName string, idx index.Index[T]) error {
	if strings.EqualFold(fieldName, query.IDIndexFieldName) {
		return fmt.Errorf("ID is a reserved field name")
	}

	return l.list.CreateIndex(fieldName, idx)
}

// RemoveIndex removed the Index with the given fieldName (what the name of the Index is).
// The reserved fieldNmae 'id' can not be removed.
func (l *IDList[T, ID]) RemoveIndex(fieldName string) {
	if strings.EqualFold(fieldName, query.IDIndexFieldName) {
		return
	}

	l.list.indexMap.RemoveIndex(fieldName)
}

// InitialBulkInsert can be used for a more performant inserting of initial values.
// The List MUST be empty!
func (l *IDList[T, ID]) InitialBulkInsert(values FreeList[T]) error {
	return l.list.InitialBulkInsert(values)
}

// Get returns an item by the given ID.
// errors: ID not found
func (l *IDList[T, ID]) Get(id ID) (T, error) {
	lidx, found := l.idIndex.GetIndex(id)
	if !found {
		var null T
		return null, errr.ValueNotFoundError{Value: id}
	}

	// not found should be NOT possible
	item, _ := l.list.list.Get(int(lidx))
	return item, nil
}

// Contains check, is this ID found in the list.
func (l *IDList[T, ID]) Contains(id ID) bool {
	_, found := l.idIndex.GetIndex(id)
	return found
}

// Insert add the given Item to the list and returns the list-index of this Item,
// which stays valid, until the Item is removed.
// An error orcurred, if the id already exist.
func (l *IDList[T, ID]) Insert(item T) (int, error) {
	id := l.idIndex.GetID(&item)

	if _, found := l.idIndex.GetIndex(id); found {
		return -1, errr.IdAlreadyExistsError[ID]{ID: id}
	}

	return l.list.Insert(item), nil
}

// Update find the item by the given id and give the item to the update function,
// so can the caller update the fields he want. If no item found for the given ID,
// the method returns a ValueNotFoundError. Changing the ID is not allowed.
func (l *IDList[T, ID]) Update(id ID, update func(*T)) error {
	lidx, found := l.idIndex.GetIndex(id)
	if !found {
		return errr.ValueNotFoundError{Value: id}
	}

	return l.updateAt(int(lidx), update)
}

func (l *IDList[T, ID]) updateAt(lidx int, update func(*T)) error {
	item, found := l.list.Get(lidx)
	if !found {
		return errr.ValueNotFoundError{Value: lidx}
	}

	// save old value, the item MUST BE copyable (not a map, slice, ...)
	oldItem := item
	update(&item)

	// the ID is the identity of an Item, so an update is not allowed to change it
	oldID, newID := l.idIndex.GetID(&oldItem), l.idIndex.GetID(&item)
	if oldID != newID {
		return fmt.Errorf("changing the id from: %v to %v is not allowed", oldID, newID)
	}

	l.list.list.Update(lidx, item)
	l.list.indexMap.Update(&oldItem, &item, lidx)

	return nil
}

// Remove an item by the given ID.
// Returns true, if the ID exist, the list-index of the removed Item and an error, if occurred.
// errors: ID not found
func (l *IDList[T, ID]) Remove(id ID) (bool, int, error) {
	lidx, found := l.idIndex.GetIndex(id)
	if !found {
		return false, -1, errr.ValueNotFoundError{Value: id}
	}

	return l.list.Remove(int(lidx)), int(lidx), nil
}

// Len the Items, which in this list exist
func (l *IDList[T, ID]) Len() int { return l.list.list.len }

// ToValues extract all Items
func (l *IDList[T, ID]) ToValues() []T { return l.list.list.ToValues() }

// Values iterate over the complete List and call the yield function, for every item
func (l *IDList[T, ID]) Values(yield func(int, T) bool) { l.list.list.Values(yield) }

// QueryStr execute the given Query-string.
func (l *IDList[T, ID]) QueryStr(queryStr string, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromStr(listQuery[T, *IDList[T, ID]]{&l.list, l}, queryStr, opts...)
}

// Query execute the given Query.
func (l *IDList[T, ID]) Query(q query.Expr, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromExpr(listQuery[T, *IDList[T, ID]]{&l.list, l}, q, opts...)
}
