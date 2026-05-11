package mind

import (
	"cmp"
	"iter"
	"sync"
)

const IDIndexFieldName = "id"

// indexMap maps a given field name to an Index
type indexMap[OBJ any, ID comparable] struct {
	index   map[string]Index[OBJ]
	idIndex idIndex[OBJ, ID]
	allIDs  *RawIDs32
}

func newIndexMap[OBJ any, ID comparable](idIndex idIndex[OBJ, ID]) indexMap[OBJ, ID] {
	return indexMap[OBJ, ID]{
		idIndex: idIndex,
		index:   make(map[string]Index[OBJ]),
		allIDs:  NewRawIDs[uint32](),
	}
}

// FilterByName finds the Filter by a given field-name
func (i indexMap[OBJ, ID]) FilterByName(fieldName string) (Filter, error) {
	if fieldName == IDIndexFieldName {
		if i.idIndex == nil {
			return nil, NoIdIndexDefinedError{}
		}
		return i.idIndex, nil
	}

	if idx, found := i.index[fieldName]; found {
		return idx, nil
	}

	return nil, InvalidNameError{fieldName}
}

// insert to all known indexes synchron the new value (including ID-index)
func (i indexMap[OBJ, ID]) insert(obj *OBJ, idx int) {
	uidx := uint32(idx)

	if i.idIndex != nil {
		i.idIndex.Set(obj, uidx)
	}

	i.allIDs.Set(uidx)

	for _, fieldIndex := range i.index {
		fieldIndex.Set(obj, uidx)
	}
}

