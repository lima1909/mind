package mind

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type car struct {
	name  string
	color string
	age   uint8
	isNew bool
}

func (c *car) Name() string { return c.name }
func (c *car) Age() uint8   { return c.age }
func (c *car) IsNew() bool  { return c.isNew }

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

	qr := il.Query(Eq("name", "Opel"))
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// with cast uint8
	qr = il.Query(Eq("age", uint8(5)))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// without cast
	qr = il.Query(Eq("age", 5))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	qr = il.Query(Eq("isnew", false))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	qr = il.Query(Eq("isnew", true))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// wrong field name, expected: age, got wrong
	qr = il.Query(Eq("wrong", 5))
	_, err = qr.Count()
	assert.Error(t, err)
}

func TestList_CreateIndex_Err(t *testing.T) {
	il := NewList[car]()

	// empty field name
	err := il.CreateIndex("", NewMapIndex((*car).Age))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// ID is a reserved index name
	err = il.CreateIndex("id", NewMapIndex((*car).Age))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID is a reserved")

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
	assert.Nil(t, il.indexMap.idIndex)
	il.Insert(car{name: "Opel", age: 22})

	err := il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(il.indexMap.index))

	// check the filter/index
	qr := il.Query(Eq("age", uint8(22)))
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// not_found doesn't exist, nothing happend
	il.RemoveIndex("not_found")
	assert.Equal(t, 1, len(il.indexMap.index))

	il.RemoveIndex("age")
	assert.Equal(t, 0, len(il.indexMap.index))
	qr = il.Query(Eq("age", uint8(22)))
	_, err = qr.Values()
	assert.ErrorIs(t, InvalidNameError{"age"}, err)
	// the index is removed, but not the data
	assert.Equal(t, 1, il.Count())
}

func TestList_RemoveIndexWithId(t *testing.T) {
	il := NewListWithID((*car).Name)
	assert.NotNil(t, il.indexMap.idIndex)
	il.Insert(car{name: "Opel", age: 22})
	assert.Equal(t, 1, il.Count())

	opel, err := il.Get("Opel")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Opel", age: 22}, opel)

	il.RemoveIndex("id")
	assert.Nil(t, il.indexMap.idIndex)
	_, err = il.Get("Opel")
	assert.ErrorIs(t, NoIdIndexDefinedError{}, err)
	// the index is removed, but not the data
	assert.Equal(t, 1, il.Count())
}

func TestList_Update(t *testing.T) {
	il := NewListWithID((*car).Name)

	err := il.CreateIndex("isnew", NewMapIndex((*car).IsNew))
	assert.NoError(t, err)
	err = il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	err = il.Update(car{name: "Dacia", age: 25})
	assert.NoError(t, err)
	// check the ID index
	dacia, err := il.Get("Dacia")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Dacia", age: 25}, dacia)

	// check the age index
	result, err := il.Query(Eq("age", uint8(25))).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{{name: "Dacia", age: 25}}, result)

	err = il.Update(car{name: "NotFound", age: 25})
	assert.Error(t, err)
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

	qr := il.Query(Eq("age", uint8(22)))
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	qr = il.Query(Eq("age", uint8(22)))
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

	qr := il.Query(All())
	count, err := qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 5, count)

	// remove item on index 3
	removed := il.removeByIdxNoLock(3)
	assert.True(t, removed)
	assert.Equal(t, 4, il.Count())

	// try to find item on index 3
	qr = il.Query(And(Eq("name", "Dacia"), Eq("age", uint8(22))))
	count, err = qr.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	removed = il.removeByIdxNoLock(99)
	assert.False(t, removed)

	qr = il.Query(Eq("name", "Dacia"))
	result, err := qr.Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []car{{name: "Dacia", age: 5, isNew: true}}, result)

	qr = il.Query(Eq("age", uint8(22)))
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

	_, err := il.Query(Eq("name", "Opel")).Values()
	assert.Error(t, err)
	assert.Equal(t, "could not found index for field name: name", err.Error())

	// create Index for name
	err = il.CreateIndex("name", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	result, err := il.Query(Eq("name", "Opel")).Values()
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

	result, err := il.Query(Eq("name", "Opel")).Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []car{{name: "Opel", age: 12}}, result)

	result, err = il.Query(Lt("age", uint8(13))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, []car{
		{name: "Dacia", age: 2, color: "red"},
		{name: "Opel", age: 12},
		{name: "Mercedes", age: 5, isNew: true},
	}, result)

	result, err = il.Query(Le("age", uint8(12))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, []car{
		{name: "Dacia", age: 2, color: "red"},
		{name: "Opel", age: 12},
		{name: "Mercedes", age: 5, isNew: true},
	}, result)

	result, err = il.Query(Gt("age", uint8(11))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, []car{
		{name: "Opel", age: 12},
		{name: "Dacia", age: 22},
	}, result)

	result, err = il.Query(Ge("age", uint8(12))).Values()
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

	result, err := il.Query(Eq("val", "Dacia")).Values()
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

	result, err := il.Query(Eq("val", &dacia)).Values()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, []*string{&dacia, &dacia}, result)

	// Eq = nil
	result, err = il.Query(Eq("val", (*string)(nil))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []*string{nil}, result)

	// IsNil
	result, err = il.Query(IsNil[string]("val")).Values()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, []*string{nil}, result)

	// Or(IsNil, Eq(dacia)
	result, err = il.Query(Or(IsNil[string]("val"), Eq("val", &dacia))).Values()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, []*string{&dacia, nil, &dacia}, result)
}

