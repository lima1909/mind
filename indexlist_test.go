package main

import (
	"strings"
	"sync"
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

func TestIndexList_Base(t *testing.T) {
	il := NewIndexList[car]()

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

	qr, err := il.Query(Eq("name", "Opel"))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())

	// with cast uint8
	qr, err = il.Query(Eq("age", uint8(5)))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())

	// without cast
	qr, err = il.Query(Eq("age", 5))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())

	qr, err = il.Query(Eq("isnew", false))
	assert.NoError(t, err)
	assert.Equal(t, 3, qr.Count())

	qr, err = il.Query(Eq("isnew", true))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())

	// wrong field name, expected: age, got wrong
	qr, err = il.Query(Eq("wrong", 5))
	assert.Error(t, err)
	assert.Equal(t, QueryResult[car, struct{}]{}, qr)
}

func TestIndexList_CreateIndex_Err(t *testing.T) {
	il := NewIndexList[car]()

	// empty field name
	err := il.CreateIndex("", NewMapIndex((*car).Age))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

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

func TestIndexList_RemoveIndex(t *testing.T) {
	il := NewIndexList[car]()
	assert.Equal(t, 0, len(il.indexMap.index))
	assert.Nil(t, il.indexMap.idIndex)
	il.Insert(car{name: "Opel", age: 22})

	err := il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(il.indexMap.index))

	// check the filter/index
	qr, err := il.Query(Eq("age", uint8(22)))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())

	// not_found doesn't exist, nothing happend
	il.RemoveIndex("not_found")
	assert.Equal(t, 1, len(il.indexMap.index))

	il.RemoveIndex("age")
	assert.Equal(t, 0, len(il.indexMap.index))
	_, err = il.Query(Eq("age", uint8(22)))
	assert.ErrorIs(t, ErrInvalidIndexdName{"age"}, err)
	// the index is removed, but not the data
	assert.Equal(t, 1, il.Count())
}

func TestIndexList_RemoveIndexWithId(t *testing.T) {
	il := NewIndexListWithID((*car).Name)
	assert.NotNil(t, il.indexMap.idIndex)
	il.Insert(car{name: "Opel", age: 22})
	assert.Equal(t, 1, il.Count())

	opel, err := il.Get("Opel")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Opel", age: 22}, opel)

	il.RemoveIndex("id")
	assert.Nil(t, il.indexMap.idIndex)
	_, err = il.Get("Opel")
	assert.ErrorIs(t, ErrNoIdIndexDefined{}, err)
	// the index is removed, but not the data
	assert.Equal(t, 1, il.Count())
}

func TestIndexList_Update(t *testing.T) {
	il := NewIndexListWithID((*car).Name)

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
	result, err := il.Query(Eq("age", uint8(25)))
	assert.NoError(t, err)
	assert.Equal(t, []car{{name: "Dacia", age: 25}}, result.Values())

	err = il.Update(car{name: "NotFound", age: 25})
	assert.Error(t, err)
}

func TestIndexList_QueryResult(t *testing.T) {
	less := func(c1, c2 *car) bool { return strings.Compare(c1.name, c2.name) < 0 }

	il := NewIndexList[car]()
	err := il.CreateIndex("age", NewMapIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Mercedes", age: 22, color: "red"})
	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Dacia", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})
	il.Insert(car{name: "Audi", age: 22})

	qr, err := il.Query(Eq("age", uint8(22)))
	assert.NoError(t, err)

	assert.False(t, qr.IsEmpty())
	assert.Equal(t, 4, qr.Count())

	assert.Equal(t, []car{
		{name: "Mercedes", age: 22, color: "red"},
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Audi", age: 22},
	},
		qr.Values(),
	)

	assert.Equal(t, []car{
		{name: "Audi", age: 22},
		{name: "Dacia", age: 22},
		{name: "Mercedes", age: 22, color: "red"},
		{name: "Opel", age: 22},
	},
		qr.Sort(less),
	)
}

func TestIndexList_Remove(t *testing.T) {
	il := NewIndexList[car]()
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

	qr, err := il.Query(All())
	assert.NoError(t, err)

	assert.False(t, qr.IsEmpty())
	assert.Equal(t, 5, qr.Count())

	// remove item on index 3
	c, removed := il.removeNoLock(3)
	assert.True(t, removed)
	assert.Equal(t, 4, il.Count())
	assert.Equal(t, c, car{name: "Dacia", age: 22})

	// try to find item on index 3
	qr, err = il.Query(And(Eq("name", "Dacia"), Eq("age", uint8(22))))
	assert.NoError(t, err)
	assert.Equal(t, 0, qr.Count())

	_, removed = il.removeNoLock(99)
	assert.False(t, removed)

	qr, err = il.Query(Eq("name", "Dacia"))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())
	assert.Equal(t, []car{{name: "Dacia", age: 5, isNew: true}}, qr.Values())

	qr, err = il.Query(Eq("age", uint8(22)))
	assert.NoError(t, err)
	assert.Equal(t, 3, qr.Count())

	qr.RemoveAll()
	assert.Equal(t, 1, il.Count())

	c, found := il.list.Get(2)
	assert.True(t, found)
	assert.Equal(t, car{name: "Dacia", age: 5, isNew: true}, c)
}

