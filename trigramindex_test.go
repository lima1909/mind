package mind

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrigram_Base(t *testing.T) {
	ti := NewTrigramIndexFrom("apple", "apply", "ban", "banana", "xapp")
	assert.Equal(t, 5, ti.Len())
	assert.Equal(t, []uint32{0, 1, 4}, ti.Get("app").ToSlice())
	assert.Equal(t, []uint32{2, 3}, ti.Get("an").ToSlice())
	// not found
	assert.Equal(t, []uint32{}, ti.Get("nix").ToSlice())

	assert.True(t, ti.Delete(2))
	assert.Equal(t, 4, ti.Len())
	// search with 2 letters
	assert.Equal(t, []uint32{3}, ti.Get("an").ToSlice())

	// grow the bucket list
	ti.Put("xban", 20)
	assert.Equal(t, 5, ti.Len())
	assert.Equal(t, []uint32{3, 20}, ti.Get("ban").ToSlice())

	// reuse index 2
	ti.Put("xappx", 2)
	assert.Equal(t, 6, ti.Len())
	assert.Equal(t, []uint32{0, 1, 2, 4}, ti.Get("app").ToSlice())

	// checks the false positive: ABCD and BCDE matching {0, 2}
	ti = NewTrigramIndexFrom("ABCD", "ZZZ", "BCDE")
	assert.Equal(t, []uint32{}, ti.Get("ABCDE").ToSlice())

	// empty init
	ti = NewTrigramIndex()
	assert.Equal(t, 0, ti.Len())
	assert.Equal(t, []uint32{}, ti.Get("nix").ToSlice())

	ti.Put("üöß€ä@", 2)
	assert.Equal(t, 1, ti.Len())
	assert.Equal(t, []uint32{2}, ti.Get("öß€ä").ToSlice())
}

func TestTrigram_abc(t *testing.T) {
	ti := NewTrigramIndex()
	ti.Put("abc---bcd", 0)
	assert.Equal(t, 1, ti.Len())

	r := ti.Get("abcd")
	assert.Equal(t, 0, r.Count())

	r = ti.Get("abc")
	assert.Equal(t, 1, r.Count())
	r = ti.Get("bcd")
	assert.Equal(t, 1, r.Count())

	ti.Put("abc---abc", 1)
	assert.Equal(t, 2, ti.Len())

	assert.Equal(t, []uint32{0, 1}, ti.Get("abc").ToSlice())
	assert.Equal(t, []uint32{1}, ti.Get("--ab").ToSlice())
}

func TestNewTrigram_Initializers(t *testing.T) {
	t.Run("Default Initialization", func(t *testing.T) {
		ti := NewTrigramIndex()
		assert.Equal(t, 0, ti.Len())
	})

	t.Run("With Capacity", func(t *testing.T) {
		ti := NewTrigramIndexWithCapacity(50)
		assert.Equal(t, 50, cap(ti.buckets))
	})

	t.Run("From Variadic Strings", func(t *testing.T) {
		ti := NewTrigramIndexFrom("cat", "dog", "mouse")
		assert.Equal(t, 3, ti.Len())
		assert.True(t, ti.buckets[0].occupied)
		assert.Equal(t, "cat", ti.buckets[0].str)
	})
}

func TestTrigram_Put_EdgeCases(t *testing.T) {
	t.Run("Bucket Expansion Mechanics", func(t *testing.T) {
		ti := NewTrigramIndex()

		// Force standard expansion branch
		ti.Put("apple", 5)
		assert.Equal(t, 1, ti.Len())
		assert.Equal(t, 6, len(ti.buckets))

		// Hit the 'else' branch when within capacity but larger than current slice len
		ti.Put("banana", 3)
		assert.Equal(t, "banana", ti.buckets[3].str)
		assert.Equal(t, 2, ti.Len())

		// Overwrite an already occupied bucket
		ti.Put("apricot", 5)
		assert.Equal(t, "apricot", ti.buckets[5].str)
		assert.Equal(t, 2, ti.Len())
	})

	t.Run("String Length Limits", func(t *testing.T) {
		ti := NewTrigramIndex()

		// Strings < 3 characters shouldn't be packed into maps
		ti.Put("go", 0)
		assert.Equal(t, 0, len(ti.rawIDs))

		// Empty string handling
		ti.Put("", 1)
		assert.Equal(t, 2, ti.Len())
	})
}

