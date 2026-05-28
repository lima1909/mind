package mind

import (
	"iter"
)

// check that PhoneticIndex implement Index
var _ Index[string] = &PhoneticIndex[string, SingleValueHandler[string, string]]{}

const PhoneticIndexName = "PhoneticIndex"

// PhoneticIndex indexes strings by their American Soundex phonetic code.
// Use the FOpSounds operator to find strings that sound similar (e.g. "Smith" finds "Smyth").
type PhoneticIndex[OBJ any, H ValueHandler[OBJ, string]] struct {
	codes     map[string]*RawIDs32
	soundexFn func(string) string
	handler   H
}

func NewPhoneticIndex[OBJ any](fieldGetFn FromField[OBJ, string]) Index[OBJ] {
	return &PhoneticIndex[OBJ, SingleValueHandler[OBJ, string]]{
		handler:   SingleValueHandler[OBJ, string]{fieldGetFn},
		soundexFn: soundex,
		codes:     make(map[string]*RawIDs32),
	}
}

func NewPhoneticIndexSlice[OBJ any](fieldGetFn FromFieldSlice[OBJ, string]) Index[OBJ] {
	return &PhoneticIndex[OBJ, MultiValueHandler[OBJ, string]]{
		handler:   MultiValueHandler[OBJ, string]{fieldGetFn},
		soundexFn: soundex,
		codes:     make(map[string]*RawIDs32),
	}
}

func (pi *PhoneticIndex[OBJ, H]) Set(obj *OBJ, lidx uint32) {
	pi.handler.Handle(obj, func(s string) {
		code := pi.soundexFn(s)
		if code == "" {
			return
		}
		ids, found := pi.codes[code]
		if !found {
			ids = NewRawIDs[uint32]()
			pi.codes[code] = ids
		}
		ids.Set(lidx)
	})
}

func (pi *PhoneticIndex[OBJ, H]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	batch := make(map[string][]uint32)
	for i, obj := range objs {
		pi.handler.Handle(obj, func(s string) {
			code := pi.soundexFn(s)
			if code != "" {
				batch[code] = append(batch[code], uint32(i))
			}
		})
	}

	for code, ids := range batch {
		pi.codes[code] = NewRawIDsFrom(ids...)
	}
}

func (pi *PhoneticIndex[OBJ, H]) UnSet(obj *OBJ, lidx uint32) {
	pi.handler.Handle(obj, func(s string) {
		code := pi.soundexFn(s)
		if ids, found := pi.codes[code]; found {
			ids.UnSet(lidx)
			if ids.Count() == 0 {
				delete(pi.codes, code)
			}
		}
	})
}

func (pi *PhoneticIndex[OBJ, H]) HasChanged(oldItem, newItem *OBJ) bool {
	return pi.handler.HasChanged(oldItem, newItem)
}

func (pi *PhoneticIndex[OBJ, H]) Equal(value any) (*RawIDs32, error) {
	s, err := ValueFromAny[string](value)
	if err != nil {
		return nil, InvalidValueTypeError[string]{value}
	}

	code := pi.soundexFn(s)
	ids, found := pi.codes[code]
	if !found {
		return NewRawIDs[uint32](), nil
	}
	return ids, nil
}

func (pi *PhoneticIndex[OBJ, H]) Match(_ *RawIDs32, op FilterOp, value any) (*RawIDs32, bool, error) {
	switch op.Op {
	case OpSounds:
		s, err := ValueFromAny[string](value)
		if err != nil {
			return nil, false, InvalidValueTypeError[string]{value}
		}
		code := pi.soundexFn(s)
		ids, found := pi.codes[code]
		if !found {
			return NewRawIDs[uint32](), true, nil
		}
		return ids, false, nil
	default:
		return nil, false, InvalidOperationError{PhoneticIndexName, op.Op}
	}
}

func (pi *PhoneticIndex[OBJ, H]) MatchMany(op FilterOp, _ ...any) (*RawIDs32, bool, error) {
	return nil, false, InvalidOperationError{PhoneticIndexName, op.Op}
}

// soundexTable maps letters 'a'-'z' to American Soundex digit codes.
// 0 = silent (vowels A E I O U, and H W Y which are transparent).
var soundexTable = "01230120022455012623010202"

// Soundex returns the Soundex code of s (4 characters, e.g. "S530").
// Only letters A-Z are considered; non‑letters are ignored.
// Empty string returns empty string.
func soundex(s string) string {
	var out [4]byte

	// find first alphabetic character
	first := -1
	for i := 0; i < len(s); i++ {
		c := s[i] | 0x20
		if c >= 'a' && c <= 'z' {
			first = i
			break
		}
	}
	if first == -1 {
		return ""
	}

	fc := s[first] &^ byte(0x20) // uppercase first letter
	out[0] = fc
	out[1] = '0'
	out[2] = '0'
	out[3] = '0'

	prevCode := soundexTable[fc|0x20-'a'] - '0'
	pos := 1

	for i := first + 1; i < len(s) && pos < 4; i++ {
		c := s[i] | 0x20
		if c < 'a' || c > 'z' {
			continue
		}
		code := soundexTable[c-'a'] - '0'
		if code == 0 {
			// vowels (code 0) are ignored, but H/W also reset prevCode
			if c != 'h' && c != 'w' {
				prevCode = 0
			}
			continue
		}
		if code == prevCode {
			continue
		}
		out[pos] = '0' + code
		pos++
		prevCode = code
	}

	return string(out[:4])
}

