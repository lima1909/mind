package mind

import (
	"fmt"
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
	assert.Equal(t, 1, il.Len())

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

	// ID (Name) changed from "Opel" to "VW", what is not allowed
	err = l.Update("Opel", func(c *car) { c.name = "VW"; c.age = 3; c.color = "blue" })
	assert.Error(t, err)

	err = l.Update("NotFound", func(*car) {})
	assert.Error(t, err)

	// second try to update
	err = l.Update("Opel", func(c *car) { c.name = "Opel"; c.age = 3; c.color = "blue" })
	assert.NoError(t, err)

	opel, err := l.Get("Opel")
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Opel", age: 3, color: "blue"}, opel)

	cars, err := l.QueryStr(`age = 3`).Values()
	assert.NoError(t, err)
	assert.Equal(t, []car{{name: "Opel", age: 3, color: "blue"}}, cars)

	h := l.QueryStr(`id = "Opel"`)
	count, err := h.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	h = l.QueryStr(`id = "VW"`)
	cars, err = h.Values()
	assert.ErrorIs(t, errr.ValueNotFoundError{Value: "VW"}, err)
	assert.Equal(t, 0, len(cars))

	h = l.QueryStr(`age = 3`)
	cars, err = h.Values()
	assert.NoError(t, err)
	assert.Equal(t, car{name: "Opel", age: 3, color: "blue"}, cars[0])
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
	assert.Equal(t, 3, il.Len())
	assert.True(t, il.Contains("Dacia"))
	assert.False(t, il.Contains("NotFound"))

	// remove dacia
	removed, idx, err := il.Remove("Dacia")
	assert.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, 2, idx)
	assert.Equal(t, 2, il.Len())

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

	qh := il.Query(query.All())

	t.Run("Paginate: 0 to 1", func(t *testing.T) {
		result := make([]car, 0, 1)
		pi, err := qh.Paginate(0, &result)
		assert.NoError(t, err)
		assert.Equal(t, query.PageInfo{Offset: 0, Limit: 1, Count: 1, Total: 3}, pi)
		assert.Equal(t, []car{{name: "Opel", age: 22}}, result)
	})

	t.Run("Paginate: 1 to 2", func(t *testing.T) {
		result := make([]car, 0, 2)
		pi, _ := qh.Paginate(1, &result)
		assert.Equal(t, query.PageInfo{Offset: 1, Limit: 2, Count: 2, Total: 3}, pi)
		assert.Equal(t, []car{
			{name: "Mercedes", age: 5, isNew: true},
			{name: "Dacia", age: 22},
		}, result)
	})

	t.Run("Paginate: offset = len(il), get the last one, 2 to 1", func(t *testing.T) {
		result := make([]car, 0, 1)
		pi, err := qh.Paginate(2, &result)
		assert.NoError(t, err)
		assert.Equal(t, query.PageInfo{Offset: 2, Limit: 1, Count: 1, Total: 3}, pi)
		assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)
	})

	t.Run("Paginate: offset = len(il) is on the end, 2 to 2", func(t *testing.T) {
		result := make([]car, 0, 2)
		pi, _ := qh.Paginate(2, &result)
		assert.Equal(t, query.PageInfo{Offset: 2, Limit: 2, Count: 1, Total: 3}, pi)
		assert.Equal(t, []car{{name: "Dacia", age: 22}}, result)
	})

	t.Run("Paginate: limit > Total, 1 to 5", func(t *testing.T) {
		result := make([]car, 0, 5)
		pi, _ := qh.Paginate(1, &result)
		assert.Equal(t, query.PageInfo{Offset: 1, Limit: 5, Count: 2, Total: 3}, pi)
		assert.Equal(t, []car{
			{name: "Mercedes", age: 5, isNew: true},
			{name: "Dacia", age: 22},
		}, result)
	})

	t.Run("Paginate: offset > Total,count = 0, 5 to 1", func(t *testing.T) {
		result := make([]car, 0, 1)
		pi, _ := qh.Paginate(5, &result)
		assert.Equal(t, query.PageInfo{Offset: 5, Limit: 1, Count: 0, Total: 3}, pi)
		assert.Equal(t, []car{}, result)
	})

	t.Run("Paginate: offset+limit > Total, 3 to 1", func(t *testing.T) {
		result := make([]car, 0, 1)
		pi, _ := qh.Paginate(3, &result)
		assert.Equal(t, query.PageInfo{Offset: 3, Limit: 1, Count: 0, Total: 3}, pi)
		assert.Equal(t, []car{}, result)
	})

	t.Run("Paginate: limit = 0", func(t *testing.T) {
		result := make([]car, 0)
		pi, _ := qh.Paginate(1, &result)
		assert.Equal(t, query.PageInfo{Offset: 1, Limit: 0, Count: 0, Total: 3}, pi)
		assert.Equal(t, []car{}, result)
	})

	t.Run("Paginate-error: NIL result", func(t *testing.T) {
		pi, err := qh.Paginate(3, nil)
		assert.Error(t, err)
		assert.Equal(t, query.PageInfo{}, pi)
	})

	t.Run("Paginate-error: result len != 0", func(t *testing.T) {
		result := make([]car, 1)
		pi, err := qh.Paginate(3, &result)
		assert.Error(t, err)
		assert.Equal(t, query.PageInfo{}, pi)
	})
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

