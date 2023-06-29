//go:build unit
// +build unit

package pili

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator_Validate(t *testing.T) {
	ast := assert.New(t)

	type Foo struct {
		Num    int    `validate:"gt=0,lte=10"`
		String string `validate:"required"`
	}

	// struct
	struct1 := Foo{}
	struct2 := Foo{Num: 1, String: "bar"}
	err := defaultValidator.Validate(struct1)
	ast.NotNil(err)
	err = defaultValidator.Validate(struct2)
	ast.Nil(err)

	// struct ptr
	err = defaultValidator.Validate(&struct1)
	ast.NotNil(err)
	err = defaultValidator.Validate(&struct2)
	ast.Nil(err)

	// slice/array struct
	slice1 := []Foo{{}, {}}
	slice2 := []Foo{{Num: 1, String: "bar"}, {Num: 2, String: "bar2"}}
	err = defaultValidator.Validate(slice1)
	ast.NotNil(err)
	err = defaultValidator.Validate(slice2)
	ast.Nil(err)

	// not dive
	type NotDive struct {
		Foos []Foo
	}
	notDive := NotDive{Foos: []Foo{{}, {}}}
	err = defaultValidator.Validate(notDive)
	ast.Nil(err)

	// dive
	type Dive struct {
		Foos []Foo `validate:"dive"`
	}
	dive1 := Dive{Foos: []Foo{{}, {}}}
	dive2 := Dive{Foos: []Foo{{Num: 1, String: "bar"}, {Num: 2, String: "bar2"}}}
	err = defaultValidator.Validate(dive1)
	ast.NotNil(err)
	err = defaultValidator.Validate(dive2)
	ast.Nil(err)
}
