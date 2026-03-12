<div align="center">

# Mind 

[![GoDoc](https://pkg.go.dev/badge/github.com/lima1909/mind)](https://pkg.go.dev/github.com/lima1909/mind)
[![Build Status](https://img.shields.io/github/actions/workflow/status/lima1909/mind/ci.yml)](https://github.com/lima1909/mind/actions)
![License](https://img.shields.io/github/license/lima1909/mind)
[![Stars](https://img.shields.io/github/stars/lima1909/mind)](https://github.com/lima1909/mind/stargazers)

</div>

`Mind (Multi INDex list)` finding list items faster by using indexes.

This allows queries / filters to be improved, as is also the case in databases.

`Mind` is particularly well suited where data is read more than written.


<div>
⚠️ <strong>Mind is in a very early stage of development and can change!</strong>
</div>

 
#### Advantage

The fast access can be achieved by using different indexes:

- `MapIndex` (hash map), supported operation art `=, !=`
- `MapIndex` ([SkipList](https://en.wikipedia.org/wiki/Skip_list)), supported operation art `=, !=, >, >=, <, <=, between(from, to)`

=> All operations can be combined with `or`, `and` or `not`.

#### Disadvantage

- it is more memory required. In addition to the user data, data for the _hash_, _index_ are also stored.
- the write operation are slower, because for every wirte operation is an another one (for storing the index data) necessary


#### Example

[List](https://github.com/lima1909/mind/blob/main/example/list/main.go)

```go
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
```

[List with ID](https://github.com/lima1909/mind/blob/main/example/idlist/main.go)

```go
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

	result, _ := l.Query(mind.Or(mind.Eq("name", "Opel"), mind.Lt("age", 10)))
	fmt.Println(result.Values())
	// Output:
	// [{1 Dacia 2} {2 Opel 12} {3 Mercedes 5}]
}
```
