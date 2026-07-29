package query_test

import (
	"testing"

	"github.com/lima1909/mind"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

func BenchmarkLexer(b *testing.B) {

	for b.Loop() {
		l := query.NewLexer(`role = "admin" OR status = 1 AND deleted = 1`)
		for l.NextToken().Op != query.OpEOF {
		}
	}
}

// GOGC=off go test -bench=Parser -cpuprofile=cpu.prof
// go tool pprof  cpu.prof
// go tool pprof -http=:8080 cpu.prof

func BenchmarkParser(b *testing.B) {
	user := User{name: "Alice", role: "admin", ok: false, price: 1.2}

	allIDs := lidx.NewRawIDsFrom[uint32](0, 1)
	indexMap := mind.NewIndexMap[User]()

	_ = indexMap.AddIndex("name", index.NewSortedIndex((*User).Name))
	_ = indexMap.AddIndex("role", index.NewSortedIndex((*User).Role))
	_ = indexMap.AddIndex("price", index.NewMapIndex((*User).Price))
	_ = indexMap.AddIndex("ok", index.NewMapIndex((*User).Ok))

	indexMap.Insert(&User{}, 0)
	indexMap.Insert(&user, 1)

	b.ResetTimer()

	for b.Loop() {
		ast, err := query.Parse(`role = "admin" OR ok = false AND price = 1.2`)
		if err != nil {
			b.Fatal(err)
		}
		ast = ast.Optimize()
		query := ast.Compile(nil)

		bs, _, err := query(indexMap.FilterByName, allIDs)
		if err != nil {
			b.Fatal(err)
		}
		if bs.ToSlice()[0] != 1 {
			b.Fatalf("expected: %v, got: %v", []uint32{1}, bs.ToSlice())
		}
	}
}