func TestTrigram_Get_AllBranches(t *testing.T) {
	ti := NewTrigramIndexFrom("banana", "cabana", "bandana", "an", "abc")

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantIDs   []uint32
	}{
		{
			name:      "Empty String Short Circuit",
			query:     "",
			wantCount: 0,
			wantIDs:   []uint32{},
		},
		{
			name:      "Short Query < 3 Chars Full Scan Match",
			query:     "an",
			wantCount: 4, // banana, cabana, bandana, an
			wantIDs:   []uint32{0, 1, 2, 3},
		},
		{
			name:      "Short Query < 3 Chars Full Scan Miss",
			query:     "xyz",
			wantCount: 0,
			wantIDs:   []uint32{},
		},
		{
			name:      "Exact Length 3 Match (No Verification Loop)",
			query:     "abc",
			wantCount: 1,
			wantIDs:   []uint32{4},
		},
		{
			name:      "Exact Length 3 Miss",
			query:     "fff",
			wantCount: 0,
			wantIDs:   []uint32{},
		},
		{
			name:      "Missing Trigram Early Termination",
			query:     "banz", // 'anz' doesn't exist anywhere
			wantCount: 0,
			wantIDs:   []uint32{},
		},
		{
			name:      "Long Query with Rarest Sort & Intersection Check",
			query:     "banana",
			wantCount: 1,
			wantIDs:   []uint32{0},
		},
		{
			name:      "False Positive Order Verification Rejection",
			query:     "nabana", // contains 'nab', 'aba', 'ban', 'ana' but in wrong order
			wantCount: 0,
			wantIDs:   []uint32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ti.Get(tt.query)
			slice := res.ToSlice()
			assert.Equal(t, tt.wantCount, len(slice))

			for _, id := range tt.wantIDs {
				assert.True(t, res.Contains(id))
			}
		})
	}
}

func TestTrigram_Delete(t *testing.T) {
	ti := NewTrigramIndexFrom("testing", "tester")

	t.Run("Delete Non-Existent or Invalid Indices", func(t *testing.T) {
		if ti.Delete(99) {
			t.Error("Delete on out-of-bounds index should return false")
		}
		if ti.Delete(4) {
			t.Error("Delete on unallocated index path should return false")
		}
	})

	t.Run("Clean Up Maps and Trigger Shrinking", func(t *testing.T) {
		// Verify initial tracking state
		if ti.Len() != 2 {
			t.Fatalf("Setup length expected 2, got %d", ti.Len())
		}

		// Delete 'testing' (ID 0)
		if !ti.Delete(0) {
			t.Error("Failed to delete valid index 0")
		}

		if ti.Len() != 1 {
			t.Errorf("Expected length 1 after deletion, got %d", ti.Len())
		}

		// Ensure 'testing' unique trigram keys (like 'ing') were fully deleted from the map
		ingTri := pack('i', 'n', 'g')
		if _, found := ti.rawIDs[ingTri]; found {
			t.Error("Unique trigram key should be completely expunged from map when bitset count hits 0")
		}

		// Delete 'tester' (ID 1) - This exercises the map cleaning path fully
		if !ti.Delete(1) {
			t.Error("Failed to delete valid index 1")
		}

		if len(ti.rawIDs) != 0 {
			t.Errorf("Expected rawIDs map to be completely empty, still contains %d items", len(ti.rawIDs))
		}
	})
}