func TestIndexList_RemoveLater(t *testing.T) {
	il := NewIndexList[car]()
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

	qr1, err := il.Query(Eq("name", "Dacia"))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr1.Count())

	qr2, err := il.Query(Eq("name", "Dacia"))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr2.Count())

	qr1.RemoveAll()
	assert.Equal(t, 0, qr1.Count())
	assert.Equal(t, 2, qr2.Count())
	assert.Equal(t, 3, il.Count())

	qr, err := il.Query(Eq("name", "Dacia"))
	// Dacia doesn't exist anymore
	assert.NoError(t, err)
	assert.True(t, qr.IsEmpty())

	// qr1 has allready remove all Dacia
	qr2.RemoveAll()
	assert.Equal(t, 3, il.Count())
}

func TestIndexList_RemoveLaterAsync(t *testing.T) {
	il := NewIndexList[car]()
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

	qr1, err := il.Query(Eq("name", "Dacia"))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr1.Count())

	qr2, err := il.Query(Eq("name", "Dacia"))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr2.Count())

	var wg sync.WaitGroup

	wg.Go(func() {
		qr1.RemoveAll()
		assert.Equal(t, 0, qr1.Count())
	})

	wg.Go(func() {
		qr2.RemoveAll()
		assert.Equal(t, 0, qr2.Count())
	})

	wg.Wait()
	assert.Equal(t, 3, il.Count())
}

func TestIndexList_CreateIndex(t *testing.T) {
	il := NewIndexList[car]()
	il.Insert(car{name: "Dacia", age: 22, color: "red"})
	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	_, err := il.Query(Eq("name", "Opel"))
	assert.Error(t, err)
	assert.Equal(t, "could not found index for field name: name", err.Error())

	// create Index for name
	err = il.CreateIndex("name", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	qr, err := il.Query(Eq("name", "Opel"))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())
	assert.Equal(t, []car{{name: "Opel", age: 22}}, qr.Values())
}

func TestIndexList_CreateIndexVarious(t *testing.T) {
	il := NewIndexList[car]()
	err := il.CreateIndex("name", NewMapIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("age", NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Dacia", age: 2, color: "red"})
	il.Insert(car{name: "Opel", age: 12})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	qr, err := il.Query(Eq("name", "Opel"))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())
	assert.Equal(t, []car{{name: "Opel", age: 12}}, qr.Values())

	qr, err = il.Query(Lt("age", uint8(13)))
	assert.NoError(t, err)
	assert.Equal(t, 3, qr.Count())
	assert.Equal(t, []car{
		{name: "Dacia", age: 2, color: "red"},
		{name: "Opel", age: 12},
		{name: "Mercedes", age: 5, isNew: true},
	}, qr.Values())

	qr, err = il.Query(Le("age", uint8(12)))
	assert.NoError(t, err)
	assert.Equal(t, 3, qr.Count())
	assert.Equal(t, []car{
		{name: "Dacia", age: 2, color: "red"},
		{name: "Opel", age: 12},
		{name: "Mercedes", age: 5, isNew: true},
	}, qr.Values())

	qr, err = il.Query(Gt("age", uint8(11)))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr.Count())
	assert.Equal(t, []car{
		{name: "Opel", age: 12},
		{name: "Dacia", age: 22},
	}, qr.Values())

	qr, err = il.Query(Ge("age", uint8(12)))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr.Count())
	assert.Equal(t, []car{
		{name: "Opel", age: 12},
		{name: "Dacia", age: 22},
	}, qr.Values())
}

func TestIndexList_StringItem(t *testing.T) {
	il := NewIndexList[string]()
	err := il.CreateIndex("val", NewMapIndex(FromValue[string]()))
	assert.NoError(t, err)

	il.Insert("Dacia")
	il.Insert("Opel")
	il.Insert("Mercedes")
	il.Insert("Dacia")

	qr, err := il.Query(Eq("val", "Dacia"))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr.Count())
	assert.Equal(t, []string{"Dacia", "Dacia"}, qr.Values())
}