func TestIDList_QueryRemove(t *testing.T) {
	l := NewIDList((*car).Name)
	err := l.CreateIndex("age", index.NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	assert.Equal(t, 0, l.Insert(car{name: "Opel", age: 22}))
	assert.Equal(t, 1, l.Insert(car{name: "Mercedes", age: 5}))
	assert.Equal(t, 2, l.Insert(car{name: "Dacia", age: 22}))
	assert.Equal(t, 3, l.Insert(car{name: "Audi", age: 22}))

	assert.Equal(t, 4, l.Len())

	count, err := l.QueryStr(`age = 22`).Remove()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, 1, l.Len())
	l.Values(func(i int, c car) bool {
		switch i {
		case 1:
			assert.Equal(t, c, car{name: "Mercedes", age: 5})
		default:
			t.Logf("invalid car: %v", c)
		}
		return true
	})

	count, err = l.QueryStr(`age = 22`).Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestIDList_QueryUpdate(t *testing.T) {
	l := NewIDList((*car).Name)
	err := l.CreateIndex("age", index.NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	assert.Equal(t, 0, l.Insert(car{name: "Opel", age: 22}))
	assert.Equal(t, 1, l.Insert(car{name: "Mercedes", age: 5}))
	assert.Equal(t, 2, l.Insert(car{name: "Dacia", age: 22}))
	assert.Equal(t, 3, l.Insert(car{name: "Audi", age: 22}))

	assert.Equal(t, 4, l.Len())

	count, err := l.QueryStr(`age = 22`).Update(func(c *car) {
		c.age = c.age + 1
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, 4, l.Len())
	assert.Equal(t, []car{{name: "Opel", age: 23}, {name: "Mercedes", age: 5}, {name: "Dacia", age: 23}, {name: "Audi", age: 23}}, l.ToValues())

	count, err = l.QueryStr(`age = 23`).Count()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestIDList_QueryUpdateError(t *testing.T) {
	l := NewIDList((*car).Name)
	err := l.CreateIndex("age", index.NewSortedIndex((*car).Age))
	assert.NoError(t, err)

	assert.Equal(t, 0, l.Insert(car{name: "Opel", age: 22}))
	assert.Equal(t, 1, l.Insert(car{name: "Mercedes", age: 5}))
	assert.Equal(t, 2, l.Insert(car{name: "Dacia", age: 22}))
	assert.Equal(t, 3, l.Insert(car{name: "Audi", age: 22}))

	assert.Equal(t, 4, l.Len())

	// Name is ID, so you can NOT change it
	count, err := l.QueryStr(`age = 22`).Update(func(c *car) {
		c.name = "FOO"
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "changing the id from:")
	assert.Equal(t, 0, count)
}

func TestIDList_QueryCheckOrder(t *testing.T) {
	newList := func() *IDList[car, string] {
		l := NewIDList((*car).Name)
		assert.NoError(t, l.CreateIndex("age", index.NewSortedIndex((*car).Age)))
		assert.NoError(t, l.CreateIndex("isnew", index.NewMapIndex((*car).IsNew)))

		// two age buckets; the isNew hit sits in the MIDDLE of the age=1 bucket,
		// so an in-place And/AndNot has to compact survivors to the left.
		for i := range 30 {
			l.Insert(car{name: fmt.Sprintf("lo-%d", i), age: 1, isNew: i == 15})
		}
		for i := range 30 {
			l.Insert(car{name: fmt.Sprintf("hi-%d", i), age: 9, isNew: i == 15})
		}

		return l
	}

	names := func(t *testing.T, l *IDList[car, string], q string) []string {
		t.Helper()

		cars, err := l.QueryStr(q).Values()
		assert.NoError(t, err)
		out := make([]string, 0, len(cars))
		for _, c := range cars {
			out = append(out, c.name)
		}

		return out
	}

	queries := []string{
		`age < 2 and isnew = true`,
		`isnew = true and age < 2`,
		`age < 2 and isnew != true`,
		`age < 2 or isnew = true`,
		`age < 2 and not isnew = true`,
	}

	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			l := newList()

			ageBefore := names(t, l, `age < 2`)
			newBefore := names(t, l, `isnew = true`)

			// the query under test is read-only
			_, err := l.QueryStr(q).Count()
			assert.NoError(t, err)

			assert.Equal(t, ageBefore, names(t, l, `age < 2`), "age index was mutated by a read-only query")
			assert.Equal(t, newBefore, names(t, l, `isnew = true`), "isnew index was mutated by a read-only query")
		})
	}
}