// bulkInsert creates a go routine for every creating Index
func (i indexMap[OBJ, ID]) bulkInsert(objs iter.Seq2[int, *OBJ]) {
	var wg sync.WaitGroup

	if i.idIndex != nil {
		wg.Go(func() {
			i.idIndex.BulkSet(objs)
		})
	}

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

// update update all known indexes synchron the new value (including ID-index)
func (i indexMap[OBJ, ID]) update(oldObj, newObj *OBJ, idx int) {
	uidx := uint32(idx)

	if i.idIndex != nil {
		if i.idIndex.HasChanged(oldObj, newObj) {
			i.idIndex.UnSet(oldObj, uidx)
			i.idIndex.Set(newObj, uidx)
		}
	}

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

// delete remove all known indexes synchron the new value (including ID-index)
func (i indexMap[OBJ, ID]) delete(obj *OBJ, idx int) {
	uidx := uint32(idx)

	if i.idIndex != nil {
		i.idIndex.UnSet(obj, uidx)
	}

	i.allIDs.UnSet(uidx)

	for _, fieldIndex := range i.index {
		fieldIndex.UnSet(obj, uidx)
	}
}

func (i indexMap[OBJ, ID]) getListIdxByID(id ID) (uint32, error) {
	if i.idIndex == nil {
		return 0, NoIdIndexDefinedError{}
	}

	return i.idIndex.GetIndex(id)
}

func (i indexMap[OBJ, ID]) getIDByItem(item *OBJ) (ID, uint32, error) {
	if i.idIndex == nil {
		var id ID
		return id, 0, NoIdIndexDefinedError{}
	}

	return i.idIndex.GetID(item)
}

type idIndex[OBJ any, ID comparable] interface {
	Index[OBJ]
	GetIndex(ID) (uint32, error)
	GetID(*OBJ) (ID, uint32, error)
}

const IDMapIndexName = "IDMapIndex"

type idMapIndex[OBJ any, ID comparable] struct {
	data       map[ID]uint32
	fieldGetFn FromField[OBJ, ID]
}

func newIDMapIndex[OBJ any, ID comparable](fieldGetFn FromField[OBJ, ID]) idIndex[OBJ, ID] {
	return &idMapIndex[OBJ, ID]{
		data:       make(map[ID]uint32),
		fieldGetFn: fieldGetFn,
	}
}

func (mi *idMapIndex[OBJ, ID]) Set(obj *OBJ, lidx uint32) {
	id := mi.fieldGetFn(obj)
	mi.data[id] = lidx
}

func (mi idMapIndex[OBJ, ID]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	for lidx, obj := range objs {
		id := mi.fieldGetFn(obj)
		mi.data[id] = uint32(lidx)
	}
}

func (mi *idMapIndex[OBJ, ID]) UnSet(obj *OBJ, lidx uint32) {
	id := mi.fieldGetFn(obj)
	delete(mi.data, id)
}

func (mi *idMapIndex[OBJ, ID]) HasChanged(oldItem, newItem *OBJ) bool {
	return mi.fieldGetFn(oldItem) != mi.fieldGetFn(newItem)
}

func (mi *idMapIndex[OBJ, ID]) GetIndex(id ID) (uint32, error) {
	if lidx, found := mi.data[id]; found {
		return lidx, nil
	}

	return 0, ValueNotFoundError{id}
}

func (mi *idMapIndex[OBJ, ID]) GetID(item *OBJ) (ID, uint32, error) {
	id := mi.fieldGetFn(item)
	if lidx, found := mi.data[id]; found {
		return id, lidx, nil
	}

	var null ID
	return null, 0, ValueNotFoundError{id}
}

func (mi *idMapIndex[OBJ, ID]) Equal(value any) (*RawIDs32, error) {
	id, ok := value.(ID)
	if !ok {
		return nil, InvalidValueTypeError[ID]{value}
	}

	idx, err := mi.GetIndex(id)
	if err != nil {
		return nil, err
	}

	return NewRawIDsFrom(uint32(idx)), nil
}

func (mi *idMapIndex[OBJ, ID]) Match(_ *RawIDs32, op FilterOp, _ any) (*RawIDs32, error) {
	return nil, InvalidOperationError{IDMapIndexName, op.Op}
}

// MatchMany is not supported by idMapIndex, so that always returns an error
func (mi *idMapIndex[OBJ, ID]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
	return nil, InvalidOperationError{IDMapIndexName, op.Op}
}

// ------------------------------------------
// here starts the Index with the Index impls
// ------------------------------------------

// Index is interface for handling the mapping of an Value: V to an List-Index: LI
// The Value V comes from a func(*OBJ) V
type Index[OBJ any] interface {
	// Set insert or update the value of the given OBJ and the associated list index
	Set(*OBJ, uint32)
	// BulkSet inserts a bulk of given OBJ and the associated list index
	BulkSet(iter.Seq2[int, *OBJ])
	// UnSet remove the list index of the given OBJ
	UnSet(*OBJ, uint32)
	// HasChanged check for an old and an new Item OBJ value
	HasChanged(oldItem, newItem *OBJ) bool
	// Filter is quering the Index
	Filter
}

// Filter returns the RawIDs or an error by a given Relation and Value
type Filter interface {
	// Equal is seperated from Match
	// because the RawIDs result you can NOT mutable
	Equal(value any) (*RawIDs32, error)
	// Match execute a query (FilterOP) with one given value
	// for example: age > 18
	Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, error)
	// MatchMany execute a query (FilterOp) for many given values
	// for example: age between 18 and 80
	MatchMany(op FilterOp, values ...any) (*RawIDs32, error)
}

var (
	FOpEq         = FilterOp{Op: OpEq}
	FOpNeq        = FilterOp{Op: OpNeq}
	FOpLe         = FilterOp{Op: OpLe}
	FOpLt         = FilterOp{Op: OpLt}
	FOpGe         = FilterOp{Op: OpGe}
	FOpGt         = FilterOp{Op: OpGt}
	FOpIn         = FilterOp{Op: OpIn}
	FOpBetween    = FilterOp{Op: OpBetween}
	FOpContains   = FilterOp{Op: OpContains}
	FOpStartsWith = FilterOp{Op: OpStartsWith}
)

// ValueHandler is a Strategy Pattern implementation that acts as an abstraction layer
// between your Index and your Objects

// Its primary purpose is to decouple the index logic from the structure of the fields it is indexing.
// It allows one single index implementation to support both single-value fields (like Age int) and
// multi-value fields (like Tags []string) without changing the index's internal code.
type ValueHandler[OBJ any, V any] interface {
	Handle(obj *OBJ, exec func(value V))
	HasChanged(oldItem, newItem *OBJ) bool
	CanInvert() bool
}

type SingleValueHandler[OBJ any, V comparable] struct {
	fieldGetFn FromField[OBJ, V]
}

func (h SingleValueHandler[OBJ, V]) Handle(obj *OBJ, exec func(V)) { exec(h.fieldGetFn(obj)) }
func (h SingleValueHandler[OBJ, V]) HasChanged(oldItem, newItem *OBJ) bool {
	return h.fieldGetFn(oldItem) != h.fieldGetFn(newItem)
}
func (h SingleValueHandler[OBJ, V]) CanInvert() bool { return true }

type MultiValueHandler[OBJ any, V comparable] struct {
	fieldGetFn FromFieldSlice[OBJ, V]
}

func (h MultiValueHandler[OBJ, V]) Handle(obj *OBJ, exec func(V)) {
	values := h.fieldGetFn(obj)
	for _, value := range values {
		exec(value)
	}
}

func (h MultiValueHandler[OBJ, V]) HasChanged(oldItem, newItem *OBJ) bool {
	ov := h.fieldGetFn(oldItem)
	nv := h.fieldGetFn(newItem)

	if len(ov) != len(nv) {
		return true
	}

	// ignored the order of old and new items
	for i, v := range ov {
		if v != nv[i] {
			return true
		}
	}

	return false
}
func (h MultiValueHandler[OBJ, V]) CanInvert() bool { return false }

// FilterOp is a wrapper over the Op, which contains the Op and a String.
// For User defined FilterOp is no Op defined, so the User defined Index can use the String.
type FilterOp struct {
	Op   Op
	Name string
}

func (f FilterOp) String() string {
	if f.Name != "" {
		return f.Name
	}
	return f.Op.String()

}

const MapIndexName = "MapIndex"

// MapIndex is a mapping of any value to the Index in the List.
// This index only supported Queries with the Equal Ralation!
type MapIndex[OBJ any, V comparable, H ValueHandler[OBJ, V]] struct {
	data         map[V]*RawIDs32
	valueHandler H
}

func NewMapIndex[OBJ any, V comparable](fieldGetFn FromField[OBJ, V]) Index[OBJ] {
	return &MapIndex[OBJ, V, SingleValueHandler[OBJ, V]]{
		data:         make(map[V]*RawIDs32),
		valueHandler: SingleValueHandler[OBJ, V]{fieldGetFn},
	}
}

func NewMapIndexSlice[OBJ any, V comparable](fieldGetFn FromFieldSlice[OBJ, V]) Index[OBJ] {
	return &MapIndex[OBJ, V, MultiValueHandler[OBJ, V]]{
		data:         make(map[V]*RawIDs32),
		valueHandler: MultiValueHandler[OBJ, V]{fieldGetFn},
	}
}

func (mi *MapIndex[OBJ, V, H]) Set(obj *OBJ, lidx uint32) {
	mi.valueHandler.Handle(obj, func(value V) {
		ids, found := mi.data[value]
		if !found {
			ids = NewRawIDs[uint32]()
			mi.data[value] = ids
		}

		ids.Set(lidx)
	})
}

func (mi *MapIndex[OBJ, V, H]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	// group the IDs by their indexed value locally
	batch := make(map[V][]uint32)
	for i, obj := range objs {
		mi.valueHandler.Handle(obj, func(value V) {
			batch[value] = append(batch[value], uint32(i))
		})
	}

	// merge the grouped batches into the main index
	for val, ids := range batch {
		mi.data[val] = NewRawIDsFrom(ids...)
	}
}

func (mi *MapIndex[OBJ, V, H]) UnSet(obj *OBJ, lidx uint32) {
	mi.valueHandler.Handle(obj, func(value V) {
		if ids, found := mi.data[value]; found {
			ids.UnSet(lidx)
			if ids.Count() == 0 {
				delete(mi.data, value)
			}
		}
	})
}

func (mi *MapIndex[OBJ, V, H]) HasChanged(oldItem, newItem *OBJ) bool {
	return mi.valueHandler.HasChanged(oldItem, newItem)
}

func (mi *MapIndex[OBJ, V, H]) Equal(value any) (*RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	ids, found := mi.data[v]
	if !found {
		return NewRawIDs[uint32](), nil
	}

	return ids, nil
}

func (mi *MapIndex[OBJ, V, H]) Match(_ *RawIDs32, op FilterOp, _ any) (*RawIDs32, error) {
	return nil, InvalidOperationError{MapIndexName, op.Op}
}

// MatchMany is not supported by MapIndex, so that always returns an error
func (mi *MapIndex[OBJ, V, H]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
	switch op.Op {
	case OpIn:
		// fast path for 0 or 1 values
		switch len(values) {
		case 0:
			return NewRawIDs[uint32](), nil
		case 1:
			key, err := ValueFromAny[V](values[0])
			if err != nil {
				return nil, err
			}
			if rid, found := mi.data[key]; found {
				return rid.Copy(), nil
			}
			return NewRawIDs[uint32](), nil
		}

		matched := make([]*RawIDs32, 0, len(values))
		var maxLen int

		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, err
			}

			if rid, found := mi.data[key]; found {
				matched = append(matched, rid)
				rcount := rid.Len()
				if rcount > maxLen {
					maxLen = rcount
				}
			}
		}

		// fast path for 0 or 1 matches
		switch len(matched) {
		case 0:
			return NewRawIDs[uint32](), nil
		case 1:
			return matched[0].Copy(), nil
		}

		result := NewRawIDsWithCapacity[uint32](maxLen)
		for _, bs := range matched {
			result.Or(bs)
		}

		return result, nil
	default:
		return nil, InvalidOperationError{MapIndexName, op.Op}
	}
}

