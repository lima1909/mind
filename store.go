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

// store is the base implementation of lists, like  List and IDList.
// A store combines a FreeList with the Items and a IndexMap with all registered indexes and the lock,
// which keeps both in sync.
type store[T any] struct {
	list     FreeList[T]
	indexMap IndexMap[T]

	lock sync.RWMutex

	// validateUpdate is an optional check, which runs after the update-function and before
	// the new Item is written back. The IDList uses it, to forbid changing the ID.
	validateUpdate func(oldItem, newItem *T) error
}

func newStore[T any]() store[T] {
	return store[T]{
		list:     NewFreeList[T](),
		indexMap: NewIndexMap[T](),
	}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - Index: a impl of the Index interface
//
// Hint: empty field-names are not allowed and field-name 'id' is reserved!
func (s *store[T]) CreateIndex(fieldName string, idx index.Index[T]) error {
	if strings.EqualFold(fieldName, query.IDIndexFieldName) {
		return fmt.Errorf("ID is a reserved field name")
	}

	if err := query.IsValidName(fieldName); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.indexMap.AddIndex(fieldName, idx); err != nil {
		return err
	}

	for pos, item := range s.list.Iter() {
		idx.Set(item, uint32(pos))
	}

	return nil
}

// RemoveIndex removed the Index with the given fieldName (what the name of the Index is).
// The reserved fieldNmae 'id' can not be removed.
func (s *store[T]) RemoveIndex(fieldName string) {
	if fieldName == "" || strings.EqualFold(fieldName, query.IDIndexFieldName) {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.indexMap.RemoveIndex(fieldName)
}

// InitialBulkInsert can be used for a more performant inserting of initial values.
// The List MUST be empty!
func (s *store[T]) InitialBulkInsert(values FreeList[T]) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.list.len > 0 {
		return errors.New("can not execute bulk insert for a non empty list")
	}

	// update all indexes (including the ID-Index of an IDList)
	s.indexMap.bulkInsert(values.Iter())
	s.list = values

	return nil
}

// Insert add the given Item to the list and returns the list-index of this Item,
// which stays valid, until the Item is removed.
func (s *store[T]) Insert(item T) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	lidx := s.list.Insert(item)
	s.indexMap.Insert(&item, lidx)

	return lidx
}

// Len the Items, which in this list exist
func (s *store[T]) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.list.Len()
}

// Values iterate over the complete List and call the yield function, for every item
// WARNING: While the Iterator is used, the List is locked.
func (s *store[T]) Values(yield func(int, T) bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for i := range s.list.slots {
		if slot := &s.list.slots[i]; slot.occupied {
			if !yield(i, slot.value) {
				return
			}
		}
	}
}

// ToValues extract all Items
func (s *store[T]) ToValues() []T {
	s.lock.RLock()
	defer s.lock.RUnlock()

	result := make([]T, 0, s.list.Len())

	for _, item := range s.list.slots {
		if item.occupied {
			result = append(result, item.value)
		}
	}

	return result
}

// updateAt gives the Item on the given slot position to the update function and keeps
// all indexes in sync. Nothing is written, if the validateUpdate check fails.
// The caller MUST hold the write lock.
func (s *store[T]) updateAt(pos int, update func(*T)) error {
	item, found := s.list.Get(pos)
	if !found {
		return errr.ValueNotFoundError{Value: pos}
	}

	// save old value
	oldItem := item

	update(&item)

	if s.validateUpdate != nil {
		if err := s.validateUpdate(&oldItem, &item); err != nil {
			return err
		}
	}

	s.list.Update(pos, item)
	s.indexMap.Update(&oldItem, &item, pos)

	return nil
}

// removeAt removes the Item on the given slot position from the list and from all indexes.
// The caller MUST hold the write lock.
func (s *store[T]) removeAt(pos int) bool {
	if item, found := s.list.Get(pos); found {
		s.indexMap.Remove(&item, pos)
		return s.list.Remove(pos)
	}

	return false
}

// QueryStr execute the given Query-string.
func (s *store[T]) QueryStr(queryStr string, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromStr(s.hfns(), queryStr, opts...)
}

// Query execute the given Query.
func (s *store[T]) Query(q query.Expr, opts ...query.Opion) query.QHandle[T] {
	return query.NewQHandleFromExpr(s.hfns(), q, opts...)
}

// implements the QHandle interface
// ------------------------------------------
func (s *store[T]) hfns() query.HandleFNs[T] {
	return query.HandleFNs[T]{
		ReadQuery:    s.readQuery,
		WriteQuery:   s.writeQuery,
		RemoveItem:   s.removeAt,
		UpdateItem:   s.updateAt,
		GetManyItems: s.list.getMany,
	}
}

func (s *store[T]) readQuery(q query.Query, exec func(*lidx.RawIDs32)) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	rids, _, err := q(s.indexMap.FilterByName, s.indexMap.allIDs)
	if err != nil {
		return err
	}

	exec(rids)
	return nil
}

func (s *store[T]) writeQuery(q query.Query, exec func(*lidx.RawIDs32)) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	rids, canMutate, err := q(s.indexMap.FilterByName, s.indexMap.allIDs)
	if err != nil {
		return err
	}

	if !canMutate {
		rids = rids.Copy()
	}

	exec(rids)
	return nil
}
