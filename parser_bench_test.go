package mind

import (
	"testing"
)

func BenchmarkLexer(b *testing.B) {

	for b.Loop() {
		l := &lexer{input: `role = "admin" OR status = 1 AND deleted = 1`, pos: 0}
		for l.nextToken().Op != OpEOF {
		}
	}
}

// GOGC=off go test -bench=Parser -cpuprofile=cpu.prof
// go tool pprof  cpu.prof
// go tool pprof -http=:8080 cpu.prof

func BenchmarkParser(b *testing.B) {
	user := User{name: "Alice", role: "admin", ok: false, price: 1.2}

	indexMap := newIndexMap[User, struct{}](nil)
	indexMap.index["name"] = NewSortedIndex((*User).Name)
	indexMap.index["name"].Set(&user, 1)
	indexMap.index["role"] = NewSortedIndex((*User).Role)
	indexMap.index["role"].Set(&user, 1)
	indexMap.index["price"] = NewMapIndex((*User).Price)
	indexMap.index["price"].Set(&User{}, 0)
	indexMap.index["ok"] = NewMapIndex((*User).Ok)
	indexMap.index["ok"].Set(&user, 1)

	b.ResetTimer()

	for b.Loop() {
		ast, err := Parse(`role = "admin" OR ok = false AND price = 0.0`)
		if err != nil {
			b.Fatal(err)
		}
		ast = ast.Optimize()
		query := ast.Compile(nil)

		bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
		if err != nil {
			b.Fatal(err)
		}
		if bs.ToSlice()[0] != 1 {
			b.Fatalf("expected: %v, got: %v", []uint32{1}, bs.ToSlice())
		}
	}
}