const SortedIndexName = "SortedIndex"

// SortedIndex is well suited for Queries with: Range, Min, Max, Greater and Less
type SortedIndex[OBJ any, V cmp.Ordered, H ValueHandler[OBJ, V]] struct {
	skipList     SkipList[V, *RawIDs32]
	valueHandler H
}

func NewSortedIndex[OBJ any, V cmp.Ordered](fieldGetFn FromField[OBJ, V]) Index[OBJ] {
	return &SortedIndex[OBJ, V, SingleValueHandler[OBJ, V]]{
		skipList:     NewSkipList[V, *RawIDs32](),
		valueHandler: SingleValueHandler[OBJ, V]{fieldGetFn},
	}
}

func NewSortedIndexSlice[OBJ any, V cmp.Ordered](fieldGetFn FromFieldSlice[OBJ, V]) Index[OBJ] {
	return &SortedIndex[OBJ, V, MultiValueHandler[OBJ, V]]{
		skipList:     NewSkipList[V, *RawIDs32](),
		valueHandler: MultiValueHandler[OBJ, V]{fieldGetFn},
	}
}

func (si *SortedIndex[OBJ, V, H]) Set(obj *OBJ, lidx uint32) {
	si.valueHandler.Handle(obj, func(value V) {
		ids, found := si.skipList.Get(value)
		if !found {
			ids = NewRawIDs[uint32]()
			si.skipList.Put(value, ids)
		}

		ids.Set(lidx)
	})
}

