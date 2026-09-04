package internal

import (
	"testing"
)

func TestIsEnable(t *testing.T) {
	tests := []struct {
		in   *bool
		want bool
	}{
		{new(true), true},
		{new(false), false},
		{nil, true},
	}
	for _, tt := range tests {
		got := IsEnable(tt.in)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}
