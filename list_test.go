package mind

import (
	"slices"
	"strings"
	"testing"

	"github.com/lima1909/mind/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList_Base(t *testing.T) {
	il := NewList[car]()

	err := il.CreateIndex("name", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("isnew", NewMapIndex((*car).IsNew))
	assert.NoError(t, err)

	il.Insert(car{name: "Dacia", age: 22, color: "red"})
	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})
	assert.Equal(t, 4, il.Count())

	err = il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)

	c, found := il.list.Get(1)
	assert.True(t, found)
	assert.Equal(t, car{name: "Opel", age: 22}, c)

	_, found = il.list.Get(99)
	assert.False(t, found)

	qr := il.Query(query.Eq("name", "Opel"))
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// with cast uint8
	qr = il.Query(query.Eq("age", uint8(5)))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// without cast
	qr = il.Query(query.Eq("age", 5))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	qr = il.Query(query.Eq("isnew", false))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	qr = il.Query(query.Eq("isnew", true))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// wrong field name, expected: age, got wrong
	qr = il.Query(query.Eq("wrong", 5))
	_, err = qr.Count()
	assert.Error(t, err)
}

func TestList_CreateIndex_Err(t *testing.T) {
	il := NewList[car]()

	// empty field name
	err := il.CreateIndex("", NewMapIndex((*car).Age))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// field name already exist
	err = il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)
	err = il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "age already exists")
}

func TestList_RemoveIndex(t *testing.T) {
	il := NewList[car]()
	assert.Equal(t, 0, len(il.indexMap.index))
	il.Insert(car{name: "Opel", age: 22})

	err := il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(il.indexMap.index))

	// check the filter/index
	qr := il.Query(query.Eq("age", uint8(22)))
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// not_found doesn't exist, nothing happend
	il.RemoveIndex("not_found")
	assert.Equal(t, 1, len(il.indexMap.index))

	il.RemoveIndex("age")
	assert.Equal(t, 0, len(il.indexMap.index))
	qr = il.Query(query.Eq("age", uint8(22)))
	_, err = qr.Values()
	assert.ErrorIs(t, InvalidNameError{"age"}, err)
	// the index is removed, but not the data
	assert.Equal(t, 1, il.Count())
}

func TestList_QueryResult(t *testing.T) {
	il := NewList[car]()
	err := il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Mercedes", age: 22, color: "red"})
	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Dacia", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})
	il.Insert(car{name: "Audi", age: 22})

	qr := il.Query(query.Eq("age", uint8(22)))
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	qr = il.Query(query.Eq("age", uint8(22)))
	result, err := qr.Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Mercedes", age: 22, color: "red"},
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Audi", age: 22},
	},
		result,
	)

	slices.SortFunc(result, func(c1, c2 car) int {
		return strings.Compare(c1.name, c2.name)
	})

	assert.Equal(t, []car{
		{name: "Audi", age: 22},
		{name: "Dacia", age: 22},
		{name: "Mercedes", age: 22, color: "red"},
		{name: "Opel", age: 22},
	},
		result,
	)
}

func removeByIdxNoLock(l *List[car], index int) (removed bool) {
	item, found := l.list.Get(index)
	if !found {
		return found
	}

	removed = l.list.Remove(index)
	l.indexMap.delete(&item, index)

	return removed
}

func TestList_Remove(t *testing.T) {
	il := NewList[car]()
	err := il.CreateIndex("name", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Mercedes", age: 22, color: "red"})
	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Dacia", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})
	il.Insert(car{name: "Audi", age: 22})
	assert.Equal(t, 5, il.Count())

	qr := il.Query(query.All())
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 5, count)

	// remove item on index 3
	removed := removeByIdxNoLock(il, 3)
	assert.True(t, removed)
	assert.Equal(t, 4, il.Count())

	// try to find item on index 3
	qr = il.Query(query.And(query.Eq("name", "Dacia"), query.Eq("age", uint8(22))))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	removed = removeByIdxNoLock(il, 99)
	assert.False(t, removed)

	qr = il.Query(query.Eq("name", "Dacia"))
	result, err := qr.Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []car{{name: "Dacia", age: 5, isNew: true}}, result)

	qr = il.Query(query.Eq("age", uint8(22)))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestList_CreateIndex(t *testing.T) {
	il := NewList[car]()
	il.Insert(car{name: "Dacia", age: 22, color: "red"})
	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	_, err := il.Query(query.Eq("name", "Opel")).Values()
	assert.Error(t, err)
	assert.Equal(t, "could not found index for field name: name", err.Error())

	// create Index for name
	err = il.CreateIndex("name", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	result, err := il.Query(query.Eq("name", "Opel")).Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []car{{name: "Opel", age: 22}}, result)
}

