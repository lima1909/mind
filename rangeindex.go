package mind

import (
	"iter"
)

const RangeIndexName = "RangeIndex"

// RangeIndex (specifically, an Equality-Encoded Array Index) is a direct-mapping data structure.
// It uses a flat array where every slot corresponds to exactly one specific value.
//
// Ideal Use Cases:
// - Status Codes (e.g., 0 = Pending, 1 = Active, 2 = Banned)
// - Categories / Enums (e.g., Department ID between 1 and 50)
// - Boolean flags (0 or 1)
// - Any small, discrete domain where values change often.
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

func (ri *RangeIndex[OBJ, H]) Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, bool, error) {
	v, err := ValueFromAny[uint8](value)
	if err != nil {
		return nil, false, InvalidValueTypeError[uint8]{value}
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
		return nil, false, InvalidOperationError{RangeIndexName, op.Op}
	}

	if end > ri.max {
		end = ri.max
	}
	if start >= end {
		return NewRawIDs[uint32](), true, nil
	}

	// Query Inversion Optimization
	// If the range we are scanning is more than half of our total active data range,
	// it's cheaper to get the inverse and subtract it from allIDs.
	if ri.valueHandler.CanInvert() && end-start > (ri.max/2) {
		// calculate the IDs we DON'T want
		inverseResult, _, err := ri.Match(allIDs, invOp, value)
		if err != nil {
			return nil, false, err
		}

		// result = allIDs - inverseResult
		finalResult := allIDs.Copy()
		finalResult.AndNot(inverseResult)
		return finalResult, true, nil
	}

	result := NewRawIDs[uint32]()
	for i := start; i < end; i++ {
		data := ri.data[i]
		if data != nil && !data.IsEmpty() {
			result.Or(data)
		}
	}

	return result, true, nil
}

