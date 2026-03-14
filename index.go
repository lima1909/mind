package mind

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

const IDIndexFieldName = "id"

// fieldIndexMap maps a given field name to an Index
type indexMap[OBJ any, ID comparable] struct {
	idIndex idIndex[OBJ, ID]
	index   map[string]Index[OBJ]
	allIDs  *BitSet[uint32]
}

func newIndexMap[OBJ any, ID comparable](idIndex idIndex[OBJ, ID]) indexMap[OBJ, ID] {
	return indexMap[OBJ, ID]{
		idIndex: idIndex,
		index:   make(map[string]Index[OBJ]),
		allIDs:  NewBitSet[uint32](),
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

// Set add to all known indexes synchron the new value (including ID-index)
func (i indexMap[OBJ, ID]) Set(obj *OBJ, idx int) {
	if i.idIndex != nil {
		i.idIndex.Set(obj, idx)
	}

	uidx := uint32(idx)
	i.allIDs.Set(uidx)
	for _, fieldIndex := range i.index {
		fieldIndex.Set(obj, uidx)
	}
}

// UnSet remove all known indexes synchron the new value (including ID-index)
func (i indexMap[OBJ, ID]) UnSet(obj *OBJ, idx int) {
	if i.idIndex != nil {
		i.idIndex.UnSet(obj, idx)
	}

	uidx := uint32(idx)
	i.allIDs.UnSet(uidx)

	for _, fieldIndex := range i.index {
		fieldIndex.UnSet(obj, uidx)
	}
}

// ReIndex update all known indexes synchron the new value (including ID-index)
func (i indexMap[OBJ, ID]) ReIndex(oldObj, newObj *OBJ, idx int) {
	if i.idIndex != nil {
		i.idIndex.UnSet(oldObj, idx)
		i.idIndex.Set(newObj, idx)
	}

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

func (i indexMap[OBJ, ID]) getIndexByID(id ID) (int, error) {
	if i.idIndex == nil {
		return 0, NoIdIndexDefinedError{}
	}

	return i.idIndex.GetIndex(id)
}

func (i indexMap[OBJ, ID]) getIDByItem(item *OBJ) (ID, int, error) {
	if i.idIndex == nil {
		var id ID
		return id, 0, NoIdIndexDefinedError{}
	}

	return i.idIndex.GetID(item)
}

type idIndex[OBJ any, ID comparable] interface {
	Set(*OBJ, int)
	UnSet(*OBJ, int)
	GetIndex(ID) (int, error)
	GetID(*OBJ) (ID, int, error)
	Filter
}

const IDAutoIncName = "IDAutoIncIndex"

type idAutoIncIndex[OBJ any] struct {
	idCounter uint64
	count2idx map[uint64]int
	idx2count map[int]uint64
}

func newIDAutoIncIndex[OBJ any]() idIndex[OBJ, uint64] {
	return &idAutoIncIndex[OBJ]{
		count2idx: make(map[uint64]int),
		idx2count: make(map[int]uint64),
	}
}

func (id *idAutoIncIndex[OBJ]) Set(_ *OBJ, lidx int) {
	id.idCounter++
	id.count2idx[id.idCounter] = lidx
	id.idx2count[lidx] = id.idCounter
}

func (id *idAutoIncIndex[OBJ]) UnSet(_ *OBJ, lidx int) {
	counter := id.idx2count[lidx]
	id.count2idx[counter] = -1
	id.idx2count[lidx] = 0
}

func (id *idAutoIncIndex[OBJ]) GetIndex(i uint64) (int, error) {
	if lidx, found := id.count2idx[i]; found && lidx >= 0 {
		return lidx, nil
	}

	return 0, ValueNotFoundError{i}
}

func (id *idAutoIncIndex[OBJ]) GetID(*OBJ) (uint64, int, error) {
	return 0, -1, errors.New("GedID is not supported for this Index")
}

func (id *idAutoIncIndex[OBJ]) Match(op FilterOp, value any) (*BitSet[uint32], error) {
	i, ok := value.(uint64)
	if !ok {
		return nil, InvalidValueTypeError[uint32]{value}
	}

	if op.Op != OpEq {
		return nil, InvalidOperationError{IDAutoIncName, op.Op}
	}

	idx, err := id.GetIndex(i)
	if err != nil {
		return nil, err
	}

	return NewBitSetFrom(uint32(idx)), nil

}

// MatchMany is not supported by idAutoIncIndex, so that always returns an error
func (id *idAutoIncIndex[OBJ]) MatchMany(op FilterOp, values ...any) (*BitSet[uint32], error) {
	return nil, InvalidOperationError{IDAutoIncName, op.Op}
}

const IDMapIndexName = "IDMapIndex"

type idMapIndex[OBJ any, ID comparable] struct {
	data       map[ID]int
	fieldGetFn FromField[OBJ, ID]
}

func newIDMapIndex[OBJ any, ID comparable](fieldGetFn FromField[OBJ, ID]) idIndex[OBJ, ID] {
	return &idMapIndex[OBJ, ID]{
		data:       make(map[ID]int),
		fieldGetFn: fieldGetFn,
	}
}

func (mi *idMapIndex[OBJ, ID]) Set(obj *OBJ, lidx int) {
	id := mi.fieldGetFn(obj)
	mi.data[id] = lidx
}

func (mi *idMapIndex[OBJ, ID]) UnSet(obj *OBJ, lidx int) {
	id := mi.fieldGetFn(obj)
	delete(mi.data, id)
}

func (mi *idMapIndex[OBJ, ID]) GetIndex(id ID) (int, error) {
	if lidx, found := mi.data[id]; found {
		return lidx, nil
	}

	return 0, ValueNotFoundError{id}
}

func (mi *idMapIndex[OBJ, ID]) GetID(item *OBJ) (ID, int, error) {
	id := mi.fieldGetFn(item)
	if lidx, found := mi.data[id]; found {
		return id, lidx, nil
	}

	var null ID
	return null, 0, ValueNotFoundError{id}
}

func (mi *idMapIndex[OBJ, ID]) Match(op FilterOp, value any) (*BitSet[uint32], error) {
	id, ok := value.(ID)
	if !ok {
		return nil, InvalidValueTypeError[ID]{value}
	}

	if op.Op != OpEq {
		return nil, InvalidOperationError{IDMapIndexName, op.Op}
	}

	idx, err := mi.GetIndex(id)
	if err != nil {
		return nil, err
	}

	return NewBitSetFrom(uint32(idx)), nil

}

// MatchMany is not supported by idMapIndex, so that always returns an error
func (mi *idMapIndex[OBJ, ID]) MatchMany(op FilterOp, values ...any) (*BitSet[uint32], error) {
	return nil, InvalidOperationError{IDMapIndexName, op.Op}
}

// ------------------------------------------
// here starts the Index with the Index impls
// ------------------------------------------

// Index is interface for handling the mapping of an Value: V to an List-Index: LI
// The Value V comes from a func(*OBJ) V
type Index[OBJ any] interface {
	Set(*OBJ, uint32)
	UnSet(*OBJ, uint32)
	HasChanged(oldItem, newItem *OBJ) bool
	Filter
}

var (
	FOpEq         = FilterOp{Op: OpEq}
	FOpLe         = FilterOp{Op: OpLe}
	FOpLt         = FilterOp{Op: OpLt}
	FOpGe         = FilterOp{Op: OpGe}
	FOpGt         = FilterOp{Op: OpGt}
	FOpIn         = FilterOp{Op: OpIn}
	FOpBetween    = FilterOp{Op: OpBetween}
	FOpStartsWith = FilterOp{String: "startswith"}
)

// FilterOp is a wrapper over the Op, which contains the Op and a String.
// For User defined FilterOp is no Op defined, so the User defined Index can use the String.
type FilterOp struct {
	Op     Op
	String string
}

// Filter returns the BitSet or an error by a given Relation and Value
type Filter interface {
	Match(op FilterOp, value any) (*BitSet32, error)
	MatchMany(op FilterOp, values ...any) (*BitSet32, error)
}

// FromField is a function, which returns a value from an given object.
// example:
// Person{name string}
// func (p *Person) Name() { return p.name }
// (*Person).Name is the FieldGetFn
type FromField[OBJ any, V any] = func(*OBJ) V

// FromValue returns a Getter that simply returns the value itself.
// Use this when your list contains the raw values you want to index.
func FromValue[V any]() FromField[V, V] { return func(v *V) V { return *v } }

// FromName returns per reflection the propery (field) value from the given object.
func FromName[OBJ any, V any](fieldName string) FromField[OBJ, V] {
	var zero OBJ
	typ := reflect.TypeOf(zero)
	isPtr := false
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		isPtr = true
	}

	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("expected struct, got %s", typ.Kind()))
	}

	field, ok := typ.FieldByName(fieldName)
	if !ok {
		panic(fmt.Sprintf("field %s not found", fieldName))
	}
	// reflection cannot access lowercase (unexported) fields via .Interface()
	// unless we use unsafe, but let's stick to standard safety checks at setup time.
	// Actually, unsafe access works on unexported fields too, but usually discouraged.
	// But let's fail as per original behavior.
	if !field.IsExported() {
		panic(fmt.Sprintf("field %s is unexported", fieldName))
	}

	offset := field.Offset

	if isPtr {
		// OBJ is *Struct. input is **Struct.
		return func(obj *OBJ) V {
			// *obj is the *Struct.
			// We need unsafe.Pointer(*obj) + offset
			structPtr := *(**unsafe.Pointer)(unsafe.Pointer(obj))
			if structPtr == nil {
				var zero V
				return zero // Or panic? Original reflect would panic on nil pointer deref usually.
			}
			return *(*V)(unsafe.Add(*structPtr, offset))
		}
	}

	// OBJ is Struct. input is *Struct.
	return func(obj *OBJ) V {
		// obj is *Struct
		return *(*V)(unsafe.Add(unsafe.Pointer(obj), offset))
	}
}

