package internal

import (
	"testing"
)

func TestBool(t *testing.T) {
	tests := []struct {
		in   bool
		want bool
	}{
		{true, true},
		{false, false},
	}
	for _, tt := range tests {
		got := Bool(tt.in)
		if *got != tt.want {
			t.Errorf("got %v\nwant %v", *got, tt.want)
		}
	}
}

func TestIsEnable(t *testing.T) {
	tests := []struct {
		in   *bool
		want bool
	}{
		{Bool(true), true},
		{Bool(false), false},
		{nil, true},
	}
	for _, tt := range tests {
		got := IsEnable(tt.in)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}