func (si *SortedIndex[OBJ, V, H]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	// group the IDs locally
	batch := make(map[V][]uint32)
	for i, obj := range objs {
		si.valueHandler.Handle(obj, func(value V) {
			batch[value] = append(batch[value], uint32(i))
		})
	}

	// merge into the SkipList
	for val, ids := range batch {
		si.skipList.Put(val, NewRawIDsFrom(ids...))
	}
}

func (si *SortedIndex[OBJ, V, H]) UnSet(obj *OBJ, lidx uint32) {
	si.valueHandler.Handle(obj, func(value V) {
		if ids, found := si.skipList.Get(value); found {
			ids.UnSet(lidx)
			if ids.Count() == 0 {
				si.skipList.Delete(value)
			}
		}
	})
}

func (si *SortedIndex[OBJ, V, H]) HasChanged(oldItem, newItem *OBJ) bool {
	return si.valueHandler.HasChanged(oldItem, newItem)
}

func (si *SortedIndex[OBJ, V, H]) Equal(value any) (*RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	ids, found := si.skipList.Get(v)
	if !found {
		return NewRawIDs[uint32](), nil
	}

	return ids, nil
}

func (si *SortedIndex[OBJ, V, H]) Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	var toMerge []*RawIDs32
	var visitor func(V, *RawIDs32) bool
	abortedForInversion := false

	if si.valueHandler.CanInvert() {
		// The Invertible Visitor (Counts and aborts)
		count := 0
		halfwayMark := si.skipList.Len() / 2

		visitor = func(_ V, bs *RawIDs32) bool {
			count++
			if count > halfwayMark {
				abortedForInversion = true
				return false // Abort traversal
			}
			toMerge = append(toMerge, bs)
			return true
		}
	} else {
		// The Lean & Mean Visitor (No counting, no branching!)
		visitor = func(_ V, bs *RawIDs32) bool {
			toMerge = append(toMerge, bs)
			return true // Always keep going
		}
	}

	var invOp FilterOp
	switch op.Op {
	case OpLt:
		invOp = FilterOp{Op: OpGe}
		si.skipList.Less(v, visitor)
	case OpLe:
		invOp = FilterOp{Op: OpGt}
		si.skipList.LessEqual(v, visitor)
	case OpGt:
		invOp = FilterOp{Op: OpLe}
		si.skipList.Greater(v, visitor)
	case OpGe:
		invOp = FilterOp{Op: OpLt}
		si.skipList.GreaterEqual(v, visitor)
	default:
		return nil, InvalidOperationError{SortedIndexName, op.Op}
	}

	// query inversion optimization
	if abortedForInversion {
		inverseResult, err := si.Match(allIDs, invOp, value)
		if err != nil {
			return nil, err
		}

		// finalResult = allIDs - inverseResult
		finalResult := allIDs.Copy()
		finalResult.AndNot(inverseResult)
		return finalResult, nil
	}

	result := NewRawIDs[uint32]()
	for _, ids := range toMerge {
		result.Or(ids)
	}
	return result, nil
}

