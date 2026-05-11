package main

import (
	"fmt"

	"github.com/lima1909/mind"
)

type Car struct {
	name string
	age  uint8
	tags []string
}

func (c *Car) Name() string   { return c.name }
func (c *Car) Age() uint8     { return c.age }
func (c *Car) Tags() []string { return c.tags }

func main() {

	l := mind.NewList[Car]()

	err := l.CreateIndex("name", mind.NewMapIndex((*Car).Name))
	if err != nil {
		panic(err)
	}
	err = l.CreateIndex("age", mind.NewSortedIndex((*Car).Age))
	if err != nil {
		panic(err)
	}
	err = l.CreateIndex("tag", mind.NewSortedIndexSlice((*Car).Tags))
	if err != nil {
		panic(err)
	}

	l.Insert(Car{name: "Dacia", age: 2, tags: []string{"blue", "new"}})
	l.Insert(Car{name: "Opel", age: 12, tags: []string{"old", "red"}})
	l.Insert(Car{name: "Mercedes", age: 5})
	l.Insert(Car{name: "Dacia", age: 22, tags: []string{"blue", "old"}})

	t := &mind.Tracer{}
	values, err := l.QueryStr(
		`(name = "Opel" or name = "Dacia") and age >= 2 and tag = "old"`,
		mind.WithTracer(t)).Values()
	if err != nil {
		panic(err)
	}

	fmt.Println(values)
	// Output:
	// [{Opel 12 [old red]} {Dacia 22 [blue old]}

	fmt.Println()
	fmt.Println("Trace:")
	fmt.Println(t.PrettyString())
	// Output:
	// Trace:
	// └── name = Opel OR name = Dacia AND age >= 2 AND tag = old  [5.695µs] (2 matches)
	//     ├── name = Opel OR name = Dacia AND age >= 2  [4.754µs] (3 matches)
	//     │   ├── name = Opel OR name = Dacia  [2.483µs] (3 matches)
	//     │   │   ├── name = Opel  [1.289µs] (1 matches)
	//     │   │   └── name = Dacia  [131ns] (2 matches)
	//     │   └── age >= 2  [1.924µs] (4 matches)
	//     └── tag = old  [733ns] (2 matches)
}
