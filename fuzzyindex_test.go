package mind

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
		{"abc", "bc", 1},
		{"microsoft", "mikrosft", 2},
	}
	for _, tc := range tests {
		got := levenshtein(tc.a, tc.b)
		assert.Equal(t, tc.want, got, "levenshtein(%q, %q)", tc.a, tc.b)
	}
}

func TestFuzzyIndex_SpecialChar(t *testing.T) {
	l := NewList[string]()
	assert.NoError(t, l.CreateIndex("w", NewFuzzyIndex(FromValue[string]())))

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

func TestFuzzyIndex_Match(t *testing.T) {
	type Word struct{ W string }

	idx := NewFuzzyIndex(func(w *Word) string { return w.W })

	words := []string{"microsoft", "apple", "google", "amazon", "facebook", "mikrosft"}
	for i, w := range words {
		word := Word{w}
		idx.Set(&word, uint32(i))
	}

	// distance 2: "microsoft" finds "microsoft"(0) and "mikrosft"(5)
	ids, canMut, err := idx.Match(nil, FOpFuzzy, "microsoft")
	assert.NoError(t, err)
	assert.True(t, canMut)
	assert.True(t, ids.Contains(0), "should find microsoft")
	assert.True(t, ids.Contains(5), "should find mikrosft")
	assert.False(t, ids.Contains(1), "should not find apple")

	// no match
	ids, _, err = idx.Match(nil, FOpFuzzy, "zzzzzzzzz")
	assert.NoError(t, err)
	assert.Equal(t, 0, ids.Count())
}

func TestFuzzyIndex_Match2(t *testing.T) {
	idx := NewFuzzyIndex(FromValue[string]())

	words := []string{"Stephen", "Steve", "Seven"}
	for i, w := range words {
		idx.Set(&w, uint32(i))
	}

	ids, _, err := idx.Match(nil, FOpFuzzy, "Stefen")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{0, 1, 2}, ids.ToSlice())
}

func TestFuzzyIndex_MatchMany_CustomDist(t *testing.T) {
	type Word struct{ W string }

	idx := NewFuzzyIndex(func(w *Word) string { return w.W })
	words := []string{"cat", "bat", "hat", "dog", "car"}
	for i, w := range words {
		word := Word{w}
		idx.Set(&word, uint32(i))
	}

	// distance 1: "cat" finds cat(0), bat(1), hat(2), car(4)
	ids, _, err := idx.MatchMany(FOpFuzzy, "cat", 1)
	assert.NoError(t, err)
	assert.True(t, ids.Contains(0))  // cat
	assert.True(t, ids.Contains(1))  // bat
	assert.True(t, ids.Contains(2))  // hat
	assert.True(t, ids.Contains(4))  // car
	assert.False(t, ids.Contains(3)) // dog (distance 3)

	// distance 0: exact only
	ids, _, err = idx.MatchMany(FOpFuzzy, "cat", 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, ids.Count())
	assert.True(t, ids.Contains(0))
}

func TestFuzzyIndex_UnSet(t *testing.T) {
	type Word struct{ W string }

	idx := NewFuzzyIndex(func(w *Word) string { return w.W })
	w1 := Word{"cat"}
	w2 := Word{"cat"} // same word, different ID
	idx.Set(&w1, 0)
	idx.Set(&w2, 1)

	ids, _, _ := idx.Match(nil, FOpFuzzy, "cat")
	assert.Equal(t, 2, ids.Count())

	idx.UnSet(&w1, 0)
	ids, _, _ = idx.Match(nil, FOpFuzzy, "cat")
	assert.Equal(t, 1, ids.Count())
	assert.True(t, ids.Contains(1))
}

func TestFuzzyIndex_BulkSet(t *testing.T) {
	l := NewList[string]()
	assert.NoError(t, l.CreateIndex("w", NewFuzzyIndex(FromValue[string]())))

	words := []string{"cat", "bat", "hat", "dog", "car"}
	for _, w := range words {
		l.Insert(w)
	}

	result, err := l.Query(FuzzyDist("w", "cat", 1)).Values()
	assert.NoError(t, err)
	assert.Equal(t, []string{"cat", "bat", "hat", "car"}, result)
}

func TestFuzzyIndex_WithList(t *testing.T) {
	type Word struct{ W string }

	l := NewList[Word]()
	assert.NoError(t, l.CreateIndex("w", NewFuzzyIndex(func(w *Word) string { return w.W })))

	for _, w := range []string{"microsoft", "apple", "google", "mikrosft", "microsft"} {
		l.Insert(Word{w})
	}

	result, err := l.Query(Fuzzy("w", "microsoft")).Values()
	assert.NoError(t, err)
	assert.Equal(t, []Word{{"microsoft"}, {"mikrosft"}, {"microsft"}}, result)
}

func TestFuzzyIndex_ParseQuery(t *testing.T) {
	type Word struct{ W string }

	l := NewList[Word]()
	assert.NoError(t, l.CreateIndex("w", NewFuzzyIndex(func(w *Word) string { return w.W })))
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

func BenchmarkFuzzy_Phonetic_Index(b *testing.B) {
	type person struct {
		Name string
		Age  uint8
	}

	ds := 3_000_000
	n := 0
	names := strings.Split(names_txt, "\n")

	start := time.Now()
	fl := NewFreeListWithCapacity[person](ds)

	for i := 1; i <= ds; i++ {
		if n%6779 == 0 {
			n = 0
		}
		n++

		fl.Insert(person{
			Name: names[n],
			Age:  uint8(i % 100),
		})
	}

	l := NewList[person]()
	err := l.CreateIndex("name", NewFuzzyIndex(FromName[person, string]("Name")))
	require.NoError(b, err)
	err = l.CreateIndex("name2", NewPhoneticIndex(FromName[person, string]("Name")))
	require.NoError(b, err)

	err = l.InitialBulkInsert(fl)
	require.NoError(b, err)

	fmt.Printf("- Count: %d, Time: %s\n", l.Count(), time.Since(start))

	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func() int
		count int
	}{
		{
			name: "fuzzy",
			bmark: func() int {
				count, err := l.Query(Fuzzy("name", "Annetta")).Count()
				require.NoError(b, err)
				return count
			},
			count: 3543,
		},
		{
			name: "phonetic",
			bmark: func() int {
				count, err := l.Query(Sounds("name2", "Annetta")).Count()
				require.NoError(b, err)
				return count
			},
			count: 3544,
		},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				count := bench.bmark()
				if count != bench.count {
					b.Fatalf("%s: expected count %d, got %d", bench.name, bench.count, count)
				}
			}
		})
	}
}
