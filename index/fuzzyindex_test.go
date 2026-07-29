package index_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lima1909/mind"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
)

func TestFuzzyIndex_SpecialChar(t *testing.T) {
	l := mind.NewList[string]()
	assert.NoError(t, l.CreateIndex("w", index.NewFuzzyIndex(index.FromValue[string]())))

	words := []string{"Paul\\'s", "Alice"}
	for _, w := range words {
		l.Insert(w)
	}

	result, err := l.QueryStr(`w fuzzy "Alice"`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []string{"Alice"}, result)

	result, err = l.QueryStr(`w fuzzy "Paul\\'s"`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []string{"Paul\\'s"}, result)
}

func TestFuzzyIndex_BulkSet(t *testing.T) {
	l := mind.NewList[string]()
	assert.NoError(t, l.CreateIndex("w", index.NewFuzzyIndex(index.FromValue[string]())))

	words := []string{"cat", "bat", "hat", "dog", "car"}
	for _, w := range words {
		l.Insert(w)
	}

	result, err := l.Query(query.FuzzyDist("w", "cat", 1)).Values()
	assert.NoError(t, err)
	assert.Equal(t, []string{"cat", "bat", "hat", "car"}, result)
}

func TestFuzzyIndex_WithList(t *testing.T) {
	type Word struct{ W string }

	l := mind.NewList[Word]()
	assert.NoError(t, l.CreateIndex("w", index.NewFuzzyIndex(func(w *Word) string { return w.W })))

	for _, w := range []string{"microsoft", "apple", "google", "mikrosft", "microsft"} {
		l.Insert(Word{w})
	}

	result, err := l.Query(query.Fuzzy("w", "microsoft")).Values()
	assert.NoError(t, err)
	assert.Equal(t, []Word{{"microsoft"}, {"mikrosft"}, {"microsft"}}, result)
}

func TestFuzzyIndex_ParseQuery(t *testing.T) {
	type Word struct{ W string }

	l := mind.NewList[Word]()
	assert.NoError(t, l.CreateIndex("w", index.NewFuzzyIndex(func(w *Word) string { return w.W })))
	for _, w := range []string{"cat", "bat", "dog"} {
		l.Insert(Word{w})
	}

	// default distance 2
	result, err := l.QueryStr("w fuzzy 'cat'").Values()
	assert.NoError(t, err)
	assert.Equal(t, []Word{{"cat"}, {"bat"}}, result)

	// explicit distance via fuzzy("term", dist)
	result, err = l.QueryStr("w fuzzy('cat', 1)").Values()
	assert.NoError(t, err)
	assert.Equal(t, []Word{{"cat"}, {"bat"}}, result)
}
