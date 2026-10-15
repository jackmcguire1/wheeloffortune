// Package wheel holds the wheel-of-fortune domain: definitions, live state,
// the probability rules for each mode and the Service that enforces them.
package wheel

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Mode decides how the winning segment of a spin is chosen.
type Mode string

const (
	// ModeEqual gives every available segment the same chance.
	ModeEqual Mode = "equal"
	// ModeWeighted gives each available segment a chance proportional to its weight.
	ModeWeighted Mode = "weighted"
	// ModeInventory gives each segment a chance proportional to its remaining stock.
	ModeInventory Mode = "inventory"
)

const (
	MinSegments    = 2
	MaxSegments    = 24
	MaxNameLen     = 60
	MaxDescLen     = 280
	MaxLabelLen    = 40
	MaxWeight      = 10_000
	MaxStock       = 1_000_000
	DefaultHistory = 20
	MaxHistory     = 100
)

var (
	ErrNotFound         = errors.New("wheel not found")
	ErrDisabled         = errors.New("wheel is paused")
	ErrExhausted        = errors.New("wheel is out of prizes")
	ErrUnauthenticated  = errors.New("sign in required")
	ErrPermissionDenied = errors.New("only the wheel owner can do that")
)

// ValidationError reports an invalid wheel definition.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Reason }

func invalid(field, format string, args ...any) error {
	return &ValidationError{Field: field, Reason: fmt.Sprintf(format, args...)}
}

// SegmentSpec is the static definition of one slice of the wheel.
type SegmentSpec struct {
	Label  string `json:"label"`
	Color  string `json:"color,omitempty"`
	Weight int    `json:"weight"`
	// Stock is the initial number of prizes; nil means unlimited.
	Stock *int `json:"stock,omitempty"`
}

// Limited reports whether the segment has a finite number of prizes.
func (s SegmentSpec) Limited() bool { return s.Stock != nil }

// Definition is everything a user chose when creating a wheel.
type Definition struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Mode        Mode          `json:"mode"`
	Segments    []SegmentSpec `json:"segments"`
	OwnerUID    string        `json:"ownerUid"`
	OwnerName   string        `json:"ownerName,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
}

// SegmentState holds the live counters of a segment.
type SegmentState struct {
	// Remaining is nil for unlimited segments.
	Remaining *int
	Wins      int
}

// Wheel is a definition together with its live state.
type Wheel struct {
	Definition
	Enabled    bool
	TotalSpins int
	// State is index aligned with Definition.Segments.
	State []SegmentState
}

// Spin records a single spin of a wheel.
type Spin struct {
	WheelID      string
	SegmentIndex int
	SegmentLabel string
	UserUID      string
	UserName     string
	Time         time.Time
}

// Normalize trims user input and fills defaults in place.
func (d *Definition) Normalize() {
	d.Name = strings.TrimSpace(d.Name)
	d.Description = strings.TrimSpace(d.Description)
	if d.Mode == "" {
		d.Mode = ModeEqual
	}
	for i := range d.Segments {
		s := &d.Segments[i]
		s.Label = strings.TrimSpace(s.Label)
		s.Color = strings.TrimSpace(s.Color)
		if s.Weight <= 0 {
			s.Weight = 1
		}
	}
}

// Validate checks a normalized definition.
func (d *Definition) Validate() error {
	if n := utf8.RuneCountInString(d.Name); n == 0 || n > MaxNameLen {
		return invalid("name", "must be between 1 and %d characters", MaxNameLen)
	}
	if utf8.RuneCountInString(d.Description) > MaxDescLen {
		return invalid("description", "must be at most %d characters", MaxDescLen)
	}
	switch d.Mode {
	case ModeEqual, ModeWeighted, ModeInventory:
	default:
		return invalid("mode", "unknown mode %q", d.Mode)
	}
	if n := len(d.Segments); n < MinSegments || n > MaxSegments {
		return invalid("segments", "must have between %d and %d segments", MinSegments, MaxSegments)
	}
	for i, s := range d.Segments {
		field := fmt.Sprintf("segments[%d]", i)
		if n := utf8.RuneCountInString(s.Label); n == 0 || n > MaxLabelLen {
			return invalid(field+".label", "must be between 1 and %d characters", MaxLabelLen)
		}
		if len(s.Color) > 32 {
			return invalid(field+".color", "is too long")
		}
		if s.Weight < 1 || s.Weight > MaxWeight {
			return invalid(field+".weight", "must be between 1 and %d", MaxWeight)
		}
		if s.Stock != nil && (*s.Stock < 0 || *s.Stock > MaxStock) {
			return invalid(field+".stock", "must be between 0 and %d", MaxStock)
		}
		if d.Mode == ModeInventory && s.Stock == nil {
			return invalid(field+".stock", "is required in inventory mode")
		}
	}
	return nil
}

// Weights returns the relative chance of each segment winning the next spin.
// It is the Go mirror of the selection in spin.lua and must stay in sync.
func (w *Wheel) Weights() []int {
	weights := make([]int, len(w.Segments))
	for i, s := range w.Segments {
		remaining := -1
		if s.Limited() {
			remaining = 0
			if i < len(w.State) && w.State[i].Remaining != nil {
				remaining = *w.State[i].Remaining
			}
			if remaining <= 0 {
				continue
			}
		}
		switch w.Mode {
		case ModeEqual:
			weights[i] = 1
		case ModeWeighted:
			weights[i] = s.Weight
		case ModeInventory:
			weights[i] = max(remaining, 0)
		}
	}
	return weights
}

// Probabilities returns each segment's chance of winning the next spin.
func (w *Wheel) Probabilities() []float64 {
	weights := w.Weights()
	total := 0
	for _, v := range weights {
		total += v
	}
	probs := make([]float64, len(weights))
	if total == 0 {
		return probs
	}
	for i, v := range weights {
		probs[i] = float64(v) / float64(total)
	}
	return probs
}

// Exhausted reports whether no segment can be won any more.
func (w *Wheel) Exhausted() bool {
	for _, v := range w.Weights() {
		if v > 0 {
			return false
		}
	}
	return true
}
