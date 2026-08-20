package mind

import (
	"fmt"
	"iter"
	"sync"

	"github.com/lima1909/mind/errr"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

// IndexMap maps a given field name to an Index
type IndexMap[OBJ any] struct {
	index  map[string]index.Index[OBJ]
	allIDs *lidx.RawIDs32
}

func NewIndexMap[OBJ any]() IndexMap[OBJ] {
	return IndexMap[OBJ]{
		index:  make(map[string]index.Index[OBJ]),
		allIDs: lidx.NewRawIDs[uint32](),
	}
}

// FilterByName finds the Filter by a given field-name
func (i IndexMap[OBJ]) FilterByName(fieldName string) (query.Filter, error) {
	if idx, found := i.index[fieldName]; found {
		return idx, nil
	}

	return nil, errr.InvalidNameError{FieldName: fieldName}
}

// Insert to all known indexes synchron the new value (including ID-index)
func (i IndexMap[OBJ]) AddIndex(fieldName string, index index.Index[OBJ]) error {
	if _, exist := i.index[fieldName]; exist {
		return fmt.Errorf("field-name: %s already exists", fieldName)
	}

	i.index[fieldName] = index
	return nil
}

// Insert to all known indexes synchron the new value (including ID-index)
func (i IndexMap[OBJ]) Insert(obj *OBJ, idx int) {
	uidx := uint32(idx)
	i.allIDs.Set(uidx)

	for _, fieldIndex := range i.index {
		fieldIndex.Set(obj, uidx)
	}
}

// bulkInsert creates a go routine for every creating Index
func (i IndexMap[OBJ]) bulkInsert(objs iter.Seq2[int, *OBJ]) {
	var wg sync.WaitGroup

	wg.Go(func() {
		for lidx := range objs {
			i.allIDs.Set(uint32(lidx))
		}
	})

	for _, fieldIndex := range i.index {
		wg.Go(func() {
			fieldIndex.BulkSet(objs)
		})
	}

	wg.Wait()
}

// Update Update all known indexes synchron the new value (including ID-index)
func (i IndexMap[OBJ]) Update(oldObj, newObj *OBJ, idx int) {
	uidx := uint32(idx)

	i.allIDs.UnSet(uidx)
	i.allIDs.Set(uidx)

	for _, index := range i.index {
		// only update, if the value has changed
		if index.HasChanged(oldObj, newObj) {
			index.UnSet(oldObj, uidx)
			index.Set(newObj, uidx)
		}
	}
}

// Delete remove all known indexes synchron the new value (including ID-index)
func (i IndexMap[OBJ]) Delete(obj *OBJ, idx int) {
	uidx := uint32(idx)

	i.allIDs.UnSet(uidx)

	for _, fieldIndex := range i.index {
		fieldIndex.UnSet(obj, uidx)
	}
}