func (si *SortedIndex[OBJ, V, H]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
	switch op.Op {
	case OpBetween:
		if len(values) != 2 {
			return nil, InvalidArgsLenError{Defined: "2", Got: len(values)}
		}

		min, err := ValueFromAny[V](values[0])
		if err != nil {
			return nil, InvalidValueTypeError[V]{values[0]}
		}
		max, err := ValueFromAny[V](values[1])
		if err != nil {
			return nil, InvalidValueTypeError[V]{values[1]}
		}

		result := NewRawIDs[uint32]()
		si.skipList.Range(min, max, func(_ V, bs *RawIDs32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpIn:
		// fast path for 0 or 1 values
		switch len(values) {
		case 0:
			return NewRawIDs[uint32](), nil
		case 1:
			key, err := ValueFromAny[V](values[0])
			if err != nil {
				return nil, err
			}
			if rid, found := si.skipList.Get(key); found {
				return rid.Copy(), nil
			}
			return NewRawIDs[uint32](), nil
		}

		matched := make([]*RawIDs32, 0, len(values))
		var maxLen int

		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, err
			}

			if rid, found := si.skipList.Get(key); found {
				matched = append(matched, rid)
				rcount := rid.Len()
				if rcount > maxLen {
					maxLen = rcount
				}
			}
		}

		// fast path for 0 or 1 matches
		switch len(matched) {
		case 0:
			return NewRawIDs[uint32](), nil
		case 1:
			return matched[0].Copy(), nil
		}

		result := NewRawIDsWithCapacity[uint32](maxLen)
		for _, bs := range matched {
			result.Or(bs)
		}

		return result, nil
	default:
		return nil, InvalidOperationError{SortedIndexName, op.Op}
	}
}