func (ri *RangeIndex[OBJ, H]) MatchMany(op FilterOp, values ...any) (*RawIDs32, bool, error) {
	switch op.Op {
	case OpBetween:
		if len(values) != 2 {
			return nil, false, InvalidArgsLenError{Defined: "2", Got: len(values)}
		}

		minVal, err := ValueFromAny[uint8](values[0])
		if err != nil {
			return nil, false, InvalidValueTypeError[uint8]{values[0]}
		}
		maxVal, err := ValueFromAny[uint8](values[1])
		if err != nil {
			return nil, false, InvalidValueTypeError[uint8]{values[1]}
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
		return result, true, nil
	case OpIn:
		result := NewRawIDs[uint32]()
		for _, v := range values {
			i, err := ValueFromAny[uint8](v)
			if err != nil {
				return nil, false, err
			}

			valInt := int(i)
			if valInt < ri.max && ri.data[valInt] != nil && !ri.data[valInt].IsEmpty() {
				result.Or(ri.data[i])
			}
		}
		return result, true, nil

	default:
		return nil, false, InvalidOperationError{RangeIndexName, op.Op}
	}
}

const RangeEncodedIndexName = "RangeEncodedIndex"

// RangeEncodedIndex is a specialized index designed to execute range queries
// like: <, <=, >, >=, and Between, at absolute maximum speed.
// Unlike a traditional index that maps a record to its exact value, a range-encoded index stores data cumulatively.
// Every slot in the index answers a "less than or equal to" question.
//
// Ideal Use Cases:
// This index is a specialized weapon for read-heavy analytics where data is infrequently updated, and the fields have small, fixed numeric boundaries.
// - Percentages (0 to 100)
// - Age fields (0 to 120)
// - Star Ratings (1 to 5)
// - Days of the week or year (1 to 7, or 1 to 365)
//
// Disadvantages:
// - High Write Amplification: Inserting or updating a record is expensive.
// - It is completely unusable for unbounded or massive fields (like unique timestamps, UUIDs, or floating-point prices)
// - used a lot of memory
type RangeEncodedIndex[OBJ any, H ValueHandler[OBJ, uint8]] struct {
	prefixTree []*RawIDs32
	// the max length of the prefixTree
	// max can be: 256 if the data is full from 0-255
	max     int
	handler H
}

func NewRangeEncodedIndex[OBJ any](fieldGetFn FromField[OBJ, uint8], max uint8) Index[OBJ] {
	// Array size must be 256 to cover indices 0-255
	slices := make([]*RawIDs32, int(max)+1)
	for i := range slices {
		slices[i] = NewRawIDs[uint32]()
	}
	return &RangeEncodedIndex[OBJ, SingleValueHandler[OBJ, uint8]]{
		prefixTree: slices,
		max:        int(max),
		handler:    SingleValueHandler[OBJ, uint8]{fieldGetFn},
	}
}

func (ri *RangeEncodedIndex[OBJ, H]) Set(obj *OBJ, lidx uint32) {
	ri.handler.Handle(obj, func(value uint8) {
		for i := int(value); i <= ri.max; i++ {
			ri.prefixTree[i].Set(lidx)
		}
	})
}

func (ri *RangeEncodedIndex[OBJ, H]) UnSet(obj *OBJ, lidx uint32) {
	ri.handler.Handle(obj, func(value uint8) {
		for i := int(value); i <= ri.max; i++ {
			ri.prefixTree[i].UnSet(lidx)
		}
	})
}

func (ri *RangeEncodedIndex[OBJ, H]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	for i, obj := range objs {
		ri.Set(obj, uint32(i))
	}
}
func (ri *RangeEncodedIndex[OBJ, H]) HasChanged(oldItem, newItem *OBJ) bool {
	return ri.handler.HasChanged(oldItem, newItem)
}
func (ri *RangeEncodedIndex[OBJ, H]) Equal(value any) (*RawIDs32, error) {
	return nil, InvalidOperationError{RangeEncodedIndexName, OpEq}
}

func (ri *RangeEncodedIndex[OBJ, H]) Match(allIDs *RawIDs32, op FilterOp, value any) (*RawIDs32, bool, error) {
	v, err := ValueFromAny[uint8](value)
	if err != nil {
		return nil, false, InvalidValueTypeError[uint8]{value}
	}
	iv := int(v)

	switch op.Op {
	case OpLt:
		target := iv - 1
		if target < 0 {
			return NewRawIDs[uint32](), true, nil
		}
		if target > ri.max {
			target = ri.max
		}
		return ri.prefixTree[target], false, nil

	case OpLe:
		if iv < 0 {
			return NewRawIDs[uint32](), true, nil
		}
		if iv > ri.max {
			iv = ri.max
		}
		return ri.prefixTree[iv], false, nil

	case OpGt:
		if iv > ri.max-1 {
			return NewRawIDs[uint32](), true, nil
		}
		if iv < 0 {
			return allIDs, false, nil
		}
		result := allIDs.Copy()
		result.AndNot(ri.prefixTree[iv])
		return result, true, nil

	case OpGe:
		if iv > ri.max {
			return NewRawIDs[uint32](), true, nil
		}
		if iv <= 0 {
			return allIDs, false, nil
		}
		result := allIDs.Copy()
		result.AndNot(ri.prefixTree[iv-1]) // Exactly 1 MERGE
		return result, true, nil

	default:
		return nil, false, InvalidOperationError{RangeEncodedIndexName, op.Op}
	}
}

func (ri *RangeEncodedIndex[OBJ, H]) MatchMany(op FilterOp, values ...any) (*RawIDs32, bool, error) {
	switch op.Op {
	case OpBetween:
		if len(values) != 2 {
			return nil, false, InvalidArgsLenError{Defined: "2", Got: len(values)}
		}
		minVal, err := ValueFromAny[uint8](values[0])
		if err != nil {
			return nil, false, InvalidValueTypeError[uint8]{values[0]}
		}
		maxVal, err := ValueFromAny[uint8](values[1])
		if err != nil {
			return nil, false, InvalidValueTypeError[uint8]{values[1]}
		}

		if maxVal < minVal {
			return NewRawIDs[uint32](), true, nil
		}

		imax := min(int(maxVal), ri.max)
		imin := max(int(minVal), 0)
		if imin > 0 {
			result := ri.prefixTree[imax].Copy()
			result.AndNot(ri.prefixTree[imin-1])
			return result, true, nil
		}
		return ri.prefixTree[imax], false, nil

	default:
		return nil, false, InvalidOperationError{RangeEncodedIndexName, op.Op}
	}
}
