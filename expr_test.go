package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createIndexMap() indexMap[User, struct{}] {
	indexMap := indexMap[User, struct{}]{
		index:  make(map[string]Index[User]),
		allIDs: NewRawIDs[uint32](),
	}
	indexMap.index["name"] = NewSortedIndex((*User).Name)
	indexMap.index["name"].Set(&User{name: "a"}, 0)
	indexMap.index["name"].Set(&User{name: "b"}, 1)
	indexMap.index["name"].Set(&User{name: "c"}, 2)
	indexMap.index["price"] = NewSortedIndex((*User).Price)
	indexMap.index["price"].Set(&User{price: 1}, 0)
	indexMap.index["price"].Set(&User{price: 2}, 1)
	indexMap.index["price"].Set(&User{price: 0}, 2)

	return indexMap
}

func TestExpr_Trace(t *testing.T) {
	indexMap := createIndexMap()

	tracer := &Tracer{}
	nameEq := TermExpr{Field: "name", Op: FOpEq, Value: "a"}
	query := nameEq.Compile(tracer)

	bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
	require.NoError(t, err)
	assert.True(t, bs.Count() > 0)

	// fmt.Println(tracer)
}

// func TestExpr_TraceAnd(t *testing.T) {
// 	indexMap := createIndexMap()
//
// 	tracer := &Tracer{}
// 	left := TermExpr{Field: "name", Op: FOpEq, Value: "b"}
// 	right := TermExpr{Field: "price", Op: FOpEq, Value: 2.}
// 	and := BinaryExpr{Ekind: ExprAnd, Left: left, Right: right}
// 	query := and.Compile(tracer)
//
// 	bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
// 	require.NoError(t, err)
// 	assert.True(t, bs.Count() > 0)
//
// 	fmt.Println(tracer)
// }

// func TestExpr_TraceOr(t *testing.T) {
// 	indexMap := createIndexMap()
//
// 	tracer := &Tracer{}
// 	left := TermExpr{Field: "name", Op: FOpEq, Value: "a"}
// 	right := TermExpr{Field: "price", Op: FOpEq, Value: 2.}
// 	or := BinaryExpr{Ekind: ExprOr, Left: left, Right: right}
//
// 	start := time.Now()
// 	query := or.Compile(tracer)
//
// 	bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
// 	fmt.Println("___", time.Since(start))
// 	require.NoError(t, err)
// 	assert.True(t, bs.Count() > 0)
//
// 	fmt.Println(tracer)
// }
//
// func TestExpr_TraceAnd(t *testing.T) {
// 	indexMap := createIndexMap()
//
// 	tracer := &Tracer{}
// 	left := TermExpr{Field: "price", Op: FOpGe, Value: 1.}
// 	right := TermExpr{Field: "price", Op: FOpLe, Value: 2.}
// 	and := BinaryExpr{Ekind: ExprAnd, Left: left, Right: right}
// 	query := and.Compile(tracer)
//
// 	s := time.Now()
// 	bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
// 	require.NoError(t, err)
// 	assert.True(t, bs.Count() > 0)
//
// 	fmt.Println("**", time.Since(s))
// 	fmt.Println(tracer)
// }
//
// func TestExpr_TraceBetween(t *testing.T) {
// 	indexMap := createIndexMap()
//
// 	tracer := &Tracer{}
// 	left := TermExpr{Field: "price", Op: FOpGe, Value: 1.}
// 	right := TermExpr{Field: "price", Op: FOpLe, Value: 2.}
// 	and := BinaryExpr{Ekind: ExprAnd, Left: left, Right: right}
// 	exp := optimize(and)
// 	query := exp.Compile(tracer)
//
// 	s := time.Now()
// 	bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
// 	require.NoError(t, err)
// 	assert.True(t, bs.Count() > 0)
//
// 	fmt.Println("**", time.Since(s))
// 	fmt.Println(tracer)
// }
//
// func TestExpr_TraceNot(t *testing.T) {
// 	indexMap := createIndexMap()
//
// 	tracer := &Tracer{}
// 	// RULE: NOT (A != B)  -->  A = B
// 	child := TermExpr{Field: "name", Op: FOpNeq, Value: "a"}
// 	not := NotExpr{Child: child}
// 	exp := optimize(not)
// 	query := exp.Compile(tracer)
//
// 	bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
// 	require.NoError(t, err)
// 	assert.True(t, bs.Count() > 0)
//
// 	fmt.Println(tracer)
// }
//
// func TestExpr_TraceNotNoOptimize(t *testing.T) {
// 	indexMap := createIndexMap()
//
// 	tracer := &Tracer{}
// 	// RULE: NOT (A != B)  -->  A = B
// 	child := TermExpr{Field: "name", Op: FOpNeq, Value: "a"}
// 	not := NotExpr{Child: child}
// 	query := not.Compile(tracer)
//
// 	bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
// 	require.NoError(t, err)
// 	assert.True(t, bs.Count() > 0)
//
// 	fmt.Println(tracer)
// }
