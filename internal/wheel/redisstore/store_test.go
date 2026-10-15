package redisstore_test

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/jackmcguire1/wheeloffortune/internal/wheel"
	"github.com/jackmcguire1/wheeloffortune/internal/wheel/redisstore"
)

// newStore uses TEST_REDIS_URL when set (e.g. redis://localhost:6379/15),
// otherwise an in-process miniredis.
func newStore(t *testing.T) *redisstore.Store {
	t.Helper()
	var rdb *redis.Client
	if url := os.Getenv("TEST_REDIS_URL"); url != "" {
		opts, err := redis.ParseURL(url)
		if err != nil {
			t.Fatal(err)
		}
		rdb = redis.NewClient(opts)
		if err := rdb.FlushDB(t.Context()).Err(); err != nil {
			t.Fatal(err)
		}
	} else {
		rdb = redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return redisstore.New(rdb)
}

var alice = &wheel.User{UID: "alice", Name: "Alice"}

func create(t *testing.T, svc *wheel.Service, def wheel.Definition) *wheel.Wheel {
	t.Helper()
	w, err := svc.Create(t.Context(), alice, def)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return w
}

func TestCreateGetList(t *testing.T) {
	svc := wheel.NewService(newStore(t))
	ctx := t.Context()

	w := create(t, svc, wheel.Definition{
		Name: "Prize draw",
		Mode: wheel.ModeInventory,
		Segments: []wheel.SegmentSpec{
			{Label: "Gold", Stock: new(1)},
			{Label: "Silver", Stock: new(3)},
		},
	})
	if !w.Enabled || w.TotalSpins != 0 || w.OwnerUID != "alice" {
		t.Fatalf("unexpected wheel %+v", w)
	}
	if got := *w.State[1].Remaining; got != 3 {
		t.Fatalf("remaining = %d, want 3", got)
	}

	create(t, svc, wheel.Definition{Name: "Second", Segments: []wheel.SegmentSpec{{Label: "a"}, {Label: "b"}}})
	ws, err := svc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 2 {
		t.Fatalf("list returned %d wheels, want 2", len(ws))
	}

	if _, err := svc.Get(ctx, "missing"); !errors.Is(err, wheel.ErrNotFound) {
		t.Fatalf("get missing: %v", err)
	}
}

func TestInventoryDrainsExactly(t *testing.T) {
	svc := wheel.NewService(newStore(t))
	ctx := t.Context()
	w := create(t, svc, wheel.Definition{
		Name: "Draw",
		Mode: wheel.ModeInventory,
		Segments: []wheel.SegmentSpec{
			{Label: "A", Stock: new(5)},
			{Label: "B", Stock: new(10)},
			{Label: "C", Stock: new(0)},
		},
	})

	// Spin concurrently: the Lua script must never oversell.
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := map[int]int{}
	for range 20 {
		wg.Go(func() {
			spin, _, err := svc.Spin(ctx, &wheel.User{UID: "bob"}, w.ID)
			if errors.Is(err, wheel.ErrExhausted) {
				return
			}
			if err != nil {
				t.Errorf("spin: %v", err)
				return
			}
			mu.Lock()
			wins[spin.SegmentIndex]++
			mu.Unlock()
		})
	}
	wg.Wait()

	if wins[0] != 5 || wins[1] != 10 || wins[2] != 0 {
		t.Fatalf("wins = %v, want A:5 B:10 C:0", wins)
	}
	got, err := svc.Get(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Exhausted() || got.TotalSpins != 15 {
		t.Fatalf("exhausted=%v spins=%d, want true 15", got.Exhausted(), got.TotalSpins)
	}

	restocked, err := svc.Restock(ctx, alice, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restocked.Exhausted() || *restocked.State[1].Remaining != 10 {
		t.Fatalf("restock did not reset stock: %+v", restocked.State)
	}
}

func TestWeightedDistribution(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	svc := wheel.NewService(newStore(t), wheel.WithRand(rng.Float64))
	w := create(t, svc, wheel.Definition{
		Name: "Weighted",
		Mode: wheel.ModeWeighted,
		Segments: []wheel.SegmentSpec{
			{Label: "Common", Weight: 6},
			{Label: "Rare", Weight: 3},
			{Label: "Epic", Weight: 1},
		},
	})

	const n = 3000
	counts := make([]int, 3)
	for range n {
		spin, _, err := svc.Spin(t.Context(), alice, w.ID)
		if err != nil {
			t.Fatal(err)
		}
		counts[spin.SegmentIndex]++
	}
	for i, want := range []float64{0.6, 0.3, 0.1} {
		got := float64(counts[i]) / n
		if math.Abs(got-want) > 0.03 {
			t.Errorf("segment %d won %.3f of spins, want ~%.2f", i, got, want)
		}
	}
}

func TestEqualSkipsEmptySegments(t *testing.T) {
	// r close to 1 would land on the last segment if empty ones were counted.
	svc := wheel.NewService(newStore(t), wheel.WithRand(func() float64 { return 0.999 }))
	w := create(t, svc, wheel.Definition{
		Name: "Equal",
		Segments: []wheel.SegmentSpec{
			{Label: "A"},
			{Label: "B", Stock: new(1)},
			{Label: "C", Stock: new(0)},
		},
	})
	want := []int{1, 0, 0}
	for i, idx := range want {
		spin, _, err := svc.Spin(t.Context(), alice, w.ID)
		if err != nil {
			t.Fatal(err)
		}
		if spin.SegmentIndex != idx {
			t.Fatalf("spin %d landed on %d, want %d", i, spin.SegmentIndex, idx)
		}
	}
}

func TestDisabledDeleteAndHistory(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	svc := wheel.NewService(newStore(t), wheel.WithClock(func() time.Time { return now }))
	ctx := t.Context()
	w := create(t, svc, wheel.Definition{Name: "W", Segments: []wheel.SegmentSpec{{Label: "a"}, {Label: "b"}}})

	if _, _, err := svc.Spin(ctx, alice, w.ID); err != nil {
		t.Fatal(err)
	}
	spins, err := svc.ListSpins(ctx, w.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(spins) != 1 || spins[0].UserUID != "alice" || !spins[0].Time.Equal(now) || spins[0].SegmentLabel == "" {
		t.Fatalf("unexpected history %+v", spins)
	}

	if _, err := svc.SetEnabled(ctx, &wheel.User{UID: "mallory"}, w.ID, false); !errors.Is(err, wheel.ErrPermissionDenied) {
		t.Fatalf("non owner pause: %v", err)
	}
	if _, err := svc.SetEnabled(ctx, alice, w.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Spin(ctx, alice, w.ID); !errors.Is(err, wheel.ErrDisabled) {
		t.Fatalf("spin paused wheel: %v", err)
	}

	if err := svc.Delete(ctx, alice, w.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Spin(ctx, alice, w.ID); !errors.Is(err, wheel.ErrNotFound) {
		t.Fatalf("spin deleted wheel: %v", err)
	}
	if ws, _ := svc.List(context.Background()); len(ws) != 0 {
		t.Fatalf("deleted wheel still listed")
	}
}
