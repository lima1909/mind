package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lima1909/mind"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

type User struct {
	id    int64
	name  string
	role  string
	ok    bool
	price float64
}

func (u *User) Name() string   { return u.name }
func (u *User) Role() string   { return u.role }
func (u *User) Ok() bool       { return u.ok }
func (u *User) Price() float64 { return u.price }
func (u *User) ID() int64      { return u.id }

var allIDs = lidx.NewRawIDsFrom[uint32](0, 1, 2)

func createIndexMap() mind.IndexMap[User] {
	indexMap := mind.NewIndexMap[User]()

	_ = indexMap.AddIndex("name", index.NewSortedIndex((*User).Name))
	_ = indexMap.AddIndex("price", index.NewSortedIndex((*User).Price))

	indexMap.Insert(&User{name: "a", price: 1}, 0)
	indexMap.Insert(&User{name: "b", price: 2}, 1)
	indexMap.Insert(&User{name: "c", price: 0}, 2)

	return indexMap
}

func TestExpr_Trace(t *testing.T) {
	indexMap := createIndexMap()

	tracer := &query.Tracer{}
	nameEq := query.TermExpr{Field: "name", Op: query.FOpEq, Value: "a"}
	query := nameEq.Compile(tracer)

	bs, _, err := query(indexMap.FilterByName, allIDs)
	require.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	assert.True(t, tracer.Duration > 0)
	assert.Nil(t, tracer.Children)
	assert.Equal(t, 1, tracer.Matches)
}

func TestExpr_TraceAnd(t *testing.T) {
	indexMap := createIndexMap()

	tracer := &query.Tracer{}
	left := query.TermExpr{Field: "name", Op: query.FOpEq, Value: "b"}
	right := query.TermExpr{Field: "price", Op: query.FOpEq, Value: 2.}
	and := query.AndExpr{Left: left, Right: right}
	query := and.Compile(tracer)

	bs, _, err := query(indexMap.FilterByName, allIDs)
	require.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	assert.True(t, tracer.Duration > 0)
	assert.Equal(t, 2, len(tracer.Children))
	assert.Equal(t, 1, tracer.Matches)
}

func TestExpr_TraceOr(t *testing.T) {
	indexMap := createIndexMap()

	tracer := &query.Tracer{}
	left := query.TermExpr{Field: "name", Op: query.FOpEq, Value: "a"}
	right := query.TermExpr{Field: "price", Op: query.FOpEq, Value: 2.}
	or := query.OrExpr{Left: left, Right: right}

	query := or.Compile(tracer)

	bs, _, err := query(indexMap.FilterByName, allIDs)
	require.NoError(t, err)
	assert.Equal(t, 2, bs.Count())
	assert.True(t, tracer.Duration > 0)
	assert.Equal(t, 2, len(tracer.Children))
	assert.Equal(t, 2, tracer.Matches)
}

func TestExpr_TraceBetween(t *testing.T) {
	indexMap := createIndexMap()

	tracer := &query.Tracer{}
	left := query.TermExpr{Field: "price", Op: query.FOpGe, Value: 1.}
	right := query.TermExpr{Field: "price", Op: query.FOpLe, Value: 2.}
	and := query.AndExpr{Left: left, Right: right}
	exp := and.Optimize()
	query := exp.Compile(tracer)

	bs, _, err := query(indexMap.FilterByName, allIDs)
	require.NoError(t, err)
	assert.Equal(t, 2, bs.Count())
	assert.True(t, tracer.Duration > 0)
	assert.Equal(t, 0, len(tracer.Children))
	assert.Equal(t, 2, tracer.Matches)
}

func TestExpr_TraceNot(t *testing.T) {
	indexMap := createIndexMap()

	tracer := &query.Tracer{}
	// RULE: NOT (A != B)  -->  A = B
	child := query.TermExpr{Field: "name", Op: query.FOpNeq, Value: "a"}
	not := query.NotExpr{Child: child}
	exp := not.Optimize()
	query := exp.Compile(tracer)

	bs, _, err := query(indexMap.FilterByName, allIDs)
	require.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	assert.True(t, tracer.Duration > 0)
	assert.Equal(t, 0, len(tracer.Children))
	assert.Equal(t, 1, tracer.Matches)
}

func TestExpr_TraceNotNoOptimize(t *testing.T) {
	indexMap := createIndexMap()

	tracer := &query.Tracer{}
	// RULE: NOT (A != B)  -->  A = B
	child := query.TermExpr{Field: "name", Op: query.FOpNeq, Value: "a"}
	not := query.NotExpr{Child: child}
	query := not.Compile(tracer)

	bs, _, err := query(indexMap.FilterByName, allIDs)
	require.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	assert.True(t, tracer.Duration > 0)
	assert.Equal(t, 1, len(tracer.Children))
	assert.Equal(t, 1, tracer.Matches)
}
