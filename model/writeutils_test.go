package model

import "testing"

func TestFmtFloat(t *testing.T) {
	tests := []struct {
		want string
		args Fl
	}{
		{"1", 1},
		{"1.123", 1.123},
		{"1.12345", 1.12345},
		{"1.12346", 1.123456},
		{"1", 1.0000000000000001},
	}
	for _, tt := range tests {
		if got := FmtFloat(tt.args); got != tt.want {
			t.Errorf("FmtFloat() = %v, want %v", got, tt.want)
		}
	}
}