// colognePhonetics returns the Cologne Phonetics (Kölner Phonetik) code for s.
// This is the standard German phonetic algorithm, far more suitable for German
// names than American Soundex.
//
// The four steps of the algorithm:
//  1. Normalise: convert to uppercase; expand Ä→A, Ö→O, Ü→U, ß→SS.
//  2. Encode each letter to a digit based on phonetic context (see table below).
//  3. Remove consecutive duplicate digits.
//  4. Remove all '0' (vowel marker) digits except at the very start.
//
// Encoding table:
//
//	A E I J O U Y          → 0  (vowels / Y)
//	H                      → –  (silent, not coded)
//	B                      → 1
//	P  (not before H)      → 1
//	P  (before H, i.e. PH) → 3
//	D T  (not before C/S/Z)→ 2
//	D T  (before C/S/Z)    → 8
//	F V W                  → 3
//	G K Q                  → 4
//	C  (context-dependent) → 4 or 8  (see cologneC)
//	X  (not after C/K/Q)   → 4 then 8  (two digits)
//	X  (after C/K/Q)       → 8
//	L                      → 5
//	M N                    → 6
//	R                      → 7
//	S Z                    → 8
//
// Examples: "Müller"→"657", "Meier"/"Meyer"/"Maier"/"Mayer"→"67",
// "Schmidt"→"862", "Schneider"→"8627", "Wikipedia"→"3412".
// func colognePhonetics(s string) string {
func ColognePhonetics(s string) string {
	if len(s) == 0 {
		return ""
	}

	// normalize to uppercase A–Z, expanding German special characters.
	raw := []rune(s)
	norm := make([]rune, 0, len(raw)+4)
	for _, r := range raw {
		switch {
		case r >= 'A' && r <= 'Z':
			norm = append(norm, r)
		case r >= 'a' && r <= 'z':
			norm = append(norm, r-('a'-'A'))
		case r == 'Ä' || r == 'ä':
			norm = append(norm, 'A')
		case r == 'Ö' || r == 'ö':
			norm = append(norm, 'O')
		case r == 'Ü' || r == 'ü':
			norm = append(norm, 'U')
		case r == 'ß':
			norm = append(norm, 'S', 'S')
			// everything else (digits, punctuation, spaces) is skipped
		}
	}

	if len(norm) == 0 {
		return ""
	}

	// encode each character to its Cologne digit(s).
	codes := make([]byte, 0, len(norm)+4)
	for i, r := range norm {
		var prev, next rune
		if i > 0 {
			prev = norm[i-1]
		}
		if i < len(norm)-1 {
			next = norm[i+1]
		}

		switch r {
		case 'A', 'E', 'I', 'J', 'O', 'U', 'Y':
			codes = append(codes, '0')
		case 'H':
			// silent – not coded, not even a separator
		case 'B':
			codes = append(codes, '1')
		case 'P':
			if next == 'H' {
				codes = append(codes, '3') // PH sounds like F
			} else {
				codes = append(codes, '1')
			}
		case 'D', 'T':
			if next == 'C' || next == 'S' || next == 'Z' {
				codes = append(codes, '8')
			} else {
				codes = append(codes, '2')
			}
		case 'F', 'V', 'W':
			codes = append(codes, '3')
		case 'G', 'K', 'Q':
			codes = append(codes, '4')
		case 'C':
			// isFirst
			if i == 0 {
				switch next {
				case 'A', 'H', 'K', 'L', 'O', 'Q', 'R', 'U', 'X':
					codes = append(codes, '4')
				default:
					codes = append(codes, '8')
				}
			} else if prev == 'S' || prev == 'Z' {
				codes = append(codes, '8')
			} else {
				switch next {
				case 'A', 'H', 'K', 'O', 'Q', 'U', 'X':
					codes = append(codes, '4')
				default:
					codes = append(codes, '8')
				}
			}
		case 'X':
			// X = "ks": after C/K/Q the K-sound is already coded, so only S-sound remains.
			if prev == 'C' || prev == 'K' || prev == 'Q' {
				codes = append(codes, '8')
			} else {
				codes = append(codes, '4', '8') // two digits: K-sound + S-sound
			}
		case 'L':
			codes = append(codes, '5')
		case 'M', 'N':
			codes = append(codes, '6')
		case 'R':
			codes = append(codes, '7')
		case 'S', 'Z':
			codes = append(codes, '8')
		}
	}

	if len(codes) == 0 {
		return ""
	}

	// remove consecutive duplicate digits.
	deduped := make([]byte, 0, len(codes))
	deduped = append(deduped, codes[0])
	for i := 1; i < len(codes); i++ {
		if codes[i] != codes[i-1] {
			deduped = append(deduped, codes[i])
		}
	}

	// remove all '0' (vowel markers) except at position 0.
	out := make([]byte, 0, len(deduped))
	for i, b := range deduped {
		if b != '0' || i == 0 {
			out = append(out, b)
		}
	}

	return string(out)
}