func TestTrigram_Len3Immunity(t *testing.T) {
	ti := NewTrigramIndexFrom("axbxc", "ab c", "abc")
	// This will lookup exactly one key. It will ONLY find ID 2 ("abc").
	res := ti.Get("abc")
	assert.Equal(t, []uint32{2}, res.ToSlice())
	assert.Equal(t, ti.buckets[2].str, "abc")

	ti = NewTrigramIndexFrom("cats", "catmats")
	res = ti.Get("cats")
	assert.Equal(t, []uint32{0}, res.ToSlice())
}

func TestTrigram_UnicodeAndUTF8(t *testing.T) {
	ti := NewTrigramIndexFrom("你好吗", "👋🚀", "Go语言", "café")

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantIDs   []uint32
	}{
		{
			name:      "Exact Match for 3-Byte Chinese Character",
			query:     "吗", // Exactly 3 bytes. Triggers the len(s)==3 optimization path safely.
			wantCount: 1,
			wantIDs:   []uint32{0},
		},
		{
			name:      "Substring Match Spanning Multi-byte Chinese Characters",
			query:     "你好", // 6 bytes. Generates 4 byte-trigrams.
			wantCount: 1,
			wantIDs:   []uint32{0},
		},
		{
			name:      "Single 4-Byte Emoji Query",
			query:     "🚀", // 4 bytes. Bypasses the <3 byte full-scan path and hits the map index.
			wantCount: 1,
			wantIDs:   []uint32{1},
		},
		{
			name:      "Straddling ASCII and Unicode Boundary",
			query:     "o语", // 'o' (1 byte) + '语' (3 bytes) = 4 bytes total.
			wantCount: 1,
			wantIDs:   []uint32{2},
		},
		{
			name:      "Accented Multi-byte Character at End",
			query:     "fé", // 'f' (1 byte) + 'é' (2 bytes) = 3 bytes total. Triggers len(s)==3 path.
			wantCount: 1,
			wantIDs:   []uint32{3},
		},
		{
			name:      "Unicode Substring Miss",
			query:     "再见", // 6 bytes of Chinese characters not in index.
			wantCount: 0,
			wantIDs:   []uint32{},
		},
		{
			name:      "Short-scan Path for 2-Byte Accent Fragment",
			query:     "é", // 2 bytes. Triggers the len(s) < 3 sequential full table scan path.
			wantCount: 1,
			wantIDs:   []uint32{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ti.Get(tt.query)
			assert.Equal(t, tt.wantCount, res.Count())
			assert.Equal(t, tt.wantIDs, res.ToSlice())
		})
	}
}

func TestTrigram_BulkPut(t *testing.T) {
	data := slices.All([]*string{ptr("apple"), ptr("apply"), ptr("ban"), ptr("banana"), ptr("xapp")})
	ti := NewTrigramIndex()
	handler := SingleValueHandler[string, string]{func(s *string) string { return *s }}
	TrigramIndexBulkPut(&ti, handler, data)

	assert.Equal(t, 5, ti.Len())
	assert.Equal(t, []uint32{0, 1, 4}, ti.Get("app").ToSlice())
	assert.Equal(t, []uint32{2, 3}, ti.Get("an").ToSlice())
	// not found
	assert.Equal(t, []uint32{}, ti.Get("nix").ToSlice())
}

func TestTrigram_BulkPut2(t *testing.T) {
	ti := NewTrigramIndex() // Empty map initialization verification

	// Mock payload collection
	items := map[int]*string{
		0:  ptr("alpha"),
		1:  ptr("beta"),
		10: ptr("omega"), // Forces large structural exponential growth jump
	}

	// Convert native map to modern Go iter.Seq2
	seq := func(yield func(int, *string) bool) {
		for k, v := range items {
			if !yield(k, v) {
				return
			}
		}
	}

	handler := SingleValueHandler[string, string]{func(s *string) string { return *s }}
	TrigramIndexBulkPut(&ti, handler, seq)

	assert.Equal(t, 11, ti.Len())
	assert.Equal(t, []uint32{1}, ti.Get("bet").ToSlice())

}

func ptr(s string) *string {
	return &s
}
