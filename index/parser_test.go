package index

import (
	"math"
	"reflect"
	"testing"
)

func TestParser(t *testing.T) {
	tc := []struct {
		name     string
		code     string
		result   []Field
		err      error
		progress int
	}{
		{
			name: "simple",
			code: `field: value string: "value" exact: !value number: 10`,
			result: []Field{
				{
					Name:   "field",
					Type:   FTString,
					String: "value",
				},
				{
					Name:   "string",
					Type:   FTString,
					String: "value",
				},
				{
					Name:   "exact",
					Type:   FTExactString,
					String: "value",
				},
				{
					Name: "number",
					Type: FTInt,
					Int1: 10,
				},
			},
		},
		{
			name: "simple",
			code: `range: -10-20 range: >-20 range: <-20`,
			result: []Field{
				{
					Name: "range",
					Type: FTRange,
					Int1: -10,
					Int2: 20,
				},
				{
					Name: "range",
					Type: FTRange,
					Int1: -20,
					Int2: math.MaxInt32,
				},
				{
					Name: "range",
					Type: FTRange,
					Int1: math.MinInt32,
					Int2: -20,
				},
			},
		},
	}

	var parser Parser

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			result, progress, err := parser.ParseItem([]byte(test.code))
			if err != test.err {
				t.Errorf("expected error %v, got %v", test.err, err)
				if progress != test.progress {
					t.Errorf("expected progress %v, got %v", test.progress, progress)
				}
			}
			if !reflect.DeepEqual(result, test.result) {
				t.Errorf("\n%v\n%v", test.result, result)
			}
		})
	}

}
