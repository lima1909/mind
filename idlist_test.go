package mind

import (
	"testing"

	"github.com/lima1909/mind/errr"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
	"github.com/stretchr/testify/assert"
)

func TestIDList_RemoveIndexWithId(t *testing.T) {
	il := NewIDList((*car).Name)
	assert.NotNil(t, il.idIndex)
	il.Insert(car{name: "Opel", age: 22})
	assert.Equal(t, 1, il.Count())

	opel, err := il.Get("Opel")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Opel", age: 22}, opel)
}

func TestIDList_Replace(t *testing.T) {
	il := NewIDList((*car).Name)

	err := il.CreateIndex("isnew", index.NewMapIndex((*car).IsNew))
	assert.NoError(t, err)
	err = il.CreateIndex("age", index.NewMapIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	old, err := il.Replace(car{name: "Dacia", age: 25})
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Dacia", age: 22}, old)
	// check the ID index
	dacia, err := il.Get("Dacia")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Dacia", age: 25}, dacia)

	// check the age index
	result, err := il.Query(query.Eq("age", uint8(25))).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{{name: "Dacia", age: 25}}, result)

	old, err = il.Replace(car{name: "NotFound", age: 25})
	assert.Error(t, err)
	assert.Equal(t, car{}, old)
}

func TestIDList_Update(t *testing.T) {
	l := NewIDList((*car).Name)
	err := l.CreateIndex("age", index.NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	l.Insert(car{name: "Opel", age: 22})
	l.Insert(car{name: "Mercedes", age: 5})
	l.Insert(car{name: "Dacia", age: 22})
	l.Insert(car{name: "Audi", age: 5})

	err = l.Update("Opel", func(c *car) { c.name = "VW"; c.age = 3; c.color = "blue" })
	assert.NoError(t, err)

	err = l.Update("NotFound", func(*car) {})
	assert.Error(t, err)

	vw, err := l.Get("VW")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "VW", age: 3, color: "blue"}, vw)

	h := l.QueryStr(`id = "Opel"`)
	count, err := h.Count()
	assert.ErrorIs(t, errr.ValueNotFoundError{Value: "Opel"}, err)
	assert.Equal(t, 0, count)

	h = l.QueryStr(`id = "VW"`)
	cars, err := h.Values()
	assert.NoError(t, err)
	assert.Equal(t, car{name: "VW", age: 3, color: "blue"}, cars[0])

	h = l.QueryStr(`age = 3`)
	cars, err = h.Values()
	assert.NoError(t, err)
	assert.Equal(t, car{name: "VW", age: 3, color: "blue"}, cars[0])
}

func TestIDList_WithID(t *testing.T) {
	il := NewIDList((*car).Name)
	err := il.CreateIndex("isnew", index.NewMapIndex((*car).IsNew))
	assert.NoError(t, err)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 42})

	dacia, err := il.Get("Dacia")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Dacia", age: 42}, dacia)
	assert.Equal(t, 3, il.Count())
	assert.True(t, il.Contains("Dacia"))
	assert.False(t, il.Contains("NotFound"))

	// remove dacia
	removed, idx, err := il.Remove("Dacia")
	assert.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, 2, idx)
	assert.Equal(t, 2, il.Count())

	// check not found after remove
	_, err = il.Get("Dacia")
	assert.ErrorIs(t, err, errr.ValueNotFoundError{Value: "Dacia"})
	_, idx, err = il.Remove("Dacia")
	assert.ErrorIs(t, err, errr.ValueNotFoundError{Value: "Dacia"})
	assert.Equal(t, -1, idx)
}

func TestIDList_NoID_QueryIDs(t *testing.T) {
	il := NewIDList((*car).Name)
	_, err := il.Query(query.ID("Opel")).Values()
	assert.ErrorIs(t, err, errr.ValueNotFoundError{Value: "Opel"})
}

func TestList_QueryIDs(t *testing.T) {
	il := NewIDList((*car).Name)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	result, err := il.Query(query.ID("Opel")).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
	}, result)

	result, err = il.Query(query.Or(query.ID("Dacia"), query.ID("Opel"))).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, result)

	result, err = il.Query(query.Not(query.ID("Mercedes"))).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, result)
}

func TestList_Pagination(t *testing.T) {
	il := NewIDList((*car).Name)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	result, pi, err := il.Query(query.All()).Paginate(0, 1)
	assert.NoError(t, err)
	assert.Equal(t, query.PageInfo{Offset: 0, Limit: 1, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Opel", age: 22}}, result)

	result, pi, _ = il.Query(query.All()).Paginate(1, 2)
	assert.Equal(t, query.PageInfo{Offset: 1, Limit: 2, Count: 2, Total: 3}, pi)
	assert.Equal(t, []car{
		{name: "Mercedes", age: 5, isNew: true},
		{name: "Dacia", age: 22},
	}, result)

	// offset = len(il), get the last one
	result, pi, _ = il.Query(query.All()).Paginate(2, 1)
	assert.NoError(t, err)
	assert.Equal(t, query.PageInfo{Offset: 2, Limit: 1, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)

	// offset = len(il) is on the end
	result, pi, _ = il.Query(query.All()).Paginate(2, 2)
	assert.Equal(t, query.PageInfo{Offset: 2, Limit: 2, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)

	// limit > Total
	result, pi, _ = il.Query(query.All()).Paginate(1, 5)
	assert.Equal(t, query.PageInfo{Offset: 1, Limit: 5, Count: 2, Total: 3}, pi)
	assert.Equal(t, []car{
		{name: "Mercedes", age: 5, isNew: true},
		{name: "Dacia", age: 22},
	}, result)

	// count = 0
	// offset > Total
	result, pi, _ = il.Query(query.All()).Paginate(5, 1)
	assert.Equal(t, query.PageInfo{Offset: 5, Limit: 1, Count: 0, Total: 3}, pi)
	assert.Equal(t, []car{}, result)

	// offset+limit > Total
	result, pi, _ = il.Query(query.All()).Paginate(3, 1)
	assert.Equal(t, query.PageInfo{Offset: 3, Limit: 1, Count: 0, Total: 3}, pi)
	assert.Equal(t, []car{}, result)
}

func TestList_QueryStr(t *testing.T) {
	il := NewIDList((*car).Name)
	err := il.CreateIndex("name", index.NewSortedIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("name2", index.NewMapIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("age", index.NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5})
	il.Insert(car{name: "Dacia", age: 22})
	il.Insert(car{name: "Opel", age: 5})

	result, err := il.QueryStr(`name = "Opel"`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Opel", age: 5},
	}, result)

	result, err = il.QueryStr(`age = 22`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, result)

	result, err = il.QueryStr(`name = "Opel" or name = "Dacia"`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Opel", age: 5},
	}, result)

	result, err = il.QueryStr(`name = "Opel" or name = "Dacia" or age > 20`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Opel", age: 5},
	}, result)

	result, err = il.QueryStr(`name IN("Opel", "Dacia") or age > 20`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Opel", age: 5},
	}, result)

	// same test for MapIndex
	result, err = il.QueryStr(`name2 IN("Opel", "Dacia") or age > 20`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Opel", age: 5},
	}, result)
}
