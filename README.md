<div align="center">

# Mind 

[![Build Status](https://img.shields.io/github/actions/workflow/status/lima1909/mind/ci.yml?style=for-the-badge)](https://github.com/lima1909/mind/actions)
![License](https://img.shields.io/github/license/lima1909/mind?style=for-the-badge)
[![Stars](https://img.shields.io/github/stars/lima1909/mind?style=for-the-badge)](https://github.com/lima1909/mind/stargazers)

</div>

`Mind (Multi Index List)` finding list items faster by using indexes.

This allows queries / filters to be improved, as is also the case with databases.

<div>
⚠️ <strong>Mind is in a very early stage of development and can change!</strong>
</div>
 
#### Advantage

The fast access can be achieved by using different methods, like;

- hash tables
- indexing
- ...

#### Disadvantage

- it is more memory required. In addition to the user data, data for the _hash_, _index_ are also stored.
- the write operation are slower, because for every wirte operation is an another one (for storing the index data) necessary


#### [Example](https://github.com/lima1909/mind/blob/main/example/main.go)

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
