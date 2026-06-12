package main

import "testing"

func TestStripUnit(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want float64
	}{
		{"plain number", "210", 210},
		{"mhz no space", "210MHz", 210},
		{"mhz with space", "210 MHz", 210},
		{"celsius no space", "39C", 39},
		{"celsius with space", "39 C", 39},
		{"percent no space", "100%", 100},
		{"percent with space", "0 %", 0},
		{"watts with decimal and space", "15.51 W", 15.51},
		{"surrounding whitespace", "  42 W  ", 42},
		{"empty is zero", "", 0},
		{"whitespace only is zero", "   ", 0},
		{"raw bytes value", "25769803776", 25769803776},
		{"unparseable is zero", "n/a", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripUnit(tt.in); got != tt.want {
				t.Errorf("stripUnit(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// nvtop 3.3.1 omits the comma between fields: a value-closing
			// quote is followed by a newline and the next field's opening quote.
			name: "inserts missing comma between fields",
			in:   "{\n  \"a\": \"1\"\n  \"b\": \"2\"\n}",
			want: "{\n  \"a\": \"1\",\n  \"b\": \"2\"\n}",
		},
		{
			name: "leaves well-formed json untouched",
			in:   "{\n  \"a\": \"1\",\n  \"b\": \"2\"\n}",
			want: "{\n  \"a\": \"1\",\n  \"b\": \"2\"\n}",
		},
		{
			name: "fixes multiple missing commas",
			in:   "{\n  \"a\": \"1\"\n  \"b\": \"2\"\n  \"c\": \"3\"\n}",
			want: "{\n  \"a\": \"1\",\n  \"b\": \"2\",\n  \"c\": \"3\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(sanitizeJSON([]byte(tt.in))); got != tt.want {
				t.Errorf("sanitizeJSON(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
