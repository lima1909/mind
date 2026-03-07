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

	qr, err := l.QueryStr(`name = "Opel" or name = "Dacia" or age > 10`)
	if err != nil {
		panic(err)
	}

	fmt.Println(qr.Values())

	// Output:
	// [{Dacia 2} {Opel 12} {Dacia 22}]
}
