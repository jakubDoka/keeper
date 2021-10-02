package index

import (
	"reflect"
	"testing"
)

func TestIndexSearch(t *testing.T) {
	index := New()
	var parser Parser

	index.AddCategory("string", &StringIndexCategory{})
	index.AddCategory("int", &IntIndexCategory{})

	var id int

	insert := func(data string) {
		fields, idx, err := parser.Parse([]byte(data))
		if err != nil {
			t.Fatal(err, idx)
		}
		for i := range fields {
			fields[i].Value = id
		}
		id++
		index.Insert(fields...)
	}

	insertAll := func(data ...string) {
		for _, d := range data {
			insert(d)
		}
	}

	insertAll(
		"string: ab",
		"string: abc",
		"string: abcd",
		"string: abcde",
		"string: abcdef",
		"int: 10",
		"int: 20",
		"int: 30",
		"int: 40 string: goo",
		"int: 50 string: foo",
	)

	tests := []struct {
		name, query string
		result      testBuffer
	}{
		{
			"string",
			"string: ab",
			testBuffer{
				0: 1,
				1: 1,
				2: 1,
				3: 1,
				4: 1,
			},
		},
		{
			"int",
			"int: 2-60",
			testBuffer{
				5: 1,
				6: 1,
				7: 1,
				8: 1,
				9: 1,
			},
		},
		{
			"concrete string",
			"string: !ab",
			testBuffer{
				0: 1,
			},
		},
		{
			"concrete int",
			"int: 30",
			testBuffer{
				7: 1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fields, i, err := parser.Parse([]byte(test.query))
			if err != nil {
				t.Fatal(err, i)
			}
			result := testBuffer{}
			index.Search(result, fields...)
			if !reflect.DeepEqual(result, test.result) {
				t.Errorf("\n%v\n%v", test.result, result)
			}
		})
	}
}

type testBuffer map[int]int

func (m testBuffer) Add(value interface{}) {
	m[value.(int)]++
}
