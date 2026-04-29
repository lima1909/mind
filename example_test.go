package mind

import (
	"fmt"
)

type carType = int

const (
	van carType = iota
	suv
	electric
	coupe
)

type car struct {
	name    string
	color   string
	age     uint8
	isNew   bool
	carType carType
}

func (c *car) Name() string { return c.name }
func (c *car) Age() uint8   { return c.age }
func (c *car) IsNew() bool  { return c.isNew }
func (c *car) Type() int    { return c.carType }

func ExampleParserExt() {
	il := NewList[car]()
	// ignore error
	_ = il.CreateIndex("type",
		NewParserExt(
			NewSortedIndex((*car).Type), func(s string) any {
				switch s {
				case "van":
					return van
				case "suv":
					return suv
				case "electric":
					return electric
				case "coupe":
					return coupe

				}
				return -1
			}),
	)

	il.Insert(car{name: "Opel", carType: electric})
	il.Insert(car{name: "Mercedes", carType: suv})
	il.Insert(car{name: "Dacia", carType: suv})
	il.Insert(car{name: "Opel", carType: coupe})

	// ignore error
	cars, _ := il.QueryStr(`type = "electric" or type = "coupe"`).Values()
	for _, c := range cars {
		fmt.Printf("%#v\n", c)
	}
	// Output:
	// mind.car{name:"Opel", color:"", age:0x0, isNew:false, carType:2}
	// mind.car{name:"Opel", color:"", age:0x0, isNew:false, carType:3}
}
