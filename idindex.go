package mind

import (
	"fmt"
	"iter"

	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

type idIndex[OBJ any, ID comparable] interface {
	Index[OBJ]
	GetIndex(ID) (uint32, bool)
	GetID(*OBJ) ID
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

func (mi *idMapIndex[OBJ, ID]) GetIndex(id ID) (uint32, bool) {
	lidx, found := mi.data[id]
	return lidx, found
}

func (mi *idMapIndex[OBJ, ID]) GetID(item *OBJ) ID { return mi.fieldGetFn(item) }

func (mi *idMapIndex[OBJ, ID]) Equal(value any) (*lidx.RawIDs32, error) {
	id, ok := value.(ID)
	if !ok {
		return nil, InvalidValueTypeError[ID]{value}
	}

	idx, found := mi.GetIndex(id)
	if !found {
		return nil, ValueNotFoundError{id}
	}

	return lidx.NewRawIDsFrom(uint32(idx)), nil
}

func (mi *idMapIndex[OBJ, ID]) Match(_ *lidx.RawIDs32, op query.FilterOp, _ any) (*lidx.RawIDs32, bool, error) {
	return nil, false, InvalidOperationError{IDMapIndexName, op.Op}
}

// MatchMany is not supported by idMapIndex, so that always returns an error
func (mi *idMapIndex[OBJ, ID]) MatchMany(op query.FilterOp, values ...any) (*lidx.RawIDs32, bool, error) {
	return nil, false, InvalidOperationError{IDMapIndexName, op.Op}
}

const IDSliceIndexName = "IDSliceIndex"

type SliceID interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type uintBucket struct {
	val      uint32
	occupied bool
}

type idSliceIndex[OBJ any, ID SliceID] struct {
	idToIdx    []uintBucket
	fieldGetFn FromField[OBJ, ID]
	offset     ID // The minimum allowed value (e.g., -10)
}

func NewIDSliceIndex[OBJ any, ID SliceID](fieldGetFn FromField[OBJ, ID]) *idSliceIndex[OBJ, ID] {
	return &idSliceIndex[OBJ, ID]{
		idToIdx:    make([]uintBucket, 0),
		fieldGetFn: fieldGetFn,
	}
}

//go:inline
func (si *idSliceIndex[OBJ, ID]) toIndex(id ID) int { return int(id) - int(si.offset) }

func (si *idSliceIndex[OBJ, ID]) Set(obj *OBJ, lidx uint32) {
	id := si.fieldGetFn(obj)
	iid := si.toIndex(id)

	if iid < 0 {
		panic(fmt.Sprintf("ID: %v is below the configured minimum offset of %v", id, si.offset))
	}

	if iid >= len(si.idToIdx) {
		if cap(si.idToIdx) <= iid {
			// Double capacity to avoid frequent allocations
			newIdToIdx := make([]uintBucket, id+1, (id+1)*2)
			copy(newIdToIdx, si.idToIdx)
			si.idToIdx = newIdToIdx
		} else {
			si.idToIdx = si.idToIdx[:id+1]
		}
	}
	si.idToIdx[id] = uintBucket{lidx, true}
}

// FIXED: Changed to pointer receiver so reallocations persist
func (si *idSliceIndex[OBJ, ID]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	for lidx, obj := range objs {
		si.Set(obj, uint32(lidx))
	}
}

func (si *idSliceIndex[OBJ, ID]) UnSet(obj *OBJ, lidx uint32) {
	id := si.fieldGetFn(obj)
	iid := si.toIndex(id)

	if iid < 0 || iid >= len(si.idToIdx) {
		return
	}

	if si.idToIdx[iid].occupied && si.idToIdx[iid].val == lidx {
		si.idToIdx[iid] = uintBucket{}
	}
}

func (si *idSliceIndex[OBJ, ID]) HasChanged(oldItem, newItem *OBJ) bool {
	return si.fieldGetFn(oldItem) != si.fieldGetFn(newItem)
}

func (si *idSliceIndex[OBJ, ID]) GetIndex(id ID) (uint32, bool) {
	iid := si.toIndex(id)

	if iid < 0 || iid >= len(si.idToIdx) {
		return 0, false
	}

	bucket := si.idToIdx[id]
	return bucket.val, bucket.occupied
}

func (si *idSliceIndex[OBJ, ID]) GetID(item *OBJ) ID { return si.fieldGetFn(item) }

func (si *idSliceIndex[OBJ, ID]) Equal(value any) (*lidx.RawIDs32, error) {
	id, ok := value.(ID)
	if !ok {
		return nil, InvalidValueTypeError[ID]{value}
	}

	idx, found := si.GetIndex(id)
	if !found {
		return nil, ValueNotFoundError{id}
	}

	return lidx.NewRawIDsFrom(uint32(idx)), nil
}

func (*idSliceIndex[OBJ, ID]) Match(_ *lidx.RawIDs32, op query.FilterOp, _ any) (*lidx.RawIDs32, bool, error) {
	return nil, false, InvalidOperationError{IDSliceIndexName, op.Op}
}

func (*idSliceIndex[OBJ, ID]) MatchMany(op query.FilterOp, values ...any) (*lidx.RawIDs32, bool, error) {
	return nil, false, InvalidOperationError{IDSliceIndexName, op.Op}
}
