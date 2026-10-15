package wheel

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	two := []SegmentSpec{{Label: "a"}, {Label: "b"}}
	tests := []struct {
		name  string
		def   Definition
		field string
	}{
		{"ok", Definition{Name: "Lunch", Segments: two}, ""},
		{"empty name", Definition{Name: "  ", Segments: two}, "name"},
		{"long name", Definition{Name: strings.Repeat("x", MaxNameLen+1), Segments: two}, "name"},
		{"bad mode", Definition{Name: "x", Mode: "chaos", Segments: two}, "mode"},
		{"one segment", Definition{Name: "x", Segments: two[:1]}, "segments"},
		{"empty label", Definition{Name: "x", Segments: []SegmentSpec{{Label: "a"}, {Label: " "}}}, "segments[1].label"},
		{"negative stock", Definition{Name: "x", Segments: []SegmentSpec{{Label: "a", Stock: new(-1)}, {Label: "b"}}}, "segments[0].stock"},
		{"inventory needs stock", Definition{Name: "x", Mode: ModeInventory, Segments: []SegmentSpec{{Label: "a", Stock: new(1)}, {Label: "b"}}}, "segments[1].stock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.def.Normalize()
			err := tt.def.Validate()
			if tt.field == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Field != tt.field {
				t.Fatalf("got %v, want error on %s", err, tt.field)
			}
		})
	}
}

func TestProbabilities(t *testing.T) {
	segs := []SegmentSpec{
		{Label: "a", Weight: 3},
		{Label: "b", Weight: 1, Stock: new(4)},
		{Label: "c", Weight: 6, Stock: new(2)},
	}
	state := []SegmentState{{}, {Remaining: new(4)}, {Remaining: new(0)}}
	tests := []struct {
		mode Mode
		want []float64
	}{
		{ModeEqual, []float64{0.5, 0.5, 0}},
		{ModeWeighted, []float64{0.75, 0.25, 0}},
		{ModeInventory, []float64{0, 1, 0}},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			w := &Wheel{Mode: tt.mode, Segments: segs, State: state}
			got := w.Probabilities()
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("probabilities = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestExhausted(t *testing.T) {
	w := &Wheel{
		Mode: ModeEqual, Segments: []SegmentSpec{{Label: "a", Stock: new(1)}, {Label: "b", Stock: new(1)}},
		State: []SegmentState{{Remaining: new(0)}, {Remaining: new(0)}},
	}
	if !w.Exhausted() {
		t.Fatal("wheel with no stock should be exhausted")
	}
}

func TestNewID(t *testing.T) {
	re := regexp.MustCompile(`^[a-z0-9-]+-[a-z2-7]{6}$`)
	for name, prefix := range map[string]string{
		"Lunch Picker!": "lunch-picker-",
		"🎉🎉":            "wheel-",
		"  --Friday--":  "friday-",
	} {
		id := newID(name)
		if !strings.HasPrefix(id, prefix) || !re.MatchString(id) {
			t.Errorf("newID(%q) = %q, want prefix %q", name, id, prefix)
		}
	}
}
