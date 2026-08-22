package mind

// List is a fast in-memory store, which is extended by Indices for fast finding Items.
//
// WARNING: If T is a pointer type, modifying the items returned by Get() or Query()
// will corrupt the database indexes. Always use Update() to modify data.
type List[T any] struct {
	store[T]
}

// NewList create a new List
func NewList[T any]() *List[T] {
	return &List[T]{
		store: newStore[T](),
	}
}

// Get the Item of the given list-index, or the zero value and false,
func (l *List[T]) Get(lidx int) (T, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.list.Get(lidx)
}

// Update gives the Item of the given list-index to the update function, so the caller can
// update the fields he wants, and keeps all indexes in sync.
// Returns false, if the list-index is unknown or the Item was removed.
func (l *List[T]) Update(lidx int, update func(*T)) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.updateAt(lidx, update) == nil
}

// Remove the Item of the given list-index.
// Returns false, if the list-index is unknown or the Item was already removed.
func (l *List[T]) Remove(lidx int) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.removeAt(lidx)
}
