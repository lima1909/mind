package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhonetic_Soundex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Basic examples from Soundex original algorithm
		{"Robert", "R163"},
		{"Rupert", "R163"},
		{"Rubin", "R150"},
		{"Ashcraft", "A261"},
		{"Ashcroft", "A261"},
		{"Tymczak", "T522"},
		{"Pfister", "P236"},

		// Common names
		{"Smith", "S530"},
		{"Smythe", "S530"},
		{"Johnson", "J525"},
		{"Williams", "W452"},
		{"Jones", "J520"},
		{"Brown", "B650"},
		{"Davis", "D120"},
		{"Miller", "M460"},
		{"Wilson", "W425"},
		{"Moore", "M600"},
		{"Taylor", "T460"},
		{"Anderson", "A536"},

		// Edge cases: empty string
		{"", ""},

		// Only non-letters
		{"123", ""},
		{"!@#$%", ""},
		{"123 abc", "A120"}, // first letter 'a' from "abc"

		// Single character
		{"A", "A000"},
		{"B", "B000"},
		{"Z", "Z000"},
		{"a", "A000"}, // lowercase
		{"z", "Z000"},

		// Two characters
		{"Ab", "A100"},
		{"Ay", "A000"},
		{"Ac", "A200"},
		{"Bz", "B200"},
		{"Bt", "B300"},
		{"Bd", "B300"},

		// Words with non-letter characters in the middle
		{"John Doe", "J530"},
		{"John-Doe", "J530"},
		{"John123Doe", "J530"},

		// H and W handling (they reset previous code but don't produce a digit)
		{"Hugh", "H200"},
		{"Ashworth", "A263"},
		{"Wright", "W623"},
		{"Lloyd", "L300"},
		{"Gough", "G200"},

		// Consecutive same codes
		{"Bobby", "B100"},
		{"Butter", "B360"},

		// Names with silent letters and tricky patterns
		{"Jackson", "J250"},
		{"Jackson", "J250"},

		// Non-letter at beginning
		{"123Smith", "S530"},
		{"_Jones", "J520"},

		// All letters are vowels
		{"Aeio", "A000"},
		{"Eau", "E000"},
		{"Yay", "Y000"},

		// Single vowel
		{"I", "I000"},

		// Strings with only H and W
		{"Hw", "H000"},
		{"Wh", "W000"},

		// Very long string (should only process first 4 digits)
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "A000"},

		// Mixed case
		{"Robert", "R163"},
		{"rObErT", "R163"},
		{"ROBERT", "R163"},

		// Special: 'Pfister' already tested, but ensure 'P' first
		{"pfister", "P236"},

		// Names with numbers
		{"R2D2", "R300"},

		// Edge: first character not letter but later letters - should find first letter
		{"!@#A123B", "A100"},
	}

	for _, tc := range tests {
		got := soundex(tc.input)
		assert.Equal(t, tc.want, soundex(tc.input), "soundex(%q) = %q, want %q", tc.input, got, tc.want)
	}
}

func TestGermanPhonetics_CommonGermanNames(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Müller", "657"},
		{"Wikipedia", "3412"},
		{"Breschnew", "17863"},
		{"Schmidt", "862"},
		{"Schmid", "862"},
		{"Schmitt", "862"},
		{"Schneider", "8627"},
		{"Fischer", "387"},
		{"Weber", "317"},
		{"Wagner", "3467"},
		{"Becker", "147"},
		{"Schulz", "858"},
		{"Hoffmann", "0366"},
		{"Schäfer", "837"},
		{"Koch", "44"},
		{"Richter", "7427"},
		{"Klein", "456"},
		{"Wolf", "353"},
		{"Schröder", "8727"},
		{"Neumann", "666"},
		{"Braun", "176"},
		{"Zimmermann", "86766"},
		{"Köhler", "457"},
		{"Ärger", "0747"},
		{"Öl", "05"},
		{"Übung", "0164"},
		{"Strauß", "8278"},
		{"MÜLLER", "657"},
		{"müller", "657"},
		{"Müller", "657"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, ColognePhonetics(tc.input), "colognePhonetics(%q)", tc.input)
	}
}

