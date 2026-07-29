package index

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

func NewSingleValueHandler[OBJ any, V comparable](fieldGetFn FromField[OBJ, V]) SingleValueHandler[OBJ, V] {
	return SingleValueHandler[OBJ, V]{fieldGetFn: fieldGetFn}
}

func (h SingleValueHandler[OBJ, V]) Handle(obj *OBJ, exec func(V)) { exec(h.fieldGetFn(obj)) }
func (h SingleValueHandler[OBJ, V]) HasChanged(oldItem, newItem *OBJ) bool {
	return h.fieldGetFn(oldItem) != h.fieldGetFn(newItem)
}
func (h SingleValueHandler[OBJ, V]) CanInvert() bool { return true }

type MultiValueHandler[OBJ any, V comparable] struct {
	fieldGetFn FromFieldSlice[OBJ, V]
}

func NewMultiValueHandler[OBJ any, V comparable](fieldGetFn FromFieldSlice[OBJ, V]) MultiValueHandler[OBJ, V] {
	return MultiValueHandler[OBJ, V]{fieldGetFn: fieldGetFn}
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
