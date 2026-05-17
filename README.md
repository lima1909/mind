<div align="center">

# Mind 

**Fast, in-memory indexed collections for Go — filter your data like a database.**

[![GoDoc](https://pkg.go.dev/badge/github.com/lima1909/mind)](https://pkg.go.dev/github.com/lima1909/mind)
[![Build Status](https://img.shields.io/github/actions/workflow/status/lima1909/mind/ci.yml)](https://github.com/lima1909/mind/actions)
![License](https://img.shields.io/github/license/lima1909/mind)
[![Stars](https://img.shields.io/github/stars/lima1909/mind)](https://github.com/lima1909/mind/stargazers)

</div>

**Mind** (Multi INDex list) lets you query in-memory collections by multiple fields using indexes, just like a database — but without one.
It is particularly well suited where data is **read more often than written**.

> ⚠️ Mind is in an early stage of development and the API may change.

## Installation

```bash
go get github.com/lima1909/mind
```

## Features

### Index Types

| Index         | Backed by                                           | Supported operations                                                        |
|---------------|-----------------------------------------------------|-----------------------------------------------------------------------------|
| `MapIndex`    | Hash map                                            | `=`, `!=`, `In`                                                             |
| `SortedIndex` | [SkipList](https://en.wikipedia.org/wiki/Skip_list) | `=`, `!=` , `>`, `>=`, `<`, `<=`, `Between`, `In`                           |
| `RangeIndex`  | uint8 slice                                         | `=`, `!=` , `>`, `>=`, `<`, `<=`, `Between`, `In`                           |
| `StringIndex` | SkipList and TrigramIndex                           | `=`, `!=` , `>`, `>=`, `<`, `<=`, `Between`, `In`, `contains`, `startswith` |

**All operations** can be combined with `AND`, `OR` and `NOT`.

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
		mind.WithTracer(t),
	).Values()
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

	t := &mind.Tracer{}
	result, _ := l.Query(mind.Or(mind.Eq("name", "Opel"), mind.Lt("age", 10)), mind.WithTracer(t))
	fmt.Println(result.Values())
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
```