func TestIndexList_StringPtrItemWithNil(t *testing.T) {
	il := NewIndexList[*string]()
	err := il.CreateIndex("val", NewMapIndex(FromValue[*string]()))
	assert.NoError(t, err)

	dacia := "Dacia"
	il.Insert(&dacia)
	il.Insert(nil)
	il.Insert(&dacia)

	qr, err := il.Query(Eq("val", &dacia))
	assert.NoError(t, err)
	assert.Equal(t, 2, qr.Count())
	assert.Equal(t, []*string{&dacia, &dacia}, qr.Values())

	// Eq = nil
	qr, err = il.Query(Eq("val", (*string)(nil)))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())
	assert.Equal(t, []*string{nil}, qr.Values())

	// IsNil
	qr, err = il.Query(IsNil[string]("val"))
	assert.NoError(t, err)
	assert.Equal(t, 1, qr.Count())
	assert.Equal(t, []*string{nil}, qr.Values())

	// Or(IsNil, Eq(dacia)
	qr, err = il.Query(Or(IsNil[string]("val"), Eq("val", &dacia)))
	assert.NoError(t, err)
	assert.Equal(t, 3, qr.Count())
	assert.Equal(t, []*string{&dacia, nil, &dacia}, qr.Values())
}

func TestIndexList_WithID(t *testing.T) {
	il := NewIndexListWithID((*car).Name)
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
	assert.ErrorIs(t, err, ErrValueNotFound{"Dacia"})
	_, err = il.Remove("Dacia")
	assert.ErrorIs(t, err, ErrValueNotFound{"Dacia"})
}

func TestIndexList_NoID_QueryIDs(t *testing.T) {
	il := NewIndexList[car]()
	_, err := il.Query(ID("Opel"))
	assert.ErrorIs(t, err, ErrNoIdIndexDefined{})

}

func TestIndexList_QueryIDs(t *testing.T) {
	il := NewIndexListWithID((*car).Name)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	qr, err := il.Query(ID("Opel"))
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
	}, qr.Values())

	qr, err = il.Query(Or(ID("Dacia"), ID("Opel")))
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, qr.Values())

	qr, err = il.Query(Not(ID("Mercedes")))
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, qr.Values())
}

func TestIndexList_Pagination(t *testing.T) {
	il := NewIndexListWithID((*car).Name)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5, isNew: true})
	il.Insert(car{name: "Dacia", age: 22})

	qr, err := il.Query(All())
	assert.NoError(t, err)

	result, pi := qr.Pagination(0, 1)
	assert.Equal(t, PageInfo{Offset: 0, Limit: 1, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Opel", age: 22}}, result)

	result, pi = qr.Pagination(1, 2)
	assert.Equal(t, PageInfo{Offset: 1, Limit: 2, Count: 2, Total: 3}, pi)
	assert.Equal(t, []car{
		{name: "Mercedes", age: 5, isNew: true},
		{name: "Dacia", age: 22},
	}, result)

	// offset = len(il)
	result, pi = qr.Pagination(2, 1)
	assert.NoError(t, err)
	assert.Equal(t, PageInfo{Offset: 2, Limit: 1, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)

	// limit > Total
	result, pi = qr.Pagination(1, 5)
	assert.Equal(t, PageInfo{Offset: 1, Limit: 5, Count: 2, Total: 3}, pi)
	assert.Equal(t, []car{
		{name: "Mercedes", age: 5, isNew: true},
		{name: "Dacia", age: 22},
	}, result)
	// offset = len(il) is on the end
	result, pi = qr.Pagination(2, 2)
	assert.Equal(t, PageInfo{Offset: 2, Limit: 2, Count: 1, Total: 3}, pi)
	assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)

	// count = 0
	// offset > Total
	result, pi = qr.Pagination(5, 1)
	assert.Equal(t, PageInfo{Offset: 5, Limit: 1, Count: 0, Total: 3}, pi)
	assert.Equal(t, []car{}, result)

	// offset+limit > Total
	result, pi = qr.Pagination(3, 1)
	assert.Equal(t, PageInfo{Offset: 3, Limit: 1, Count: 0, Total: 3}, pi)
	assert.Equal(t, []car{}, result)
}

func TestIndexList_QueryStr(t *testing.T) {
	il := NewIndexListWithID((*car).Name)
	err := il.CreateIndex("name", NewSortedIndex((*car).Name))
	assert.NoError(t, err)
	err = il.CreateIndex("age", NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	il.Insert(car{name: "Opel", age: 22})
	il.Insert(car{name: "Mercedes", age: 5})
	il.Insert(car{name: "Dacia", age: 22})
	il.Insert(car{name: "Opel", age: 5})

	qr, err := il.QueryStr(`name = "Opel"`)
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Opel", age: 5},
	}, qr.Values())

	qr, err = il.QueryStr(`age = 22`)
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
	}, qr.Values())

	qr, err = il.QueryStr(`name = "Opel" or name = "Dacia"`)
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Opel", age: 5},
	}, qr.Values())

	qr, err = il.QueryStr(`name = "Opel" or name = "Dacia" or age > 20`)
	assert.NoError(t, err)
	assert.Equal(t, []car{
		{name: "Opel", age: 22},
		{name: "Dacia", age: 22},
		{name: "Opel", age: 5},
	}, qr.Values())
}
