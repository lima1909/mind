package mind

import (
	"cmp"
	"errors"
)

const IDIndexFieldName = "id"

// fieldIndexMap maps a given field name to an Index
type indexMap[OBJ any, ID comparable] struct {
	idIndex idIndex[OBJ, ID]
	index   map[string]Index[OBJ]
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

func (id *idAutoIncIndex[OBJ]) Equal(value any) (*RawIDs32, error) {
	i, ok := value.(uint64)
	if !ok {
		return nil, InvalidValueTypeError[uint32]{value}
	}

	idx, err := id.GetIndex(i)
	if err != nil {
		return nil, err
	}

	return NewRawIDsFrom(uint32(idx)), nil
}

func (id *idAutoIncIndex[OBJ]) Match(op FilterOp, value any) (*RawIDs32, error) {
	return nil, InvalidOperationError{IDAutoIncName, op.Op}
}

// MatchMany is not supported by idAutoIncIndex, so that always returns an error
func (id *idAutoIncIndex[OBJ]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
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

func (mi *idMapIndex[OBJ, ID]) Match(op FilterOp, value any) (*RawIDs32, error) {
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
	Set(*OBJ, uint32)
	UnSet(*OBJ, uint32)
	HasChanged(oldItem, newItem *OBJ) bool
	Filter
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
	FOpStartsWith = FilterOp{Name: "startswith"}
)

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

// Filter returns the RawIDs or an error by a given Relation and Value
type Filter interface {
	// Equal is seperated from Match
	// because the RawIDs result you can NOT mutable
	Equal(value any) (*RawIDs32, error)
	Match(op FilterOp, value any) (*RawIDs32, error)
	MatchMany(op FilterOp, values ...any) (*RawIDs32, error)
}

const MapIndexName = "MapIndex"

// MapIndex is a mapping of any value to the Index in the List.
// This index only supported Queries with the Equal Ralation!
type MapIndex[OBJ any, V comparable] struct {
	data       map[any]*RawIDs32
	fieldGetFn FromField[OBJ, V]
}

func NewMapIndex[OBJ any, V comparable](fromField FromField[OBJ, V]) Index[OBJ] {
	return &MapIndex[OBJ, V]{
		data:       make(map[any]*RawIDs32),
		fieldGetFn: fromField,
	}
}

func (mi *MapIndex[OBJ, V]) Set(obj *OBJ, lidx uint32) {
	value := mi.fieldGetFn(obj)
	bs, found := mi.data[value]
	if !found {
		bs = NewRawIDs[uint32]()
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

func (mi *MapIndex[OBJ, V]) Equal(value any) (*RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	bs, found := mi.data[v]
	if !found {
		return NewRawIDs[uint32](), nil
	}

	return bs, nil
}

func (mi *MapIndex[OBJ, V]) Match(op FilterOp, value any) (*RawIDs32, error) {
	return nil, InvalidOperationError{MapIndexName, op.Op}
}

// MatchMany is not supported by MapIndex, so that always returns an error
func (mi *MapIndex[OBJ, V]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
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
type SortedIndex[OBJ any, V cmp.Ordered] struct {
	skipList   SkipList[V, *RawIDs32]
	fieldGetFn FromField[OBJ, V]
}

func NewSortedIndex[OBJ any, V cmp.Ordered](fieldGetFn FromField[OBJ, V]) Index[OBJ] {
	return &SortedIndex[OBJ, V]{
		skipList:   NewSkipList[V, *RawIDs32](),
		fieldGetFn: fieldGetFn,
	}
}

func (si *SortedIndex[OBJ, V]) Set(obj *OBJ, lidx uint32) {
	value := si.fieldGetFn(obj)
	bs, found := si.skipList.Get(value)
	if !found {
		bs = NewRawIDs[uint32]()
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

func (si *SortedIndex[OBJ, V]) Equal(value any) (*RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	if bs, found := si.skipList.Get(v); found {
		return bs, nil
	}
	return NewRawIDs[uint32](), nil
}

func (si *SortedIndex[OBJ, V]) Match(op FilterOp, value any) (*RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	switch op.Op {
	case OpLt:
		result := NewRawIDs[uint32]()
		si.skipList.Less(v, func(_ V, bs *RawIDs32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpLe:
		result := NewRawIDs[uint32]()
		si.skipList.LessEqual(v, func(_ V, bs *RawIDs32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpGt:
		result := NewRawIDs[uint32]()
		si.skipList.Greater(v, func(v V, bs *RawIDs32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	case OpGe:
		result := NewRawIDs[uint32]()
		si.skipList.GreaterEqual(v, func(_ V, bs *RawIDs32) bool {
			result.Or(bs)
			return true
		})
		return result, nil
	default:
		if op.Name == FOpStartsWith.Name {
			if _, ok := value.(string); !ok {
				return nil, InvalidValueTypeError[string]{value}
			}

			result := NewRawIDs[uint32]()
			si.skipList.StringStartsWith(v, func(_ V, bs *RawIDs32) bool {
				result.Or(bs)
				return true
			})
			return result, nil
		}

		return nil, InvalidOperationError{SortedIndexName, op.Op}
	}
}

func (si *SortedIndex[OBJ, V]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
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

type RangeIndex[OBJ any] struct {
	data [256]*RawIDs32
	// the length of the data (the max value)
	// max can be: 256 if the data is full from 0-255
	max        int
	fieldGetFn FromField[OBJ, uint8]
}

func NewRangeIndex[OBJ any](fieldGetFn FromField[OBJ, uint8]) Index[OBJ] {
	return &RangeIndex[OBJ]{
		// Array size must be 256 to cover indices 0-255
		data:       [256]*RawIDs32{},
		fieldGetFn: fieldGetFn,
	}
}

func (ri *RangeIndex[OBJ]) Set(obj *OBJ, lidx uint32) {
	value := ri.fieldGetFn(obj)
	valInt := int(value)

	r := ri.data[valInt]
	if r == nil {
		r = NewRawIDs[uint32]()
		ri.data[valInt] = r
	}
	r.Set(lidx)

	// new max value, if value greater the old max value
	if ri.max < valInt+1 {
		ri.max = valInt + 1
	}
}

func (ri *RangeIndex[OBJ]) UnSet(obj *OBJ, lidx uint32) {
	value := ri.fieldGetFn(obj)
	valInt := int(value)

	r := ri.data[valInt]
	if r == nil {
		return
	}
	r.UnSet(lidx)

	if r.IsEmpty() {
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
}

func (ri *RangeIndex[OBJ]) HasChanged(oldItem, newItem *OBJ) bool {
	return ri.fieldGetFn(oldItem) != ri.fieldGetFn(newItem)
}

func (ri *RangeIndex[OBJ]) Equal(value any) (*RawIDs32, error) {
	v, err := ValueFromAny[uint8](value)
	if err != nil {
		return nil, InvalidValueTypeError[uint8]{value}
	}

	r := ri.data[v]
	if r == nil {
		return NewRawIDs[uint32](), nil
	}
	return r, nil
}

func (ri *RangeIndex[OBJ]) Match(op FilterOp, value any) (*RawIDs32, error) {
	v, err := ValueFromAny[uint8](value)
	if err != nil {
		return nil, InvalidValueTypeError[uint8]{value}
	}
	valInt := int(v)

	switch op.Op {
	case OpLt, OpLe, OpGt, OpGe:
		start, end := 0, ri.max
		if op.Op == OpLt {
			end = valInt
		}
		if op.Op == OpLe {
			end = valInt + 1
		}
		if op.Op == OpGt {
			start = valInt + 1
		}
		if op.Op == OpGe {
			start = valInt
		}

		// Bound checks
		if end > ri.max {
			end = ri.max
		}

		result := NewRawIDs[uint32]()
		for i := start; i < end; i++ {
			if ri.data[i] != nil && !ri.data[i].IsEmpty() {
				result.Or(ri.data[i])
			}
		}
		return result, nil

	default:
		return nil, InvalidOperationError{SortedIndexName, op.Op}
	}
}

func (ri *RangeIndex[OBJ]) MatchMany(op FilterOp, values ...any) (*RawIDs32, error) {
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
		return nil, InvalidOperationError{SortedIndexName, op.Op}
	}
}
