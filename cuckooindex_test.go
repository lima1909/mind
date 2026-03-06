package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCuckooIndexBase(t *testing.T) {
	ci := newCuckoo()
	assert.True(t, ci.Put(1, 1))
	assert.True(t, ci.Put(2, 2))

	//TODO: overwrite, is true correct
	assert.True(t, ci.Put(1, 1))

	val, found := ci.Get(1)
	assert.True(t, found)
	assert.Equal(t, uint32(1), val)

	found = ci.Delete(1)
	assert.True(t, found)
	val, found = ci.Get(1)
	assert.False(t, found)
	assert.Equal(t, uint32(0), val)
}
