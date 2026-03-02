package main

import "testing"

// func BenchmarkValueFromString_String(b *testing.B) {
// 	for b.Loop() {
// 		_, err := ValueFromString[string]("my strint")
// 		if err != nil {
// 			b.Fatalf("%s", err)
// 		}
// 	}
// }
//
// func BenchmarkValueFromString_StringPtr(b *testing.B) {
// 	for b.Loop() {
// 		_, err := ValueFromString[*string]("my strint")
// 		if err != nil {
// 			b.Fatalf("%s", err)
// 		}
// 	}
// }
//
// func BenchmarkValueFromString_Bool(b *testing.B) {
// 	for b.Loop() {
// 		_, err := ValueFromString[bool]("true")
// 		if err != nil {
// 			b.Fatalf("%s", err)
// 		}
// 	}
// }
//
// func BenchmarkValueFromString_Float(b *testing.B) {
// 	for b.Loop() {
// 		_, err := ValueFromString[float32]("123.456")
// 		if err != nil {
// 			b.Fatalf("%s", err)
// 		}
// 	}
// }
//
// func BenchmarkValueFromString_Uint(b *testing.B) {
// 	for b.Loop() {
// 		_, err := ValueFromString[uint]("123456")
// 		if err != nil {
// 			b.Fatalf("%s", err)
// 		}
// 	}
// }

func BenchmarkValueFromAny_Int64_Int(b *testing.B) {
	for b.Loop() {
		_, err := ValueFromAny[int](int64(-1234567890))
		if err != nil {
			b.Fatalf("%s", err)
		}
	}
}

func BenchmarkValueFromAny_Int_Int32(b *testing.B) {
	for b.Loop() {
		_, err := ValueFromAny[int32](-1234567890)
		if err != nil {
			b.Fatalf("%s", err)
		}
	}
}

func BenchmarkValueFromAny_Int32_Int32(b *testing.B) {
	for b.Loop() {
		_, err := ValueFromAny[int32](int32(-1234567890))
		if err != nil {
			b.Fatalf("%s", err)
		}
	}
}

func BenchmarkValueFromAny_Float64_Float32(b *testing.B) {
	for b.Loop() {
		_, err := ValueFromAny[float32](float64(-12345678.90))
		if err != nil {
			b.Fatalf("%s", err)
		}
	}
}