const MapIndexName = "MapIndex"

// MapIndex is a mapping of any value to the Index in the List.
// This index only supported Queries with the Equal Ralation!
type MapIndex[OBJ any, V comparable] struct {
	data       map[any]*BitSet32
	fieldGetFn FromField[OBJ, V]
}

func NewMapIndex[OBJ any, V comparable](fromField FromField[OBJ, V]) Index[OBJ] {
	return &MapIndex[OBJ, V]{
		data:       make(map[any]*BitSet32),
		fieldGetFn: fromField,
	}
}

func (mi *MapIndex[OBJ, V]) Set(obj *OBJ, lidx uint32) {
	value := mi.fieldGetFn(obj)
	bs, found := mi.data[value]
	if !found {
		bs = NewEmptyBitSet[uint32]()
	}
	bs.Set(lidx)
	mi.data[value] = bs
}

func (mi *MapIndex[OBJ, V]) UnSet(obj *OBJ, lidx uint32) {
	value := mi.fieldGetFn(obj)
	if bs, found := mi.data[value]; found {
		bs.UnSet(lidx)
		if bs.Count() == 0 {
			delete(mi.data, value)
		}
	}
}

func (mi *MapIndex[OBJ, V]) HasChanged(oldItem, newItem *OBJ) bool {
	return mi.fieldGetFn(oldItem) != mi.fieldGetFn(newItem)
}

