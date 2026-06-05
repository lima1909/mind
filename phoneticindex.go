package mind

import (
	"iter"
)

const PhoneticIndexName = "PhoneticIndex"

// PhoneticIndex indexes strings by their American Soundex phonetic code.
// Use the FOpSounds operator to find strings that sound similar (e.g. "Smith" finds "Smyth").
//
// A Soundex code is always exactly 4 bytes (a letter + 3 digits), so it is stored
// packed into a uint32: this keeps coding allocation-free and makes the map lookups
// integer-keyed. A code of 0 means "no phonetic code" (empty / non-alphabetic input).
type PhoneticIndex[OBJ any, H ValueHandler[OBJ, string]] struct {
	codes     map[uint32]*RawIDs32
	soundexFn func(string) uint32
	handler   H
}

func NewPhoneticIndex[OBJ any](fieldGetFn FromField[OBJ, string]) Index[OBJ] {
	return &PhoneticIndex[OBJ, SingleValueHandler[OBJ, string]]{
		handler:   SingleValueHandler[OBJ, string]{fieldGetFn},
		soundexFn: soundex,
		codes:     make(map[uint32]*RawIDs32),
	}
}

func NewPhoneticIndexSlice[OBJ any](fieldGetFn FromFieldSlice[OBJ, string]) Index[OBJ] {
	return &PhoneticIndex[OBJ, MultiValueHandler[OBJ, string]]{
		handler:   MultiValueHandler[OBJ, string]{fieldGetFn},
		soundexFn: soundex,
		codes:     make(map[uint32]*RawIDs32),
	}
}

func (pi *PhoneticIndex[OBJ, H]) Set(obj *OBJ, lidx uint32) {
	pi.handler.Handle(obj, func(s string) {
		code := pi.soundexFn(s)
		if code == 0 {
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
	batch := make(map[uint32][]uint32)
	for i, obj := range objs {
		pi.handler.Handle(obj, func(s string) {
			code := pi.soundexFn(s)
			if code != 0 {
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
	return nil, InvalidOperationError{PhoneticIndexName, OpEq}
}

func (pi *PhoneticIndex[OBJ, H]) Match(_ *RawIDs32, op FilterOp, value any) (*RawIDs32, bool, error) {
	// only support for Sounds match
	if op.Op != OpSounds {
		return nil, false, InvalidOperationError{PhoneticIndexName, op.Op}
	}

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
}

func (pi *PhoneticIndex[OBJ, H]) MatchMany(op FilterOp, _ ...any) (*RawIDs32, bool, error) {
	return nil, false, InvalidOperationError{PhoneticIndexName, op.Op}
}

// soundexTable maps letters 'a'-'z' to American Soundex digit codes.
// 0 = silent (vowels A E I O U, and H W Y which are transparent).
var soundexTable = "01230120022455012623010202"

// soundex returns the Soundex code of s packed into a uint32 (the 4 code bytes
// in big-endian order, e.g. "S530" -> 'S'<<24 | '5'<<16 | '3'<<8 | '0').
// Only letters A-Z are considered; non-letters are ignored. Input without any
// letter returns 0, which is a safe sentinel because a real code's high byte is
// always an uppercase letter and therefore never zero.
func soundex(s string) uint32 {
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
		return 0
	}

	fc := s[first] &^ byte(0x20) // uppercase first letter
	out := [4]byte{fc, '0', '0', '0'}

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

	return uint32(out[0])<<24 | uint32(out[1])<<16 | uint32(out[2])<<8 | uint32(out[3])
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
//
// The code is returned packed into a uint32 so it can key a PhoneticIndex without
// allocating. The digits are stored as 4-bit nibbles behind a leading 1-nibble
// marker (so length and leading zeros survive): "657" -> 0x1657, "06" -> 0x106.
// This holds up to 7 digits, which is enough for any realistic name; longer codes
// are truncated. Empty / non-alphabetic input returns 0.
func ColognePhonetics(s string) uint32 {
	if len(s) == 0 {
		return 0
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
		return 0
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
		return 0
	}

	// Pack the digits into a uint32, in a single pass that also drops consecutive
	// duplicates and drops '0' (vowel markers) except at the first deduped position.
	// A leading 1-nibble marker preserves length and leading zeros.
	const maxDigits = 7 // uint32 holds the marker + 7 nibbles
	packed := uint32(1)
	emitted := 0
	prev := byte(0xFF) // sentinel that never equals a code byte, so codes[0] is kept
	pos := -1          // index within the deduped stream
	for _, b := range codes {
		if b == prev {
			continue // collapse consecutive duplicates
		}
		prev = b
		pos++
		if b == '0' && pos != 0 {
			continue // drop vowel markers except at the very start
		}
		packed = packed<<4 | uint32(b-'0')
		if emitted++; emitted == maxDigits {
			break
		}
	}

	if emitted == 0 {
		return 0
	}
	return packed
}