func TestList_WithID(t *testing.T) {
	il := NewListWithID((*car).Name)
	err := il.CreateIndex("isnew", NewMapIndex((*car).IsNew))
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
	removed, err := il.Remove("Dacia")
	assert.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, 2, il.Count())

	// check not found after remove
	_, err = il.Get("Dacia")
	assert.ErrorIs(t, err, ValueNotFoundError{"Dacia"})
	_, err = il.Remove("Dacia")
	assert.ErrorIs(t, err, ValueNotFoundError{"Dacia"})
}

func TestList_NoID_QueryIDs(t *testing.T) {
	il := NewList[car]()
	_, err := il.Query(ID("Opel")).Values()
	assert.ErrorIs(t, err, NoIdIndexDefinedError{})

}

func TestList_QueryIDs(t *testing.T) {
	il := NewListWithID((*car).Name)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	result, err := il.Query(ID("Opel")).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
	}, result)

	result, err = il.Query(Or(ID("Dacia"), ID("Opel"))).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, result)

	result, err = il.Query(Not(ID("Mercedes"))).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, result)
}

func TestList_Pagination(t *testing.T) {
	il := NewListWithID((*car).Name)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	result, pi, err := il.Query(All()).Paginate(0, 1)
	assert.NoError(t, err)

	assert.Equal(t, PageInfo{Offset: 0, Limit: 1, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Opel", age: 22}}, result)

	result, pi, _ = il.Query(All()).Paginate(1, 2)
	assert.Equal(t, PageInfo{Offset: 1, Limit: 2, Count: 2, Total: 3}, pi)

	assert.Equal(t, []car{
		{name: "Mercedes", age: 5, isNew: true},
		{name: "Dacia", age: 22},
	}, result)

	// offset = len(il)
	result, pi, _ = il.Query(All()).Paginate(2, 1)
	assert.NoError(t, err)
	assert.Equal(t, PageInfo{Offset: 2, Limit: 1, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)

	// limit > Total
	result, pi, _ = il.Query(All()).Paginate(1, 5)
	assert.Equal(t, PageInfo{Offset: 1, Limit: 5, Count: 2, Total: 3}, pi)

	assert.Equal(t, []car{
		{name: "Mercedes", age: 5, isNew: true},
		{name: "Dacia", age: 22},
	}, result)

	// offset = len(il) is on the end
	result, pi, _ = il.Query(All()).Paginate(2, 2)
	assert.Equal(t, PageInfo{Offset: 2, Limit: 2, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)

	// count = 0
	// offset > Total
	result, pi, _ = il.Query(All()).Paginate(5, 1)
	assert.Equal(t, PageInfo{Offset: 5, Limit: 1, Count: 0, Total: 3}, pi)
	assert.Equal(t, []car{}, result)

	// offset+limit > Total
	result, pi, _ = il.Query(All()).Paginate(3, 1)
	assert.Equal(t, PageInfo{Offset: 3, Limit: 1, Count: 0, Total: 3}, pi)
	assert.Equal(t, []car{}, result)
}

func TestList_QueryStr(t *testing.T) {
	il := NewListWithID((*car).Name)
	err := il.CreateIndex("name", NewSortedIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("name2", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("age", NewSortedIndex((*car).Age))
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