const RangeIndexName = "RangeIndex"

type RangeIndex[OBJ any, H ValueHandler[OBJ, uint8]] struct {
	data [256]*RawIDs32
	// the length of the data (the max value)
	// max can be: 256 if the data is full from 0-255
	max          int
	valueHandler H
}

func NewRangeIndex[OBJ any](fieldGetFn FromField[OBJ, uint8]) Index[OBJ] {
	return &RangeIndex[OBJ, SingleValueHandler[OBJ, uint8]]{
		// Array size must be 256 to cover indices 0-255
		data:         [256]*RawIDs32{},
		valueHandler: SingleValueHandler[OBJ, uint8]{fieldGetFn},
	}
}

func NewRangeIndexSlice[OBJ any](fieldGetFn FromFieldSlice[OBJ, uint8]) Index[OBJ] {
	return &RangeIndex[OBJ, MultiValueHandler[OBJ, uint8]]{
		// Array size must be 256 to cover indices 0-255
		data:         [256]*RawIDs32{},
		valueHandler: MultiValueHandler[OBJ, uint8]{fieldGetFn},
	}
}

func (ri *RangeIndex[OBJ, H]) Set(obj *OBJ, lidx uint32) {
	ri.valueHandler.Handle(obj, func(value uint8) {
		valInt := int(value)

		ids := ri.data[valInt]
		if ids == nil {
			ids = NewRawIDs[uint32]()
			ri.data[valInt] = ids
		}
		ids.Set(lidx)

		// new max value, if value greater the old max value
		if ri.max < valInt+1 {
			ri.max = valInt + 1
		}
	})
}

func (ri *RangeIndex[OBJ, H]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	for i, obj := range objs {
		lidx := uint32(i)
		ri.valueHandler.Handle(obj, func(value uint8) {
			ids := ri.data[value]
			if ids == nil {
				ids = NewRawIDs[uint32]()
				ri.data[value] = ids
			}
			ids.Set(lidx)

			// new max value, if value greater the old max value
			if ri.max < int(value)+1 {
				ri.max = int(value + 1)
			}
		})
	}
}

func (ri *RangeIndex[OBJ, H]) UnSet(obj *OBJ, lidx uint32) {
	ri.valueHandler.Handle(obj, func(value uint8) {
		valInt := int(value)

		ids := ri.data[valInt]
		if ids == nil {
			return
		}
		ids.UnSet(lidx)

		if ids.IsEmpty() {
			ri.data[valInt] = nil

			// if is empty, calculate the new max value
			if ri.max == valInt+1 {
				ri.max = 0 // default fallback
				for i := valInt - 1; i >= 0; i-- {
					if ri.data[i] != nil && !ri.data[i].IsEmpty() {
						ri.max = i + 1
						break
					}
				}
			}
		}
	})
}

func (ri *RangeIndex[OBJ, H]) HasChanged(oldItem, newItem *OBJ) bool {
	return ri.valueHandler.HasChanged(oldItem, newItem)
}