func (mi *MapIndex[OBJ, V]) Match(op FilterOp, value any) (*BitSet32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	if op.Op != OpEq {
		return nil, InvalidOperationError{MapIndexName, op.Op}
	}

	bs, found := mi.data[v]
	if !found {
		return NewEmptyBitSet[uint32](), nil
	}

	return bs, nil
}

// MatchMany is not supported by MapIndex, so that always returns an error
func (mi *MapIndex[OBJ, V]) MatchMany(op FilterOp, values ...any) (*BitSet32, error) {
	switch op.Op {
	case OpIn:
		if len(values) == 0 {
			return NewEmptyBitSet[uint32](), nil
		}

		matched := make([]*BitSet32, 0, len(values))
		var maxLen int

		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, err
			}

			if bs, found := mi.data[key]; found {
				matched = append(matched, bs)
				if len(bs.data) > maxLen {
					maxLen = len(bs.data)
				}
			}
		}

		if len(matched) == 0 {
			return NewEmptyBitSet[uint32](), nil
		}

		result := NewBitSetWithCapacity[uint32](maxLen)
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
type SortedIndex[OBJ any, V cmp.Ordered] struct {
	skipList   SkipList[V, *BitSet32]
	fieldGetFn FromField[OBJ, V]
}

func NewSortedIndex[OBJ any, V cmp.Ordered](fieldGetFn FromField[OBJ, V]) Index[OBJ] {
	return &SortedIndex[OBJ, V]{
		skipList:   NewSkipList[V, *BitSet32](),
		fieldGetFn: fieldGetFn,
	}
}

