package main

import (
	"testing"
	"time"
)

func TestKv(t *testing.T) {
	tests := []struct {
		key    string
		value  any
		expect string
	}{
		{
			key:    "int",
			value:  1,
			expect: "int=1",
		},
		{
			key:    "duration",
			value:  1 * time.Second,
			expect: "duration=1s",
		},
		{
			key:    "string",
			value:  "test",
			expect: `string="test"`,
		},
	}
	for _, tc := range tests {
		got := kv(tc.key, tc.value)
		if tc.expect != got {
			t.Errorf("expected: `%s`, got: `%s`", tc.expect, got)
		}
	}
}
