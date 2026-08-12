package scale

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		model string
		frame string
		want  string
	}{
		{
			name:  "HEB-TW captured frame",
			model: "HEB-TW",
			frame: "wn 0002.02kg\r\n",
			want:  "ST,GS     2.02kg",
		},
		{
			name:  "HEB-TW model ignores whitespace and casing",
			model: "  Heb-Tw ",
			frame: "WN 0000.00kg",
			want:  "ST,GS     0.00kg",
		},
		{
			name:  "empty model uses default",
			model: "",
			frame: "ST,GS     59.6kg",
			want:  "ST,GS     59.6kg",
		},
		{
			name:  "default accepts legacy comma separator",
			model: "default",
			frame: "ST,GS,+001.234kg",
			want:  "ST,GS     +1.234kg",
		},
		{
			name:  "default negative weight",
			model: "default",
			frame: "ST,GS-     2.5kg",
			want:  "ST,GS     -2.5kg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.model, tt.frame)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Parse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRejectsInvalidFrame(t *testing.T) {
	if _, err := Parse(ModelHEBTW, "ST,GS     2.02kg"); err == nil {
		t.Fatal("Parse() should reject a frame from another model")
	}
}