func (si *SortedIndex[OBJ, V]) Set(obj *OBJ, lidx uint32) {
	value := si.fieldGetFn(obj)
	bs, found := si.skipList.Get(value)
	if !found {
		bs = NewEmptyBitSet[uint32]()
	}
	bs.Set(lidx)
	si.skipList.Put(value, bs)
}

func (si *SortedIndex[OBJ, V]) UnSet(obj *OBJ, lidx uint32) {
	value := si.fieldGetFn(obj)
	if bs, found := si.skipList.Get(value); found {
		bs.UnSet(lidx)
		if bs.Count() == 0 {
			si.skipList.Delete(value)
		}
	}
}

func (si *SortedIndex[OBJ, V]) HasChanged(oldItem, newItem *OBJ) bool {
	return si.fieldGetFn(oldItem) != si.fieldGetFn(newItem)
}

func (si *SortedIndex[OBJ, V]) Match(op FilterOp, value any) (*BitSet32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	switch op.Op {
	case OpEq:
		if bs, found := si.skipList.Get(v); found {
			return bs, nil
		}
		return NewEmptyBitSet[uint32](), nil
	case OpLt:
		result := NewBitSet[uint32]()
		si.skipList.Less(v, func(v V, bs *BitSet32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpLe:
		result := NewBitSet[uint32]()
		si.skipList.LessEqual(v, func(_ V, bs *BitSet32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpGt:
		result := NewBitSet[uint32]()
		si.skipList.Greater(v, func(v V, bs *BitSet32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpGe:
		result := NewBitSet[uint32]()
		si.skipList.GreaterEqual(v, func(_ V, bs *BitSet32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	default:
		if op.String == FOpStartsWith.String {
			if _, ok := value.(string); !ok {
				return nil, InvalidValueTypeError[string]{value}
			}

			result := NewBitSet[uint32]()
			si.skipList.StringStartsWith(v, func(_ V, bs *BitSet32) bool {
				result.Or(bs)
				return true
			})
			return result, nil
		}

		return nil, InvalidOperationError{SortedIndexName, op.Op}
	}
}

func (si *SortedIndex[OBJ, V]) MatchMany(op FilterOp, values ...any) (*BitSet32, error) {
	switch op.Op {
	case OpBetween:
		if len(values) != 2 {
			return nil, InvalidArgsLenError{defined: "2", got: len(values)}
		}

		min, err := ValueFromAny[V](values[0])
		if err != nil {
			return nil, InvalidValueTypeError[V]{values[0]}
		}
		max, err := ValueFromAny[V](values[1])
		if err != nil {
			return nil, InvalidValueTypeError[V]{values[1]}
		}

		result := NewBitSet[uint32]()
		si.skipList.Range(min, max, func(_ V, bs *BitSet32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpIn:
		if len(values) == 0 {
			return NewEmptyBitSet[uint32](), nil
		}

		matched := make([]*BitSet32, 0, len(values))
		var maxLen int

		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, err
			}

			if bs, found := si.skipList.Get(key); found {
				matched = append(matched, bs)
				if len(bs.data) > maxLen {
					maxLen = len(bs.data)
				}
			}
		}

		if len(matched) == 0 {
			return NewEmptyBitSet[uint32](), nil
		}

		result := NewBitSetWithCapacity[uint32](maxLen)
		for _, bs := range matched {
			result.Or(bs)
		}

		return result, nil
	default:
		return nil, InvalidOperationError{SortedIndexName, op.Op}
	}
}

const FullScanName = "FullScan"

type ListFilterFn[OBJ any] interface {
	SetListFilterFn(func(predicat func(item *OBJ) bool) *BitSet32)
}

// FullScan, reads every item and execute the given predicate
// Is very slow, but don't need memory for saving the index-data.
// (is more an experiment)
type FullScan[OBJ any, V cmp.Ordered] struct {
	fieldGetFn FromField[OBJ, V]
	// will be injected from the List
	filter func(predicat func(item *OBJ) bool) *BitSet32
}

func NewFullScan[OBJ any, V cmp.Ordered](fieldGetFn FromField[OBJ, V]) Index[OBJ] {
	return &FullScan[OBJ, V]{fieldGetFn: fieldGetFn}
}

func (ft *FullScan[OBJ, V]) SetListFilterFn(filter func(predicat func(item *OBJ) bool) *BitSet32) {
	ft.filter = filter
}

func (ft *FullScan[OBJ, V]) Set(obj *OBJ, lidx uint32)   {}
func (ft *FullScan[OBJ, V]) UnSet(obj *OBJ, lidx uint32) {}
func (ft *FullScan[OBJ, V]) HasChanged(oldItem, newItem *OBJ) bool {
	// always false, because no update necessary
	return false
}

func (ft *FullScan[OBJ, V]) Match(op FilterOp, value any) (*BitSet32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	switch op.Op {
	case OpEq:
		return ft.filter(func(item *OBJ) bool {
			return ft.fieldGetFn(item) == v
		}), nil
	case OpLt:
		return ft.filter(func(item *OBJ) bool {
			return ft.fieldGetFn(item) < v
		}), nil
	case OpLe:
		return ft.filter(func(item *OBJ) bool {
			return ft.fieldGetFn(item) <= v
		}), nil
	case OpGt:
		return ft.filter(func(item *OBJ) bool {
			return ft.fieldGetFn(item) > v
		}), nil
	case OpGe:
		return ft.filter(func(item *OBJ) bool {
			return ft.fieldGetFn(item) >= v
		}), nil
	default:
		return nil, InvalidOperationError{FullScanName, op.Op}
	}
}

func (ft *FullScan[OBJ, V]) MatchMany(op FilterOp, values ...any) (*BitSet32, error) {

	switch op.Op {
	case OpBetween:
		if len(values) != 2 {
			return nil, InvalidArgsLenError{defined: "2", got: len(values)}
		}

		min, err := ValueFromAny[V](values[0])
		if err != nil {
			return nil, InvalidValueTypeError[V]{values[0]}
		}
		max, err := ValueFromAny[V](values[1])
		if err != nil {
			return nil, InvalidValueTypeError[V]{values[1]}
		}

		return ft.filter(func(item *OBJ) bool {
			v := ft.fieldGetFn(item)
			return v >= min && v <= max
		}), nil
	case OpIn:
		if len(values) == 0 {
			return NewEmptyBitSet[uint32](), nil
		}

		// check the values type corresponds to the field value
		for _, v := range values {
			if _, err := ValueFromAny[V](v); err != nil {
				return nil, err
			}
		}

		return ft.filter(func(item *OBJ) bool {
			fieldValue := ft.fieldGetFn(item)

			for _, v := range values {
				argValue, _ := ValueFromAny[V](v)
				if argValue == fieldValue {
					return true
				}
			}

			return false
		}), nil
	default:
		return nil, InvalidOperationError{FullScanName, op.Op}
	}
}