func TestList_CreateIndexVarious(t *testing.T) {
	il := NewList[car]()
	err := il.CreateIndex("name", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("age", NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Dacia", age: 2, color: "red"})
	il.Insert(car{name: "Opel", age: 12})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	result, err := il.Query(query.Eq("name", "Opel")).Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []car{{name: "Opel", age: 12}}, result)

	result, err = il.Query(query.Lt("age", uint8(13))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, []car{
		{name: "Dacia", age: 2, color: "red"},
		{name: "Opel", age: 12},
		{name: "Mercedes", age: 5, isNew: true},
	}, result)

	result, err = il.Query(query.Le("age", uint8(12))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, []car{
		{name: "Dacia", age: 2, color: "red"},
		{name: "Opel", age: 12},
		{name: "Mercedes", age: 5, isNew: true},
	}, result)

	result, err = il.Query(query.Gt("age", uint8(11))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, []car{
		{name: "Opel", age: 12},
		{name: "Dacia", age: 22},
	}, result)

	result, err = il.Query(query.Ge("age", uint8(12))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, []car{
		{name: "Opel", age: 12},
		{name: "Dacia", age: 22},
	}, result)
}

func TestList_StringItem(t *testing.T) {
	il := NewList[string]()
	err := il.CreateIndex("val", NewMapIndex(FromValue[string]()))
	assert.NoError(t, err)

	il.Insert("Dacia")
	il.Insert("Opel")
	il.Insert("Mercedes")
	il.Insert("Dacia")

	result, err := il.Query(query.Eq("val", "Dacia")).Values()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, []string{"Dacia", "Dacia"}, result)
}

func TestList_StringPtrItemWithNil(t *testing.T) {
	il := NewList[*string]()
	err := il.CreateIndex("val", NewMapIndex(FromValue[*string]()))
	assert.NoError(t, err)

	dacia := "Dacia"
	il.Insert(&dacia)
	il.Insert(nil)
	il.Insert(&dacia)

	result, err := il.Query(query.Eq("val", &dacia)).Values()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, []*string{&dacia, &dacia}, result)

	// Eq = nil
	result, err = il.Query(query.Eq("val", (*string)(nil))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []*string{nil}, result)

	// // IsNil
	// result, err = il.Query(IsNil[string]("val")).Values()
	// assert.NoError(t, err)
	// assert.Equal(t, 1, len(result))
	// assert.Equal(t, []*string{nil}, result)
	//
	// // Or(IsNil, Eq(dacia)
	// result, err = il.Query(Or(IsNil[string]("val"), Eq("val", &dacia))).Values()
	// assert.NoError(t, err)
	// assert.Equal(t, 3, len(result))
	// assert.Equal(t, []*string{&dacia, nil, &dacia}, result)
}

func TestList_EscapedString(t *testing.T) {
	il := NewList[car]()

	err := il.CreateIndex("name", NewStringIndex((*car).Name).AddTrigramIndex())
	require.NoError(t, err)

	il.Insert(car{name: "Opel 1", age: 22})
	il.Insert(car{name: "Opel 2", age: 5})
	il.Insert(car{name: "Dacia\\'s", age: 22})
	il.Insert(car{name: "\"Dacia\"", age: 22})

	result, err := il.QueryStr(`name like "%pel%"`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel 1", age: 22},
		{name: "Opel 2", age: 5},
	}, result)

	result, err = il.QueryStr(`name like "Op%"`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel 1", age: 22},
		{name: "Opel 2", age: 5},
	}, result)

	result, err = il.QueryStr(`name = "Dacia\\'s"`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{{name: "Dacia\\'s", age: 22}}, result)

	result, err = il.QueryStr(`name = "\"Dacia\""`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{{name: "\"Dacia\"", age: 22}}, result)
}

func TestList_GetRemove(t *testing.T) {
	l := NewList[car]()
	err := l.CreateIndex("name", NewSortedIndex((*car).Name))
	assert.NoError(t, err)

	l.Insert(car{name: "Opel", age: 22})
	l.Insert(car{name: "Mercedes", age: 5})
	l.Insert(car{name: "Dacia", age: 22})
	l.Insert(car{name: "Opel", age: 5})

	dacia, found := l.Get(2)
	assert.True(t, found)
	assert.Equal(t, car{name: "Dacia", age: 22}, dacia)

	removed := l.Remove(2)
	assert.True(t, removed)
	removed = l.Remove(2)
	assert.False(t, removed)

	dacia, found = l.Get(2)
	assert.False(t, found)
	assert.Equal(t, car{}, dacia)

	h := l.QueryStr(`name = "Dacia"`)
	count, err := h.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// iterate over all cars
	type icar struct {
		i int
		c car
	}
	cars := make([]icar, 0)
	l.Values(func(i int, c car) bool {
		cars = append(cars, icar{i, c})
		return true
	})
	assert.Equal(t, []icar{
		{0, car{name: "Opel", age: 22}},
		{1, car{name: "Mercedes", age: 5}},
		{3, car{name: "Opel", age: 5}},
	}, cars)
}

func TestList_Update(t *testing.T) {
	l := NewList[car]()
	err := l.CreateIndex("name", NewSortedIndex((*car).Name))
	assert.NoError(t, err)

	l.Insert(car{name: "Opel", age: 22})
	l.Insert(car{name: "Mercedes", age: 5})
	l.Insert(car{name: "Dacia", age: 22})
	l.Insert(car{name: "Opel", age: 5})

	updated := l.Update(0, func(c *car) { c.name = "VW"; c.age = 3; c.color = "blue" })
	assert.True(t, updated)

	updated = l.Update(100, func(c *car) { c.name = "VW" })
	assert.False(t, updated)

	vw, found := l.Get(0)
	assert.True(t, found)
	assert.Equal(t, car{name: "VW", age: 3, color: "blue"}, vw)

	h := l.QueryStr(`name = "Opel"`)
	count, err := h.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	h = l.QueryStr(`name = "VW"`)
	cars, err := h.Values()
	assert.NoError(t, err)
	assert.Equal(t, car{name: "VW", age: 3, color: "blue"}, cars[0])
}
