package main

import (
	"fmt"

	"github.com/lima1909/mind"
)

type Car struct {
	id   uint
	name string
	age  uint8
}

func (c *Car) ID() uint     { return c.id }
func (c *Car) Name() string { return c.name }
func (c *Car) Age() uint8   { return c.age }

func main() {

	l := mind.NewListWithID((*Car).ID)

	// ignore error
	_ = l.CreateIndex("name", mind.NewMapIndex((*Car).Name))
	_ = l.CreateIndex("age", mind.NewSortedIndex((*Car).Age))

	l.Insert(Car{id: 1, name: "Dacia", age: 2})
	l.Insert(Car{id: 2, name: "Opel", age: 12})
	l.Insert(Car{id: 3, name: "Mercedes", age: 5})
	l.Insert(Car{id: 4, name: "Dacia", age: 22})

	// ignore error
	mercedes, _ := l.Get(3)
	fmt.Println(mercedes)
	// Output:
	// {3 Mercedes 5

	removed, _ := l.Remove(4)
	fmt.Println(removed)
	// Output:
	// true

	t := &mind.Tracer{}
	values, _ := l.Query(mind.Or(mind.Eq("name", "Opel"), mind.Lt("age", 10)), mind.WithTracer(t)).Values()
	fmt.Println(values)
	// Output:
	// [{1 Dacia 2} {2 Opel 12} {3 Mercedes 5}]

	fmt.Println()
	fmt.Println("Trace:")
	fmt.Println(t.PrettyString())
	// Output:
	// Trace:
	// └── name = Opel OR age < 10  [1.85µs] (3 matches)
	//     ├── name = Opel  [620ns] (1 matches)
	//     └── age < 10  [790ns] (2 matches)
}
