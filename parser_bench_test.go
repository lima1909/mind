package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkLexer(b *testing.B) {

	for b.Loop() {
		l := &lexer{input: `role = "admin" OR status = 1 AND deleted = 1`, pos: 0}
		for l.nextToken().Op != OpEOF {
		}
	}
}

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
		query, err := Parse(`role = "admin" OR ok = false AND price = 0.0`)
		assert.NoError(b, err)

		bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
		assert.NoError(b, err)
		assert.Equal(b, []uint32{1}, bs.ToSlice())
	}
}
