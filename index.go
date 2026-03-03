package main

import (
	"cmp"
	"fmt"
	"reflect"
	"unsafe"
)

const IDIndexFieldName = "id"

// fieldIndexMap maps a given field name to an Index
type indexMap[OBJ any, ID comparable] struct {
	idIndex idIndex[OBJ, ID]
	index   map[string]Index32[OBJ]
	allIDs  *BitSet[uint32]
}

func newIndexMap[OBJ any, ID comparable](idIndex idIndex[OBJ, ID]) indexMap[OBJ, ID] {
	return indexMap[OBJ, ID]{
		idIndex: idIndex,
		index:   make(map[string]Index32[OBJ]),
		allIDs:  NewBitSet[uint32](),
	}
}

// FilterByName finds the Filter by a given field-name
func (i indexMap[OBJ, ID]) FilterByName(fieldName string) (Filter32, error) {
	if fieldName == IDIndexFieldName {
		if i.idIndex == nil {
			return nil, ErrNoIdIndexDefined{}
		}
		return i.idIndex, nil
	}

	if idx, found := i.index[fieldName]; found {
		return idx, nil
	}

	return nil, ErrInvalidIndexdName{fieldName}
}

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

func (i indexMap[OBJ, ID]) getIndexByID(id ID) (int, error) {
	if i.idIndex == nil {
		return 0, ErrNoIdIndexDefined{}
	}

	return i.idIndex.GetIndex(id)
}

func (i indexMap[OBJ, ID]) getIDByItem(item *OBJ) (ID, int, error) {
	if i.idIndex == nil {
		var id ID
		return id, 0, ErrNoIdIndexDefined{}
	}

	return i.idIndex.GetID(item)
}

type idIndex[OBJ any, ID comparable] interface {
	Set(*OBJ, int)
	UnSet(*OBJ, int)
	GetIndex(ID) (int, error)
	GetID(*OBJ) (ID, int, error)
	Filter32
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

	return 0, ErrValueNotFound{id}
}

func (mi *idMapIndex[OBJ, ID]) GetID(item *OBJ) (ID, int, error) {
	id := mi.fieldGetFn(item)
	if lidx, found := mi.data[id]; found {
		return id, lidx, nil
	}

	var null ID
	return null, 0, ErrValueNotFound{id}
}

func (mi *idMapIndex[OBJ, ID]) Match(op Op, value any) (*BitSet[uint32], error) {
	id, ok := value.(ID)
	if !ok {
		return nil, ErrInvalidIndexValue[ID]{value}
	}

	if op != OpEq {
		return nil, ErrInvalidOperation{IDMapIndexName, op}
	}

	idx, err := mi.GetIndex(id)
	if err != nil {
		return nil, err
	}

	return NewBitSetFrom(uint32(idx)), nil

}

// MatchMany is not supported by idMapIndex, so that always returns an error
func (mi *idMapIndex[OBJ, ID]) MatchMany(op Op, values ...any) (*BitSet[uint32], error) {
	return nil, ErrInvalidOperation{IDMapIndexName, op}
}

// ------------------------------------------
// here starts the Index with the Index impls
// ------------------------------------------

// Index32 the IndexList only supports uint32 List-Indices
type Index32[T any] = Index[T, uint32]

// Index is interface for handling the mapping of an Value: V to an List-Index: LI
// The Value V comes from a func(*OBJ) V
type Index[OBJ any, LI Value] interface {
	Set(*OBJ, LI)
	UnSet(*OBJ, LI)
	Filter[LI]
}

// Filter32 the IndexList only supports uint32 List-Indices
type Filter32 = Filter[uint32]

// Filter returns the BitSet or an error by a given Relation and Value
type Filter[LI Value] interface {
	Match(op Op, value any) (*BitSet[LI], error)
	MatchMany(op Op, values ...any) (*BitSet[LI], error)
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
type MapIndex[OBJ any, V any, LI Value] struct {
	data       map[any]*BitSet[LI]
	fieldGetFn FromField[OBJ, V]
}

func NewMapIndex[OBJ any, V any](fromField FromField[OBJ, V]) Index32[OBJ] {
	return &MapIndex[OBJ, V, uint32]{
		data:       make(map[any]*BitSet[uint32]),
		fieldGetFn: fromField,
	}
}

func (mi *MapIndex[OBJ, V, LI]) Set(obj *OBJ, lidx LI) {
	value := mi.fieldGetFn(obj)
	bs, found := mi.data[value]
	if !found {
		bs = NewBitSet[LI]()
	}
	bs.Set(lidx)
	mi.data[value] = bs
}

func (mi *MapIndex[OBJ, V, LI]) UnSet(obj *OBJ, lidx LI) {
	value := mi.fieldGetFn(obj)
	if bs, found := mi.data[value]; found {
		bs.UnSet(lidx)
		if bs.Count() == 0 {
			delete(mi.data, value)
		}
	}
}

func (mi *MapIndex[OBJ, V, LI]) Match(op Op, value any) (*BitSet[LI], error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, ErrInvalidIndexValue[V]{value}
	}

	if op != OpEq {
		return nil, ErrInvalidOperation{MapIndexName, op}
	}

	bs, found := mi.data[v]
	if !found {
		return NewBitSet[LI](), nil
	}

	return bs, nil
}