func TestGermanPhonetics_Variants(t *testing.T) {
	sameCode := [][]string{
		{"Meier", "Meyer", "Maier", "Mayer"},
		{"Schmidt", "Schmid", "Schmitt"},
		{"Schreiber", "Schrieber"},
		{"Groß", "Gross"},
		{"Müller", "Mueller"},
		{"Köhler", "Koehler"},
		{"Schäfer", "Schaefer"},
	}
	for _, group := range sameCode {
		code := ColognePhonetics(group[0])
		for _, name := range group[1:] {
			assert.Equal(t, code, ColognePhonetics(name),
				"%q and %q should have the same Cologne code", group[0], name)
		}
	}
}

func TestGermanPhonetics_ContextRules(t *testing.T) {
	tests := []struct {
		input string
		want  string
		desc  string
	}{
		{"Philipp", "351", "PH → 3 (F-sound)"},
		{"Phönix", "3648", "PH at start + ö→o + X=48"},
		{"Deutsch", "28", "D→2, T before S→8, SCH collapses"},
		{"Fritz", "378", "T before Z → 8"},
		{"Carl", "475", "initial C before A → 4"},
		{"Cäsar", "487", "initial C before A (Ä→A) → 4"},
		{"Szene", "86", "S+Z both→8 collapse, vowels removed"},
		{"Xaver", "4837", "X at start → 4 then 8"},
		{"Hexe", "048", "X not after C/K/Q → 4 then 8 (vowel-start kept)"},
	}
	for _, tc := range tests {
		got := ColognePhonetics(tc.input)
		assert.Equal(t, tc.want, got, "colognePhonetics(%q) [%s]", tc.input, tc.desc)
	}
}

func TestGermanPhonetics_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},               // empty string
		{"123", ""},            // digits only → nothing
		{"!@#", ""},            // punctuation only → nothing
		{"H", ""},              // H is silent, produces no code
		{"HH", ""},             // multiple silent letters
		{"A", "0"},             // single vowel: 0 kept at position 0
		{"Ä", "0"},             // umlaut vowel at start
		{"Anna", "06"},         // vowel start kept, duplicates removed
		{"Aaa", "0"},           // all-vowel word
		{"hans", "068"},        // h silent, a→0, n→6, s→8
		{"Schmidt 123", "862"}, // spaces and digits ignored
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, ColognePhonetics(tc.input), "colognePhonetics(%q)", tc.input)
	}
}

func TestPhoneticIndex_SoundsLike(t *testing.T) {
	type Person struct{ Name string }

	idx := NewPhoneticIndex(func(p *Person) string { return p.Name })

	names := []string{
		"Meier",     // 0  → "67"
		"Meyer",     // 1  → "67"  (same code)
		"Maier",     // 2  → "67"  (same code)
		"Mayer",     // 3  → "67"  (same code)
		"Schmidt",   // 4  → "862"
		"Schmid",    // 5  → "862" (same code)
		"Schneider", // 6  → "8627"
		"Müller",    // 7  → "657"
		"Mueller",   // 8  → "657" (same code)
	}
	for i, name := range names {
		p := Person{name}
		idx.Set(&p, uint32(i))
	}

	// Meier → "67": should find indices 0-3
	ids, canMut, err := idx.Match(nil, FOpSounds, "Meier")
	assert.NoError(t, err)
	assert.False(t, canMut)
	assert.Equal(t, []uint32{0, 1, 2, 3}, ids.ToSlice())

	// Schmidt → "862": should find 4 and 5
	ids, _, err = idx.Match(nil, FOpSounds, "Schmidt")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{4, 5}, ids.ToSlice())

	// Müller / Mueller → "657": should find 7 and 8
	ids, _, err = idx.Match(nil, FOpSounds, "Müller")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{7, 8}, ids.ToSlice())

	// No match
	ids, _, err = idx.Match(nil, FOpSounds, "Zzzz")
	assert.NoError(t, err)
	assert.Equal(t, 0, ids.Count())
}

func TestPhoneticIndex_UnSet(t *testing.T) {
	type Person struct{ Name string }

	idx := NewPhoneticIndex(func(p *Person) string { return p.Name })
	meier := Person{"Meier"}
	meyer := Person{"Meyer"}
	idx.Set(&meier, 0)
	idx.Set(&meyer, 1)

	ids, _, _ := idx.Match(nil, FOpSounds, "Meier")
	assert.Equal(t, []uint32{0, 1}, ids.ToSlice())

	idx.UnSet(&meier, 0)
	ids, _, _ = idx.Match(nil, FOpSounds, "Meier")
	assert.Equal(t, []uint32{1}, ids.ToSlice())
}

