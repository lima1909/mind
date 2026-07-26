package mind

import (
	"cmp"
	"iter"
	"sync"

	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

// IndexMap maps a given field name to an Index
type IndexMap[OBJ any] struct {
	index  map[string]Index[OBJ]
	allIDs *lidx.RawIDs32
}

func NewIndexMap[OBJ any](allIDs *lidx.RawIDs32) IndexMap[OBJ] {
	return IndexMap[OBJ]{
		index:  make(map[string]Index[OBJ]),
		allIDs: allIDs,
	}
}

// FilterByName finds the Filter by a given field-name
func (i IndexMap[OBJ]) FilterByName(fieldName string) (query.Filter, error) {
	if idx, found := i.index[fieldName]; found {
		return idx, nil
	}

	return nil, InvalidNameError{fieldName}
}

// insert to all known indexes synchron the new value (including ID-index)
func (i IndexMap[OBJ]) insert(obj *OBJ, idx int) {
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

// update update all known indexes synchron the new value (including ID-index)
func (i IndexMap[OBJ]) update(oldObj, newObj *OBJ, idx int) {
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

// delete remove all known indexes synchron the new value (including ID-index)
func (i IndexMap[OBJ]) delete(obj *OBJ, idx int) {
	uidx := uint32(idx)

	i.allIDs.UnSet(uidx)

	for _, fieldIndex := range i.index {
		fieldIndex.UnSet(obj, uidx)
	}
}

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
	query.Filter
}

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

const MapIndexName = "MapIndex"

// MapIndex is a mapping of any value to the Index in the List.
// This index only supported Queries with the Equal Ralation!
type MapIndex[OBJ any, V comparable, H ValueHandler[OBJ, V]] struct {
	data         map[V]*lidx.RawIDs32
	valueHandler H
}

func NewMapIndex[OBJ any, V comparable](fieldGetFn FromField[OBJ, V]) Index[OBJ] {
	return &MapIndex[OBJ, V, SingleValueHandler[OBJ, V]]{
		data:         make(map[V]*lidx.RawIDs32),
		valueHandler: SingleValueHandler[OBJ, V]{fieldGetFn},
	}
}

func NewMapIndexSlice[OBJ any, V comparable](fieldGetFn FromFieldSlice[OBJ, V]) Index[OBJ] {
	return &MapIndex[OBJ, V, MultiValueHandler[OBJ, V]]{
		data:         make(map[V]*lidx.RawIDs32),
		valueHandler: MultiValueHandler[OBJ, V]{fieldGetFn},
	}
}

func (mi *MapIndex[OBJ, V, H]) Set(obj *OBJ, idx uint32) {
	mi.valueHandler.Handle(obj, func(value V) {
		ids, found := mi.data[value]
		if !found {
			ids = lidx.NewRawIDs[uint32]()
			mi.data[value] = ids
		}

		ids.Set(idx)
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
		mi.data[val] = lidx.NewRawIDsFrom(ids...)
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

func (mi *MapIndex[OBJ, V, H]) Equal(value any) (*lidx.RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	ids, found := mi.data[v]
	if !found {
		return lidx.NewRawIDs[uint32](), nil
	}

	return ids, nil
}

func (mi *MapIndex[OBJ, V, H]) Match(_ *lidx.RawIDs32, op query.FilterOp, _ any) (*lidx.RawIDs32, bool, error) {
	return nil, false, InvalidOperationError{MapIndexName, op.Op}
}

// MatchMany is not supported by MapIndex, so that always returns an error
func (mi *MapIndex[OBJ, V, H]) MatchMany(op query.FilterOp, values ...any) (*lidx.RawIDs32, bool, error) {
	switch op.Op {
	case query.OpIn:
		// fast path for 0 or 1 values
		switch len(values) {
		case 0:
			return lidx.NewRawIDs[uint32](), true, nil
		case 1:
			key, err := ValueFromAny[V](values[0])
			if err != nil {
				return nil, false, err
			}
			if rid, found := mi.data[key]; found {
				// not copied
				return rid, false, nil
			}
			return lidx.NewRawIDs[uint32](), true, nil
		}

		matched := make([]*lidx.RawIDs32, 0, len(values))
		var maxLen int

		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, false, err
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
			return lidx.NewRawIDs[uint32](), true, nil
		case 1:
			// not copied
			return matched[0], false, nil
		}

		result := lidx.NewRawIDsWithCapacity[uint32](maxLen)
		for _, bs := range matched {
			result.Or(bs)
		}

		return result, true, nil
	default:
		return nil, false, InvalidOperationError{MapIndexName, op.Op}
	}
}

const SortedIndexName = "SortedIndex"

// SortedIndex is well suited for Queries with: Range, Min, Max, Greater and Less
type SortedIndex[OBJ any, V cmp.Ordered, H ValueHandler[OBJ, V]] struct {
	skipList     SkipList[V, *lidx.RawIDs32]
	valueHandler H
}

func NewSortedIndex[OBJ any, V cmp.Ordered](fieldGetFn FromField[OBJ, V]) Index[OBJ] {
	return &SortedIndex[OBJ, V, SingleValueHandler[OBJ, V]]{
		skipList:     NewSkipList[V, *lidx.RawIDs32](),
		valueHandler: SingleValueHandler[OBJ, V]{fieldGetFn},
	}
}

func NewSortedIndexSlice[OBJ any, V cmp.Ordered](fieldGetFn FromFieldSlice[OBJ, V]) Index[OBJ] {
	return &SortedIndex[OBJ, V, MultiValueHandler[OBJ, V]]{
		skipList:     NewSkipList[V, *lidx.RawIDs32](),
		valueHandler: MultiValueHandler[OBJ, V]{fieldGetFn},
	}
}

func (si *SortedIndex[OBJ, V, H]) Set(obj *OBJ, idx uint32) {
	si.valueHandler.Handle(obj, func(value V) {
		ids, found := si.skipList.Get(value)
		if !found {
			ids = lidx.NewRawIDs[uint32]()
			si.skipList.Put(value, ids)
		}

		ids.Set(idx)
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
		si.skipList.Put(val, lidx.NewRawIDsFrom(ids...))
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

func (si *SortedIndex[OBJ, V, H]) Equal(value any) (*lidx.RawIDs32, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	ids, found := si.skipList.Get(v)
	if !found {
		return lidx.NewRawIDs[uint32](), nil
	}

	return ids, nil
}

func (si *SortedIndex[OBJ, V, H]) Match(allIDs *lidx.RawIDs32, op query.FilterOp, value any) (*lidx.RawIDs32, bool, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, false, InvalidValueTypeError[V]{value}
	}

	var toMerge []*lidx.RawIDs32
	var visitor func(V, *lidx.RawIDs32) bool
	abortedForInversion := false

	if si.valueHandler.CanInvert() {
		// The Invertible Visitor (Counts and aborts)
		count := 0
		halfwayMark := si.skipList.Len() / 2

		visitor = func(_ V, bs *lidx.RawIDs32) bool {
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
		visitor = func(_ V, bs *lidx.RawIDs32) bool {
			toMerge = append(toMerge, bs)
			return true // Always keep going
		}
	}

	var invOp query.FilterOp
	switch op.Op {
	case query.OpLt:
		invOp = query.FilterOp{Op: query.OpGe}
		si.skipList.Less(v, visitor)
	case query.OpLe:
		invOp = query.FilterOp{Op: query.OpGt}
		si.skipList.LessEqual(v, visitor)
	case query.OpGt:
		invOp = query.FilterOp{Op: query.OpLe}
		si.skipList.Greater(v, visitor)
	case query.OpGe:
		invOp = query.FilterOp{Op: query.OpLt}
		si.skipList.GreaterEqual(v, visitor)
	default:
		return nil, false, InvalidOperationError{SortedIndexName, op.Op}
	}

	// query inversion optimization
	if abortedForInversion {
		inverseResult, _, err := si.Match(allIDs, invOp, value)
		if err != nil {
			return nil, false, err
		}

		// finalResult = allIDs - inverseResult
		finalResult := allIDs.Copy()
		finalResult.AndNot(inverseResult)
		return finalResult, true, nil
	}

	result := lidx.NewRawIDs[uint32]()
	for _, ids := range toMerge {
		result.Or(ids)
	}
	return result, true, nil
}

func (si *SortedIndex[OBJ, V, H]) MatchMany(op query.FilterOp, values ...any) (*lidx.RawIDs32, bool, error) {
	switch op.Op {
	case query.OpBetween:
		if len(values) != 2 {
			return nil, false, InvalidArgsLenError{Defined: "2", Got: len(values)}
		}

		min, err := ValueFromAny[V](values[0])
		if err != nil {
			return nil, false, InvalidValueTypeError[V]{values[0]}
		}
		max, err := ValueFromAny[V](values[1])
		if err != nil {
			return nil, false, InvalidValueTypeError[V]{values[1]}
		}

		result := lidx.NewRawIDs[uint32]()
		si.skipList.Range(min, max, func(_ V, bs *lidx.RawIDs32) bool {
			result.Or(bs)
			return true
		})
		return result, true, nil
	case query.OpIn:
		// fast path for 0 or 1 values
		switch len(values) {
		case 0:
			return lidx.NewRawIDs[uint32](), true, nil
		case 1:
			key, err := ValueFromAny[V](values[0])
			if err != nil {
				return nil, false, err
			}
			if rid, found := si.skipList.Get(key); found {
				// not copied
				return rid, false, nil
			}
			return lidx.NewRawIDs[uint32](), true, nil
		}

		matched := make([]*lidx.RawIDs32, 0, len(values))
		var maxLen int

		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, false, err
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
			return lidx.NewRawIDs[uint32](), true, nil
		case 1:
			// not copied
			return matched[0], false, nil
		}

		result := lidx.NewRawIDsWithCapacity[uint32](maxLen)
		for _, bs := range matched {
			result.Or(bs)
		}

		return result, true, nil
	default:
		return nil, false, InvalidOperationError{SortedIndexName, op.Op}
	}
}

// StringIndex only supported if the value is the string
type StringIndex[OBJ any, IDX Index[OBJ]] struct {
	fieldGetFn     FromField[OBJ, string]
	compositeIndex CompositeIndex[OBJ, IDX]
}

func NewStringIndex[OBJ any](fieldGetFn FromField[OBJ, string]) *StringIndex[OBJ, *MapIndex[OBJ, string, SingleValueHandler[OBJ, string]]] {
	return &StringIndex[OBJ, *MapIndex[OBJ, string, SingleValueHandler[OBJ, string]]]{
		fieldGetFn: fieldGetFn,
		compositeIndex: CompositeIndex[OBJ, *MapIndex[OBJ, string, SingleValueHandler[OBJ, string]]]{
			mainIndex: NewMapIndex(fieldGetFn).(*MapIndex[OBJ, string, SingleValueHandler[OBJ, string]]),
			routes:    make(map[query.FilterOp]Index[OBJ], 0),
		},
	}
}

func NewStringSortedIndex[OBJ any](fieldGetFn FromField[OBJ, string]) *StringIndex[OBJ, *SortedIndex[OBJ, string, SingleValueHandler[OBJ, string]]] {
	return &StringIndex[OBJ, *SortedIndex[OBJ, string, SingleValueHandler[OBJ, string]]]{
		fieldGetFn: fieldGetFn,
		compositeIndex: CompositeIndex[OBJ, *SortedIndex[OBJ, string, SingleValueHandler[OBJ, string]]]{
			mainIndex: NewSortedIndex(fieldGetFn).(*SortedIndex[OBJ, string, SingleValueHandler[OBJ, string]]),
			routes:    make(map[query.FilterOp]Index[OBJ], 0),
		},
	}
}

func (si *StringIndex[OBJ, H]) Add(idx Index[OBJ], ops ...query.FilterOp) *StringIndex[OBJ, H] {
	si.compositeIndex.Add(idx, ops...)
	return si
}

func (si *StringIndex[OBJ, H]) AddPhoneticIndex() *StringIndex[OBJ, H] {
	return si.Add(NewPhoneticIndex(si.fieldGetFn), query.FOpSounds)
}

func (si *StringIndex[OBJ, H]) AddFuzzyIndex() *StringIndex[OBJ, H] {
	return si.Add(NewFuzzyIndex(si.fieldGetFn), query.FOpFuzzy)
}

func (si *StringIndex[OBJ, H]) AddTrigramIndex() *StringIndex[OBJ, H] {
	return si.Add(NewTrigramIndex(si.fieldGetFn), query.FOpLike)
}

func (si *StringIndex[OBJ, H]) Set(obj *OBJ, lidx uint32)         { si.compositeIndex.Set(obj, lidx) }
func (si *StringIndex[OBJ, H]) BulkSet(objs iter.Seq2[int, *OBJ]) { si.compositeIndex.BulkSet(objs) }
func (si *StringIndex[OBJ, H]) UnSet(obj *OBJ, lidx uint32)       { si.compositeIndex.UnSet(obj, lidx) }
func (si *StringIndex[OBJ, H]) HasChanged(oldItem, newItem *OBJ) bool {
	return si.compositeIndex.HasChanged(oldItem, newItem)
}
func (si *StringIndex[OBJ, H]) Equal(value any) (*lidx.RawIDs32, error) {
	return si.compositeIndex.Equal(value)
}
func (si *StringIndex[OBJ, H]) Match(allIDs *lidx.RawIDs32, op query.FilterOp, value any) (*lidx.RawIDs32, bool, error) {
	return si.compositeIndex.Match(allIDs, op, value)
}

func (si *StringIndex[OBJ, H]) MatchMany(op query.FilterOp, values ...any) (*lidx.RawIDs32, bool, error) {
	return si.compositeIndex.MatchMany(op, values...)
}

// CompositeIndex fans out all mutations to every registered component index
// and routes each query operator to the single component that was registered for it.
type CompositeIndex[OBJ any, IDX Index[OBJ]] struct {
	mainIndex IDX
	routes    map[query.FilterOp]Index[OBJ]
}

func NewCompositeIndex[OBJ any, IDX Index[OBJ]](mainIndex IDX) *CompositeIndex[OBJ, IDX] {
	return &CompositeIndex[OBJ, IDX]{
		mainIndex: mainIndex,
		routes:    make(map[query.FilterOp]Index[OBJ], 0),
	}
}

func NewMapCompositeIndex[OBJ any, V comparable](fieldGetFn FromField[OBJ, V]) *CompositeIndex[OBJ, *MapIndex[OBJ, V, SingleValueHandler[OBJ, V]]] {
	return &CompositeIndex[OBJ, *MapIndex[OBJ, V, SingleValueHandler[OBJ, V]]]{
		mainIndex: NewMapIndex(fieldGetFn).(*MapIndex[OBJ, V, SingleValueHandler[OBJ, V]]),
		routes:    make(map[query.FilterOp]Index[OBJ], 0),
	}
}

func (ci *CompositeIndex[OBJ, IDX]) Add(idx Index[OBJ], ops ...query.FilterOp) *CompositeIndex[OBJ, IDX] {
	for _, op := range ops {
		ci.routes[op] = idx
	}
	return ci
}

func (ci *CompositeIndex[OBJ, IDX]) Set(obj *OBJ, lidx uint32) {
	ci.mainIndex.Set(obj, lidx)
	for _, idx := range ci.routes {
		idx.Set(obj, lidx)
	}
}

func (ci *CompositeIndex[OBJ, IDX]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	ci.mainIndex.BulkSet(objs)
	for _, idx := range ci.routes {
		idx.BulkSet(objs)
	}
}

func (ci *CompositeIndex[OBJ, IDX]) UnSet(obj *OBJ, lidx uint32) {
	ci.mainIndex.UnSet(obj, lidx)
	for _, idx := range ci.routes {
		idx.UnSet(obj, lidx)
	}
}

func (ci *CompositeIndex[OBJ, IDX]) HasChanged(oldItem, newItem *OBJ) bool {
	return ci.mainIndex.HasChanged(oldItem, newItem)
}

func (ci *CompositeIndex[OBJ, IDX]) Equal(value any) (*lidx.RawIDs32, error) {
	return ci.mainIndex.Equal(value)
}

func (ci *CompositeIndex[OBJ, IDX]) Match(allIDs *lidx.RawIDs32, op query.FilterOp, value any) (*lidx.RawIDs32, bool, error) {
	if idx, ok := ci.routes[op]; ok {
		return idx.Match(allIDs, op, value)
	}

	return ci.mainIndex.Match(allIDs, op, value)
}

func (ci *CompositeIndex[OBJ, IDX]) MatchMany(op query.FilterOp, values ...any) (*lidx.RawIDs32, bool, error) {

	if idx, ok := ci.routes[op]; ok {
		return idx.MatchMany(op, values...)
	}

	return ci.mainIndex.MatchMany(op, values...)
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

func (p *ParserExt[OBJ]) Equal(value any) (*lidx.RawIDs32, error) {
	if s, ok := value.(string); ok {
		return p.inner.Equal(p.parse(s))
	}

	return nil, InvalidValueTypeError[string]{value}
}

func (p *ParserExt[OBJ]) Match(allIDs *lidx.RawIDs32, op query.FilterOp, value any) (*lidx.RawIDs32, bool, error) {
	if s, ok := value.(string); ok {
		return p.inner.Match(allIDs, op, p.parse(s))
	}

	return nil, false, InvalidValueTypeError[string]{value}
}

func (p *ParserExt[OBJ]) MatchMany(op query.FilterOp, values ...any) (*lidx.RawIDs32, bool, error) {
	pvalues := make([]any, len(values))
	for i, v := range values {
		s, ok := v.(string)
		if !ok {
			return nil, false, InvalidValueTypeError[string]{v}
		}
		pvalues[i] = p.parse(s)

	}
	return p.inner.MatchMany(op, pvalues...)
}
