package badge

import "testing"

func TestParseColor(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"green", ColorGreen, false},
		{"YellowGreen", ColorYellowGreen, false},
		{"gray", ColorGrey, false},
		{"#123456", "#123456", false},
		{"123abc", "#123ABC", false},
		{" #123456 ", "#123456", false},
		{"#123", "", true},
		{"octocov", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseColor(tt.in)
			if err != nil {
				if !tt.wantErr {
					t.Fatal(err)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("want error")
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}