func TestPhoneticIndex_Equal(t *testing.T) {
	type Person struct{ Name string }

	idx := NewPhoneticIndex(func(p *Person) string { return p.Name })
	p := Person{"Schmidt"}
	idx.Set(&p, 42)

	// Equal also uses the phonetic code, so a variant spelling finds the same entry.
	ids, err := idx.Equal("Schmitt")
	assert.NoError(t, err)
	assert.True(t, ids.Contains(42))
}

func TestPhoneticIndex_InvalidOp(t *testing.T) {
	type Person struct{ Name string }
	idx := NewPhoneticIndex(func(p *Person) string { return p.Name })
	_, _, err := idx.Match(nil, FOpLike, "x")
	assert.Error(t, err)
}

func TestPhoneticIndex_WithList(t *testing.T) {
	l := NewList[string]()
	assert.NoError(t, l.CreateIndex("name", NewPhoneticIndex(FromValue[string]())))

	l.Insert("Meier")
	l.Insert("Meyer")
	l.Insert("Maier")
	l.Insert("Mayer")
	l.Insert("Schmidt")

	// All four Meier-variants should be found via any of the spellings.
	result, err := l.Query(Sounds("name", "Meier")).Values()
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))

	result, err = l.Query(Sounds("name", "Schmitt")).Values()
	assert.NoError(t, err)
	assert.Equal(t, []string{"Schmidt"}, result)
}

func TestGermanPhoneticIndex_ParseQuery(t *testing.T) {
	l := NewList[string]()
	assert.NoError(t, l.CreateIndex("name",
		NewPhoneticIndex(FromValue[string]())))
	l.Insert("Müller")
	l.Insert("Mueller")
	l.Insert("Schmidt")

	result, err := l.QueryStr("name sounds 'Müller'").Values()
	assert.NoError(t, err)
	assert.Equal(t, []string{"Müller", "Mueller"}, result)

	result, err = l.Query(Sounds("name", "'Müller'")).Values()
	assert.NoError(t, err)
	assert.Equal(t, []string{"Müller", "Mueller"}, result)
}

func TestPhoneticIndex_BulkSet(t *testing.T) {
	type Person struct{ Name string }

	names := []Person{
		{"Meier"}, {"Meyer"}, {"Schmidt"}, {"Müller"}, {"Mueller"},
	}

	idx := NewPhoneticIndex(func(p *Person) string { return p.Name })
	idx.BulkSet(func(yield func(int, *Person) bool) {
		for i := range names {
			if !yield(i, &names[i]) {
				return
			}
		}
	})

	ids, _, err := idx.Match(nil, FOpSounds, "Meier")
	assert.NoError(t, err)
	assert.Equal(t, 2, ids.Count()) // Meier + Meyer

	ids, _, err = idx.Match(nil, FOpSounds, "Mueller")
	assert.NoError(t, err)
	assert.Equal(t, 2, ids.Count()) // Müller + Mueller
}

// TestCologneGroupsGermanVariants verifies that Cologne Phonetics reliably
// equates German spelling variants arising from:
//   - umlaut ↔ digraph substitution (Ü↔UE, Ö↔OE, Ä↔AE)
//   - ß ↔ ss
//   - diphthong permutations (EI/AI/EY/AY)
//   - double consonant vs single (Schmidt / Schmitt)
func TestCologneGroupsGermanVariants(t *testing.T) {
	pairs := [][2]string{
		{"Müller", "Mueller"},   // umlaut ↔ digraph → both "657"
		{"Köhler", "Koehler"},   // umlaut ↔ digraph → both "457"
		{"Schäfer", "Schaefer"}, // umlaut ↔ digraph → both "837"
		{"Groß", "Gross"},       // ß ↔ ss → both "478"
		{"Meier", "Meyer"},      // ei ↔ ey diphthong → both "67"
		{"Meier", "Maier"},      // ei ↔ ai diphthong → both "67"
		{"Schmidt", "Schmitt"},  // t ↔ tt double consonant → both "862"
	}
	for _, pair := range pairs {
		assert.Equal(t,
			ColognePhonetics(pair[0]),
			ColognePhonetics(pair[1]),
			"%q and %q should have the same Cologne code", pair[0], pair[1],
		)
	}
}