func (ri *RangeIndex[OBJ, H]) Equal(value any) (*RawIDs32, error) {
	v, err := ValueFromAny[uint8](value)
	if err != nil {
		return nil, InvalidValueTypeError[uint8]{value}
	}

	ids := ri.data[v]
	if ids == nil {
		return NewRawIDs[uint32](), nil
	}

	return ids, nil
}

func (ri *RangeIndex[OBJ, H]) Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, error) {
	v, err := ValueFromAny[uint8](value)
	if err != nil {
		return nil, InvalidValueTypeError[uint8]{value}
	}
	valInt := int(v)

	// Define the Range Bounds
	start, end := 0, ri.max
	var invOp FilterOp

	switch op.Op {
	case OpLt:
		end = valInt
		invOp = FilterOp{Op: OpGe}
	case OpLe:
		end = valInt + 1
		invOp = FilterOp{Op: OpGt}
	case OpGt:
		start = valInt + 1
		invOp = FilterOp{Op: OpLe}
	case OpGe:
		start = valInt
		invOp = FilterOp{Op: OpLt}
	default:
		return nil, InvalidOperationError{RangeIndexName, op.Op}
	}

	if end > ri.max {
		end = ri.max
	}
	if start >= end {
		return NewRawIDs[uint32](), nil
	}

	// Query Inversion Optimization
	// If the range we are scanning is more than half of our total active data range,
	// it's cheaper to get the inverse and subtract it from allIDs.
	if ri.valueHandler.CanInvert() && end-start > (ri.max/2) {
		// calculate the IDs we DON'T want
		inverseResult, err := ri.Match(allIDs, invOp, value)
		if err != nil {
			return nil, err
		}

		// result = allIDs - inverseResult
		finalResult := allIDs.Copy()
		finalResult.AndNot(inverseResult)
		return finalResult, nil
	}

	result := NewRawIDs[uint32]()
	for i := start; i < end; i++ {
		data := ri.data[i]
		if data != nil && !data.IsEmpty() {
			result.Or(data)
		}
	}

	return result, nil
}

func (ri *RangeIndex[OBJ, H]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
	switch op.Op {
	case OpBetween:
		if len(values) != 2 {
			return nil, InvalidArgsLenError{Defined: "2", Got: len(values)}
		}

		minVal, err := ValueFromAny[uint8](values[0])
		if err != nil {
			return nil, InvalidValueTypeError[uint8]{values[0]}
		}
		maxVal, err := ValueFromAny[uint8](values[1])
		if err != nil {
			return nil, InvalidValueTypeError[uint8]{values[1]}
		}

		// Use ints to prevent infinite loop on maxVal == 255
		min, max := int(minVal), int(maxVal)

		result := NewRawIDs[uint32]()
		for i := min; i <= max; i++ {
			if i >= ri.max {
				break
			}
			if ri.data[i] != nil && !ri.data[i].IsEmpty() {
				result.Or(ri.data[i])
			}
		}
		return result, nil
	case OpIn:
		result := NewRawIDs[uint32]()
		for _, v := range values {
			i, err := ValueFromAny[uint8](v)
			if err != nil {
				return nil, err
			}

			valInt := int(i)
			if valInt < ri.max && ri.data[valInt] != nil && !ri.data[valInt].IsEmpty() {
				result.Or(ri.data[i])
			}
		}
		return result, nil

	default:
		return nil, InvalidOperationError{RangeIndexName, op.Op}
	}
}

const StringIndexName = "StringIndex"

type StringIndex[OBJ any] struct {
	trigram     TrigramIndex
	sortedIndex SortedIndex[OBJ, string, SingleValueHandler[OBJ, string]]
}

func NewStringIndex[OBJ any](fromField FromField[OBJ, string]) Index[OBJ] {
	return &StringIndex[OBJ]{
		trigram:     NewTrigramIndex(),
		sortedIndex: *NewSortedIndex[OBJ, string](fromField).(*SortedIndex[OBJ, string, SingleValueHandler[OBJ, string]]),
	}
}