// MatchMany is not supported by MapIndex, so that always returns an error
func (mi *MapIndex[OBJ, V, LI]) MatchMany(op Op, values ...any) (*BitSet[LI], error) {
	return nil, ErrInvalidOperation{MapIndexName, op}
}

const SortedIndexName = "SortedIndex"

// SortedIndex is well suited for Queries with: Range, Min, Max, Greater and Less
type SortedIndex[OBJ any, V cmp.Ordered, LI Value] struct {
	skipList   SkipList[V, *BitSet[LI]]
	fieldGetFn FromField[OBJ, V]
}

func NewSortedIndex[OBJ any, V cmp.Ordered](fieldGetFn FromField[OBJ, V]) Index32[OBJ] {
	return &SortedIndex[OBJ, V, uint32]{
		skipList:   NewSkipList[V, *BitSet[uint32]](),
		fieldGetFn: fieldGetFn,
	}
}

func (si *SortedIndex[OBJ, V, LI]) Set(obj *OBJ, lidx LI) {
	value := si.fieldGetFn(obj)
	bs, found := si.skipList.Get(value)
	if !found {
		bs = NewBitSet[LI]()
	}
	bs.Set(lidx)
	si.skipList.Put(value, bs)
}

func (si *SortedIndex[OBJ, V, LI]) UnSet(obj *OBJ, lidx LI) {
	value := si.fieldGetFn(obj)
	if bs, found := si.skipList.Get(value); found {
		bs.UnSet(lidx)
		if bs.Count() == 0 {
			si.skipList.Delete(value)
		}
	}
}

func (si *SortedIndex[OBJ, V, LI]) Match(op Op, value any) (*BitSet[LI], error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, ErrInvalidIndexValue[V]{value}
	}

	switch op {
	case OpEq:
		if bs, found := si.skipList.Get(v); found {
			return bs, nil
		}
		return NewBitSet[LI](), nil
	case OpLt:
		result := NewBitSet[LI]()
		si.skipList.Less(v, func(v V, bs *BitSet[LI]) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpLe:
		result := NewBitSet[LI]()
		si.skipList.LessEqual(v, func(_ V, bs *BitSet[LI]) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpGt:
		result := NewBitSet[LI]()
		si.skipList.Greater(v, func(v V, bs *BitSet[LI]) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpGe:
		result := NewBitSet[LI]()
		si.skipList.GreaterEqual(v, func(_ V, bs *BitSet[LI]) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpStartsWith:
		if _, ok := value.(string); !ok {
			return nil, ErrInvalidIndexValue[string]{value}
		}

		result := NewBitSet[LI]()
		si.skipList.StringStartsWith(v, func(_ V, bs *BitSet[LI]) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	default:
		return nil, ErrInvalidOperation{SortedIndexName, op}
	}
}

func (si *SortedIndex[OBJ, V, LI]) MatchMany(op Op, values ...any) (*BitSet[LI], error) {
	switch op {
	case OpBetween:
		if len(values) != 2 {
			return nil, ErrInvalidArgsLen{defined: "2", got: len(values)}
		}

		min, err := ValueFromAny[V](values[0])
		if err != nil {
			return nil, ErrInvalidIndexValue[V]{values[0]}
		}
		max, err := ValueFromAny[V](values[1])
		if err != nil {
			return nil, ErrInvalidIndexValue[V]{values[1]}
		}

		result := NewBitSet[LI]()
		si.skipList.Range(min, max, func(_ V, bs *BitSet[LI]) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpIn:
		if len(values) == 0 {
			return NewBitSet[LI](), nil
		}

		var result *BitSet[LI]
		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, err
			}

			bs, found := si.skipList.Get(key)
			if found {
				if result == nil {
					result = bs.Copy()
				} else {
					result.Or(bs)
				}
			}
		}

		if result == nil {
			return NewBitSet[LI](), nil
		}
		// result := NewBitSet[LI]()
		// err := si.skipList.FindMaybeSortedKeys(func(_ V, bs *BitSet[LI]) bool {
		// 	result.Or(bs)
		// 	return true
		// }, values...)
		// if err != nil {
		// 	return nil, err
		// }

		return result, nil

	default:
		return nil, ErrInvalidOperation{SortedIndexName, op}
	}
}
