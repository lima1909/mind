package mind

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/lima1909/mind/errr"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/lidx"
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
	list     FreeList[T]
	idIndex  index.IdIndex[T, ID]
	indexMap IndexMap[T]

	lock sync.RWMutex
}

// NewIDList create a new List with an ID-Index
func NewIDList[T any, ID comparable](fieldIDGetFn func(*T) ID) *IDList[T, ID] {
	return &IDList[T, ID]{
		list:     NewFreeList[T](),
		idIndex:  index.NewIDMapIndex(fieldIDGetFn),
		indexMap: NewIndexMap[T](),
	}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - Index: a impl of the Index interface
//
// Hint: empty field-name or the field-name ID are not allowed!
func (l *IDList[T, ID]) CreateIndex(fieldName string, index index.Index[T]) error {
	if strings.ToLower(fieldName) == query.IDIndexFieldName {
		return fmt.Errorf("ID is a reserved field name")
	}

	if err := query.IsValidName(fieldName); err != nil {
		return err
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	// add index
	if err := l.indexMap.AddIndex(fieldName, index); err != nil {
		return err
	}

	// add index values from the list
	for idx, item := range l.list.Iter() {
		index.Set(item, uint32(idx))
	}

	return nil
}

// RemoveIndex removed a the Index with the given field-name (what the name of the Index is)
func (l *IDList[T, ID]) RemoveIndex(fieldName string) {
	if fieldName == "" {
		return
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	delete(l.indexMap.index, fieldName)
}

// InitialBulkInsert can be used for a more performant inserting of initial values.
// The List MUST be empty!
func (l *IDList[T, ID]) InitialBulkInsert(values FreeList[T]) error {
	if l.list.count > 0 {
		return errors.New("can not execute bulk insert for a non empty ID-list")
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	// update all indexes
	l.indexMap.bulkInsert(values.Iter())
	l.idIndex.BulkSet(values.Iter())
	l.list = values

	return nil
}

// Insert add the given Item to the list and
// returns the index, the position in the IDList.
func (l *IDList[T, ID]) Insert(item T) int {
	l.lock.Lock()
	defer l.lock.Unlock()

	idx := l.list.Insert(item)
	l.idIndex.Set(&item, uint32(idx))
	l.indexMap.Insert(&item, idx)

	return idx
}

// Remove an item by the given ID.
// Returns true, if the ID exist, the old Index in the List and an error, if occurred
// errors:
// - wrong datatype
// - ID not found
func (l *IDList[T, ID]) Remove(id ID) (bool, int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	index, found := l.idIndex.GetIndex(id)
	if !found {
		return false, -1, errr.ValueNotFoundError{Value: id}
	}

	idx := int(index)
	item, _ := l.list.Get(idx)
	removed := l.list.Remove(idx)

	l.idIndex.UnSet(&item, index)
	l.indexMap.Delete(&item, idx)

	return removed, int(index), nil

}

// Replace find the old item by the ID of the given item and
// replace the old item with the new item and consistently updates all registered indexes.
func (l *IDList[T, ID]) Replace(item T) (T, error) {
	id := l.idIndex.GetID(&item)

	l.lock.Lock()
	defer l.lock.Unlock()

	idx, found := l.idIndex.GetIndex(id)
	if !found {
		var zero T
		return zero, errr.ValueNotFoundError{Value: id}
	}

	// overwrite the data in the main list
	oldItem, ok := l.list.Update(int(idx), item)
	if !ok {
		var zero T
		return zero, errr.ValueNotFoundError{Value: id}
	}

	// update all indexes: re-index
	// NO Update in idIndex , because the id is on the same Idx
	l.indexMap.Update(&oldItem, &item, int(idx))

	return oldItem, nil
}

// Update find the item by the given id and give the item to the update function,
// so can the caller update the fields he want. If no item found for the given ID,
// the method returns a ValueNotFoundError.
func (l *IDList[T, ID]) Update(id ID, update func(*T)) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	idx, found := l.idIndex.GetIndex(id)
	if !found {
		return errr.ValueNotFoundError{Value: id}
	}

	if item, found := l.list.Get(int(idx)); found {
		// save old value
		oldItem := item

		update(&item)

		newID := l.idIndex.GetID(&item)
		if id != newID {
			return fmt.Errorf("changing the id from: %v to %v is not allowed", id, newID)
		}

		l.list.Update(int(idx), item)
		// NO Update in idIndex , because the id is on the same Idx
		l.indexMap.Update(&oldItem, &item, int(idx))

		return nil
	}

	return errr.ValueNotFoundError{Value: id}
}

// Get returns an item by the given ID.
// errors:
// - wrong datatype
// - ID not found
//
//go:inline
func (l *IDList[T, ID]) Get(id ID) (T, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	idx, found := l.idIndex.GetIndex(id)
	if !found {
		var null T
		return null, errr.ValueNotFoundError{Value: id}
	}

	// not found should be NOT possible
	item, _ := l.list.Get(int(idx))
	return item, nil
}

// Contains check, is this ID found in the list.
func (l *IDList[T, ID]) Contains(id ID) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	_, found := l.idIndex.GetIndex(id)
	return found
}

// Count the Items, which in this list exist
func (l *IDList[T, ID]) Count() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Count()
}

// QueryStr execute the given Query-string.
func (l *IDList[T, ID]) QueryStr(queryStr string, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromStr(l.execQuery, queryStr, opts...)
}

// Query execute the given Query.
func (l *IDList[T, ID]) Query(q query.Expr, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromExpr(l.execQuery, q, opts...)
}

// implements the QHandle interface
// -----------------------------------------
func (l *IDList[T, ID]) filterByName(fieldName string) (query.Filter, error) {
	if strings.ToLower(fieldName) == query.IDIndexFieldName {
		return l.idIndex, nil
	}

	return l.indexMap.FilterByName(fieldName)
}

func (l *IDList[T, ID]) execQuery(query query.Query, exec func(*lidx.RawIDs32, query.GetItemFn[T])) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	rids, _, err := query(l.filterByName, l.indexMap.allIDs)
	if err != nil {
		return err
	}

	exec(rids, func(lidx uint32) (T, bool) {
		return l.list.Get(int(lidx))
	})

	return nil
}
