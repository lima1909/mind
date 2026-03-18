<div align="center">

# Mind 

[![GoDoc](https://pkg.go.dev/badge/github.com/lima1909/mind)](https://pkg.go.dev/github.com/lima1909/mind)
[![Build Status](https://img.shields.io/github/actions/workflow/status/lima1909/mind/ci.yml)](https://github.com/lima1909/mind/actions)
![License](https://img.shields.io/github/license/lima1909/mind)
[![Stars](https://img.shields.io/github/stars/lima1909/mind)](https://github.com/lima1909/mind/stargazers)

**Fast, in-memory indexed collections for Go — filter your data like a database.**

</div>

`Mind` **(Multi INDex list)** lets you query in-memory collections by multiple fields using indexes, just like a database — but without one.
It is particularly well suited where data is **read more often than written**.

> ⚠️ Mind is in an early stage of development and the API may change.

## Installation

```bash
go get github.com/lima1909/mind
```

## Features

### Index Types

| Index | Backed by | Supported operations |
|-------|-----------|---------------------|
| `MapIndex` | Hash map | `=`, `!=` |
| `SortedIndex` | [SkipList](https://en.wikipedia.org/wiki/Skip_list) | `=`, `!=`, `>`, `>=`, `<`, `<=`, `between`, `startswith` |

All operations can be combined with `and`, `or` and `not`.

## Trade-offs

#### Advantages
- Zero dependencies
- Generic — works with any struct type
- Fast reads via bitmap-accelerated index intersection
- SQL-like query language (with optimizer)

#### Disadvantages
- Higher memory usage: indexes store additional data alongside user data
- Slower writes: every mutation updates all registered indexes


## Examples

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
