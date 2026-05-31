package mind

import (
	"iter"
)

const FenwickIndexName = "FenwickIndex"

type Int interface {
	~int8 | ~int16 | ~int32 | ~int |
		~uint8 | ~uint16 | ~uint32 | ~uint
}

// FenwickIndex (Binary Indexed Tree) handles bounded domains that include negative numbers.
// The index provides O(log M) writes and O(log M) range reads.
// It is well suited for range queries (Lt, Le, Gt, Ge, Between) over an integer field.
type FenwickIndex[OBJ any, V Int] struct {
	// tree is a store as a linear arrays
	// calculates the next Binary Tree Index: idx += idx & -idx
	tree    []*RawIDs32
	minVal  int
	maxVal  int
	handler SingleValueHandler[OBJ, V]
}

// NewFenwickIndex initializes the index structure.
// Pre-allocating the structure ensures the hot-path (Set/UnSet) remains 100% allocation-free.
// min value is 0, means only positive values
func NewFenwickIndex[OBJ any, V Int](fieldGetFn FromField[OBJ, V], maxVal int) Index[OBJ] {
	return NewFenwickIndexWithMinValue(fieldGetFn, 0, maxVal)
}

// NewFenwickIndexWithMinValue initializes the index.
// Example: Domain [-50, 150] yields an internal domain size of 200.
func NewFenwickIndexWithMinValue[OBJ any, V Int](fieldGetFn FromField[OBJ, V], minVal, maxVal int) Index[OBJ] {
	if minVal > maxVal {
		maxVal, minVal = minVal, maxVal
	}

	// calculate the mathematical distance between min and max
	domainSize := maxVal - minVal

	// add 2: one for the 0-inclusive boundary, one for the 1-based indexing shift
	size := domainSize + 2
	tree := make([]*RawIDs32, size)
	for i := range tree {
		tree[i] = NewRawIDs[uint32]()
	}

	return &FenwickIndex[OBJ, V]{
		tree:    tree,
		minVal:  minVal,
		maxVal:  maxVal,
		handler: SingleValueHandler[OBJ, V]{fieldGetFn},
	}
}

func (fx *FenwickIndex[OBJ, V]) Set(obj *OBJ, lidx uint32) {
	fx.handler.Handle(obj, func(value V) {
		v := int(value)
		if v < fx.minVal || v > fx.maxVal {
			return
		}

		// Shift value into positive space, then shift by +1 for 1-based indexing
		idx := fx.shift(v) + 1

		for idx < len(fx.tree) {
			fx.tree[idx].Set(lidx)
			idx += idx & -idx
		}
	})
}

func (fx *FenwickIndex[OBJ, V]) UnSet(obj *OBJ, lidx uint32) {
	fx.handler.Handle(obj, func(value V) {
		v := int(value)
		if v < fx.minVal || v > fx.maxVal {
			return
		}

		idx := fx.shift(v) + 1

		for idx < len(fx.tree) {
			fx.tree[idx].UnSet(lidx)
			idx += idx & -idx
		}
	})
}

func (fx *FenwickIndex[OBJ, V]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	for i, obj := range objs {
		fx.Set(obj, uint32(i))
	}
}

func (fx *FenwickIndex[OBJ, V]) HasChanged(oldItem, newItem *OBJ) bool {
	return fx.handler.HasChanged(oldItem, newItem)
}

// prefixUnion expects an ALREADY SHIFTED 0-based value.
func (fx *FenwickIndex[OBJ, V]) prefixUnion(shiftedV int) *RawIDs32 {
	idx := shiftedV + 1

	if idx >= len(fx.tree) {
		idx = len(fx.tree) - 1
	}

	result := NewRawIDs[uint32]()
	for idx > 0 {
		result.Or(fx.tree[idx])
		idx -= idx & -idx
	}

	return result
}

func (fx *FenwickIndex[OBJ, V]) Equal(value any) (*RawIDs32, error) {
	return nil, InvalidOperationError{FenwickIndexName, OpEq}
}

func (fx *FenwickIndex[OBJ, V]) Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, bool, error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, false, InvalidValueTypeError[V]{value}
	}
	iv := int(v)

	switch op.Op {
	case OpLe:
		if iv < fx.minVal {
			return NewRawIDs[uint32](), true, nil
		}
		if iv > fx.maxVal {
			return allIDs, false, nil
		}
		return fx.prefixUnion(fx.shift(iv)), true, nil

	case OpLt:
		if iv <= fx.minVal {
			return NewRawIDs[uint32](), true, nil
		}
		if iv > fx.maxVal {
			return allIDs, false, nil
		}
		return fx.prefixUnion(fx.shift(iv) - 1), true, nil

	case OpGt:
		if iv >= fx.maxVal {
			return NewRawIDs[uint32](), true, nil
		}
		if iv < fx.minVal {
			return allIDs, false, nil
		}
		result := allIDs.Copy()
		result.AndNot(fx.prefixUnion(fx.shift(iv)))
		return result, true, nil

	case OpGe:
		if iv > fx.maxVal {
			return NewRawIDs[uint32](), true, nil
		}
		if iv <= fx.minVal {
			return allIDs, false, nil
		}
		result := allIDs.Copy()
		result.AndNot(fx.prefixUnion(fx.shift(iv) - 1))
		return result, true, nil

	default:
		return nil, false, InvalidOperationError{FenwickIndexName, op.Op}
	}
}

func (fx *FenwickIndex[OBJ, V]) MatchMany(op FilterOp, values ...any) (*RawIDs32, bool, error) {
	switch op.Op {
	case OpBetween:
		if len(values) != 2 {
			return nil, false, InvalidArgsLenError{Defined: "2", Got: len(values)}
		}
		minV, err := ValueFromAny[V](values[0])
		if err != nil {
			return nil, false, InvalidValueTypeError[V]{values[0]}
		}
		maxV, err := ValueFromAny[V](values[1])
		if err != nil {
			return nil, false, InvalidValueTypeError[V]{values[1]}
		}

		if maxV < minV {
			return NewRawIDs[uint32](), true, nil
		}

		imin := int(minV)
		imax := int(maxV)

		if imin > fx.maxVal || imax < fx.minVal {
			return NewRawIDs[uint32](), true, nil
		}
		if imin < fx.minVal {
			imin = fx.minVal
		}
		if imax > fx.maxVal {
			imax = fx.maxVal
		}

		shiftedMax := fx.shift(imax)
		shiftedMin := fx.shift(imin)

		result := fx.prefixUnion(shiftedMax)
		if shiftedMin > 0 {
			result.AndNot(fx.prefixUnion(shiftedMin - 1))
			return result, true, nil
		}
		return result, true, nil

	default:
		return nil, false, InvalidOperationError{FenwickIndexName, op.Op}
	}
}

// shift converts the actual value to the 0-based internal value.
// E.g., for domain [-50, 150]: -50 becomes 0, 150 becomes 200.
func (fx *FenwickIndex[OBJ, V]) shift(v int) int { return v + -fx.minVal }
