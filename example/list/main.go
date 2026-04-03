package main

import (
	"fmt"

	"github.com/lima1909/mind"
)

type Car struct {
	name string
	age  uint8
}

func (c *Car) Name() string { return c.name }
func (c *Car) Age() uint8   { return c.age }

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

	l.Insert(Car{name: "Dacia", age: 2})
	l.Insert(Car{name: "Opel", age: 12})
	l.Insert(Car{name: "Mercedes", age: 5})
	l.Insert(Car{name: "Dacia", age: 22})

	t := &mind.Tracer{}
	values, err := l.QueryStr(`name = "Opel" or name = "Dacia" or age > 10`, mind.WithTracer(t)).Values()
	if err != nil {
		panic(err)
	}

	fmt.Println(values)
	// Output:
	// [{Dacia 2} {Opel 12} {Dacia 22}]

	fmt.Println()
	fmt.Println("Trace:")
	fmt.Println(t.PrettyString())
	// Output:
	// Trace:
	// └── name = Opel OR name = Dacia OR age > 10  [3.759µs] (3 matches)
	//     ├── name = Opel OR name = Dacia  [2.215µs] (3 matches)
	//     │   ├── name = Opel  [1.106µs] (1 matches)
	//     │   └── name = Dacia  [151ns] (2 matches)
	//     └── age > 10  [1.197µs] (2 matches)
}
