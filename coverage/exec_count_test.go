package coverage

import (
	"math"
	"testing"

	"github.com/goccy/go-json"
)

func TestExecCountMarshalClampsToMaxInt64(t *testing.T) {
	// Transitional: stored report.json is read by binaries that decode
	// counts into int, so "count" beyond MaxInt64 must not be emitted;
	// the raw value goes to "count_u64" instead (ignored by int readers).
	c := ExecCount(math.MaxUint64 - 4)
	bc := &BlockCoverage{
		Type:  TypeLOC,
		Count: &c,
	}
	b, err := json.Marshal(bc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Count *int `json:"count"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("marshaled count is not decodable as int: %v\n%s", err, b)
	}
	if want := math.MaxInt64; *decoded.Count != want {
		t.Errorf("got %v\nwant %v", *decoded.Count, want)
	}
}

func TestExecCountRoundTripsViaCountU64(t *testing.T) {
	// uint64-aware readers restore the unclamped value from "count_u64".
	c := ExecCount(math.MaxUint64 - 4)
	bc := &BlockCoverage{
		Type:  TypeLOC,
		Count: &c,
	}
	b, err := json.Marshal(bc)
	if err != nil {
		t.Fatal(err)
	}
	var got BlockCoverage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if *got.Count != c {
		t.Errorf("got %v\nwant %v", *got.Count, c)
	}
}

func TestExecCountMarshalAlwaysEmitsCountU64(t *testing.T) {
	// count_u64 is the canonical field and is emitted for every count, so
	// the schema does not change shape depending on the value.
	c := ExecCount(42)
	bc := &BlockCoverage{
		Type:  TypeLOC,
		Count: &c,
	}
	b, err := json.Marshal(bc)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"type":"loc","count":42,"count_u64":42}`; string(b) != want {
		t.Errorf("got %s\nwant %s", b, want)
	}
}

func TestExecCountUnmarshal(t *testing.T) {
	var bc BlockCoverage
	if err := json.Unmarshal([]byte(`{"type":"loc","count":42}`), &bc); err != nil {
		t.Fatal(err)
	}
	if want := ExecCount(42); *bc.Count != want {
		t.Errorf("got %v\nwant %v", *bc.Count, want)
	}
}

func TestSatAdd(t *testing.T) {
	tests := []struct {
		a, b, want ExecCount
	}{
		{1, 2, 3},
		{math.MaxUint64, 1, math.MaxUint64},
		{math.MaxUint64 - 4, 10, math.MaxUint64},
		{0, math.MaxUint64, math.MaxUint64},
	}
	for _, tt := range tests {
		if got := satAdd(tt.a, tt.b); got != tt.want {
			t.Errorf("satAdd(%v, %v) = %v\nwant %v", tt.a, tt.b, got, tt.want)
		}
	}
}