func (ti *StringIndex[OBJ]) Set(obj *OBJ, lidx uint32) {
	ti.sortedIndex.Set(obj, lidx)
	ti.sortedIndex.valueHandler.Handle(obj, func(value string) {
		ti.trigram.Put(value, int(lidx))
	})
}

func (ti *StringIndex[OBJ]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	ti.sortedIndex.BulkSet(objs)
	TrigramIndexBulkPut(&ti.trigram, ti.sortedIndex.valueHandler, objs)
}

func (ti *StringIndex[OBJ]) UnSet(obj *OBJ, lidx uint32) {
	ti.sortedIndex.UnSet(obj, lidx)
	ti.trigram.Delete(int(lidx))
}

func (ti *StringIndex[OBJ]) HasChanged(oldItem, newItem *OBJ) bool {
	return ti.sortedIndex.HasChanged(oldItem, newItem)
}

func (ti *StringIndex[OBJ]) Equal(value any) (*RawIDs32, error) {
	return ti.sortedIndex.Equal(value)
}

func (ti *StringIndex[OBJ]) Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, error) {

	switch op.Op {
	case OpLt, OpLe, OpGt, OpGe:
		return ti.sortedIndex.Match(allIDs, op, value)
	}

	s, err := ValueFromAny[string](value)
	if err != nil {
		return nil, InvalidValueTypeError[string]{value}
	}

	switch op.Op {
	case OpStartsWith:
		result := NewRawIDs[uint32]()
		ti.sortedIndex.skipList.StringStartsWith(s, func(_ string, ids *RawIDs32) bool {
			result.Or(ids)
			return true
		})
		return result, nil
	case OpContains:
		return ti.trigram.Get(s).Copy(), nil

	default:
		return nil, InvalidOperationError{StringIndexName, op.Op}
	}
}

func (ti *StringIndex[OBJ]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
	return ti.sortedIndex.MatchMany(op, values...)
}

// ParserExt is a FIlter/Index extension for parsing the given string to an from Filter useable value.
// For example for a given date-string to an unix-second-time
// Or to convert a given Enum to an int Value
type ParserExt[OBJ any] struct {
	inner Index[OBJ]
	parse func(string) any
}

func NewParserExt[OBJ any](index Index[OBJ], parse func(string) any) Index[OBJ] {
	return &ParserExt[OBJ]{inner: index, parse: parse}
}

func (p *ParserExt[OBJ]) Set(obj *OBJ, lidx uint32)         { p.inner.Set(obj, lidx) }
func (p *ParserExt[OBJ]) BulkSet(objs iter.Seq2[int, *OBJ]) { p.inner.BulkSet(objs) }
func (p *ParserExt[OBJ]) UnSet(obj *OBJ, lidx uint32)       { p.inner.UnSet(obj, lidx) }
func (p *ParserExt[OBJ]) HasChanged(oldItem, newItem *OBJ) bool {
	return p.inner.HasChanged(oldItem, newItem)
}

func (p *ParserExt[OBJ]) Equal(value any) (*RawIDs32, error) {
	if s, ok := value.(string); ok {
		return p.inner.Equal(p.parse(s))
	}

	return nil, InvalidValueTypeError[string]{value}
}

func (p *ParserExt[OBJ]) Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, error) {
	if s, ok := value.(string); ok {
		return p.inner.Match(allIDs, op, p.parse(s))
	}

	return nil, InvalidValueTypeError[string]{value}
}

func (p *ParserExt[OBJ]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
	pvalues := make([]any, len(values))
	for i, v := range values {
		s, ok := v.(string)
		if !ok {
			return nil, InvalidValueTypeError[string]{v}
		}
		pvalues[i] = p.parse(s)

	}
	return p.inner.MatchMany(op, pvalues...)
}
