package mind

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

// List is a fast in-memory store, which is extended by Indices for fast finding Items.
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() to modify data.
type List[T any] struct {
	list     FreeList[T]
	indexMap IndexMap[T]

	lock sync.RWMutex
}

// NewList create a new List
func NewList[T any]() *List[T] {
	return &List[T]{
		list:     NewFreeList[T](),
		indexMap: NewIndexMap[T](lidx.NewRawIDs[uint32]()),
	}
}

// CreateIndex create a new Index:
//   - fieldName: a name for a field of the saved Item
//   - Index: a impl of the Index interface
//
// Hint: empty field-name are not allowed!
func (l *List[T]) CreateIndex(fieldName string, index Index[T]) error {
	if err := query.IsValidName(fieldName); err != nil {
		return err
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	if _, exist := l.indexMap.index[fieldName]; exist {
		return fmt.Errorf("field-name: %s already exists", fieldName)
	}

	for idx, item := range l.list.Iter() {
		index.Set(item, uint32(idx))
	}
	l.indexMap.index[fieldName] = index

	return nil
}

// RemoveIndex removed the Index with the given field-name (what the name of the Index is)
func (l *List[T]) RemoveIndex(fieldName string) {
	if fieldName == "" {
		return
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	delete(l.indexMap.index, fieldName)
}

// InitialBulkInsert can be used for a more performant inserting of initial values.
// The List MUST be empty!
func (l *List[T]) InitialBulkInsert(values FreeList[T]) error {
	if l.list.count > 0 {
		return errors.New("can not execute bulk insert for a non empty list")
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	// update all indexes
	l.indexMap.bulkInsert(values.Iter())
	l.list = values

	return nil
}

// Insert add the given Item to the list,
func (l *List[T]) Insert(item T) int {
	l.lock.Lock()
	defer l.lock.Unlock()

	idx := l.list.Insert(item)
	l.indexMap.insert(&item, idx)

	return idx
}

func (l *List[T]) Update(index int, update func(*T)) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	if item, found := l.list.Get(index); found {
		// save old value
		oldItem := item

		update(&item)

		l.list.Update(index, item)
		l.indexMap.update(&oldItem, &item, index)
		return true
	}

	return false
}

func (l *List[T]) Remove(index int) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	if item, found := l.list.Get(index); found {
		l.indexMap.delete(&item, index)
		return l.list.Remove(index)
	}

	return false
}

func (l *List[T]) Get(index int) (T, bool) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.list.Get(index)
}

// Values iterate over the complete List and call the yield function, for every item
func (l *List[T]) Values(yield func(int, T) bool) {
	l.lock.Lock()
	defer l.lock.Unlock()

	for i, item := range l.list.slots {
		if item.occupied {
			if !yield(i, item.value) {
				return
			}
		}
	}
}

// Count the Items, which in this list exist
func (l *List[T]) Count() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Count()
}

// QueryStr execute the given Query-string.
func (l *List[T]) QueryStr(queryStr string, opts ...Opion) QHandle[T] {
	return NewQHandleFromStr(l.execQuery, queryStr, opts...)
}

// Query execute the given Query.
func (l *List[T]) Query(query query.Expr, opts ...Opion) QHandle[T] {
	return NewQHandleFromExpr(l.execQuery, query, opts...)
}

// implements the QHandle interface
// ------------------------------------------
func (l *List[T]) execQuery(query query.Query, exec func(*lidx.RawIDs32, getItemFn[T])) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	rids, _, err := query(l.indexMap.FilterByName, l.indexMap.allIDs)
	if err != nil {
		return err
	}

	exec(rids, func(lidx uint32) (T, bool) {
		return l.list.Get(int(lidx))
	})

	return nil
}
