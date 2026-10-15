package wheel

import (
	"context"
	crand "crypto/rand"
	"math/rand/v2"
	"strings"
	"time"
	"unicode"
)

// Store persists wheels. Spin must pick and consume a segment atomically.
type Store interface {
	Create(ctx context.Context, def Definition) error
	Get(ctx context.Context, id string) (*Wheel, error)
	List(ctx context.Context) ([]*Wheel, error)
	Delete(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	Restock(ctx context.Context, id string) error
	// Spin picks a segment using r, a uniform value in [0, 1), and records spin.
	// It returns the winning segment index.
	Spin(ctx context.Context, id string, r float64, spin Spin) (int, error)
	ListSpins(ctx context.Context, id string, limit int) ([]Spin, error)
}

// User is the authenticated caller of a Service method.
type User struct {
	UID       string
	Name      string
	Anonymous bool
}

// Service implements the wheel use cases on top of a Store.
type Service struct {
	store Store
	now   func() time.Time
	rand  func() float64
}

type Option func(*Service)

// WithClock overrides the time source, for tests.
func WithClock(now func() time.Time) Option { return func(s *Service) { s.now = now } }

// WithRand overrides the spin random source, for tests. It must return values in [0, 1).
func WithRand(r func() float64) Option { return func(s *Service) { s.rand = r } }

func NewService(store Store, opts ...Option) *Service {
	s := &Service{store: store, now: time.Now, rand: rand.Float64}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) Create(ctx context.Context, user *User, def Definition) (*Wheel, error) {
	if user == nil {
		return nil, ErrUnauthenticated
	}
	def.Normalize()
	if err := def.Validate(); err != nil {
		return nil, err
	}
	def.ID = newID(def.Name)
	def.OwnerUID = user.UID
	def.OwnerName = user.Name
	def.CreatedAt = s.now().UTC()
	if err := s.store.Create(ctx, def); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, def.ID)
}

func (s *Service) Get(ctx context.Context, id string) (*Wheel, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]*Wheel, error) {
	return s.store.List(ctx)
}

func (s *Service) Delete(ctx context.Context, user *User, id string) error {
	if _, err := s.owned(ctx, user, id); err != nil {
		return err
	}
	return s.store.Delete(ctx, id)
}

func (s *Service) SetEnabled(ctx context.Context, user *User, id string, enabled bool) (*Wheel, error) {
	if _, err := s.owned(ctx, user, id); err != nil {
		return nil, err
	}
	if err := s.store.SetEnabled(ctx, id, enabled); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, id)
}

func (s *Service) Restock(ctx context.Context, user *User, id string) (*Wheel, error) {
	if _, err := s.owned(ctx, user, id); err != nil {
		return nil, err
	}
	if err := s.store.Restock(ctx, id); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, id)
}

// Spin spins the wheel for user and returns the recorded spin and the wheel
// state after it.
func (s *Service) Spin(ctx context.Context, user *User, id string) (Spin, *Wheel, error) {
	if user == nil {
		return Spin{}, nil, ErrUnauthenticated
	}
	spin := Spin{WheelID: id, UserUID: user.UID, UserName: user.Name, Time: s.now().UTC()}
	idx, err := s.store.Spin(ctx, id, s.rand(), spin)
	if err != nil {
		return Spin{}, nil, err
	}
	w, err := s.store.Get(ctx, id)
	if err != nil {
		return Spin{}, nil, err
	}
	spin.SegmentIndex = idx
	spin.SegmentLabel = w.Segments[idx].Label
	return spin, w, nil
}

func (s *Service) ListSpins(ctx context.Context, id string, limit int) ([]Spin, error) {
	if limit <= 0 {
		limit = DefaultHistory
	}
	limit = min(limit, MaxHistory)
	return s.store.ListSpins(ctx, id, limit)
}

func (s *Service) owned(ctx context.Context, user *User, id string) (*Wheel, error) {
	if user == nil {
		return nil, ErrUnauthenticated
	}
	w, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.OwnerUID != user.UID {
		return nil, ErrPermissionDenied
	}
	return w, nil
}

// newID builds a URL friendly id such as "lunch-picker-k7q2mx".
func newID(name string) string {
	var slug strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if slug.Len() >= 32 {
			break
		}
		switch {
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			slug.WriteRune(r)
			dash = false
		case !dash && slug.Len() > 0:
			slug.WriteByte('-')
			dash = true
		}
	}
	prefix := strings.Trim(slug.String(), "-")
	if prefix == "" {
		prefix = "wheel"
	}
	return prefix + "-" + strings.ToLower(crand.Text()[:6])
}
