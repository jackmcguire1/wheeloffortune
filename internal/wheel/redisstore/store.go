// Package redisstore implements wheel.Store on Redis.
//
// Each wheel uses the following keys, hash tagged on the id so a wheel's keys
// share a cluster slot:
//
//	wheel:{id}:def      string  JSON wheel.Definition
//	wheel:{id}:state    hash    enabled (0|1), spins
//	wheel:{id}:stock    hash    segment index -> remaining prizes (limited segments only)
//	wheel:{id}:wins     hash    segment index -> wins
//	wheel:{id}:history  list    JSON spin records, newest first
//	wheels              zset    wheel id scored by creation time (unix ms)
package redisstore

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/jackmcguire1/wheeloffortune/internal/wheel"
)

//go:embed spin.lua
var spinLua string

var spinScript = redis.NewScript(spinLua)

const indexKey = "wheels"

type keys struct{ def, state, stock, wins, history string }

func keysFor(id string) keys {
	p := "wheel:{" + id + "}:"
	return keys{p + "def", p + "state", p + "stock", p + "wins", p + "history"}
}

func (k keys) all() []string { return []string{k.def, k.state, k.stock, k.wins, k.history} }

// Store is a Redis backed wheel.Store.
type Store struct {
	rdb redis.UniversalClient
}

var _ wheel.Store = (*Store)(nil)

func New(rdb redis.UniversalClient) *Store { return &Store{rdb: rdb} }

func (s *Store) Create(ctx context.Context, def wheel.Definition) error {
	raw, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("encode wheel: %w", err)
	}
	k := keysFor(def.ID)
	_, err = s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, k.def, raw, 0)
		p.HSet(ctx, k.state, "enabled", 1, "spins", 0)
		if stock := initialStock(def); len(stock) > 0 {
			p.HSet(ctx, k.stock, stock)
		}
		p.ZAdd(ctx, indexKey, redis.Z{Score: float64(def.CreatedAt.UnixMilli()), Member: def.ID})
		return nil
	})
	if err != nil {
		return fmt.Errorf("create wheel %q: %w", def.ID, err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (*wheel.Wheel, error) {
	ws, err := s.getMany(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	if ws[0] == nil {
		return nil, wheel.ErrNotFound
	}
	return ws[0], nil
}

func (s *Store) List(ctx context.Context) ([]*wheel.Wheel, error) {
	ids, err := s.rdb.ZRevRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("list wheels: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	ws, err := s.getMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := ws[:0]
	for _, w := range ws {
		if w != nil {
			out = append(out, w)
		}
	}
	return out, nil
}

// getMany loads wheels in one round trip. Missing wheels are nil.
func (s *Store) getMany(ctx context.Context, ids []string) ([]*wheel.Wheel, error) {
	type cmds struct {
		def               *redis.StringCmd
		state, stock, win *redis.MapStringStringCmd
	}
	results := make([]cmds, len(ids))
	_, err := s.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for i, id := range ids {
			k := keysFor(id)
			results[i] = cmds{
				def:   p.Get(ctx, k.def),
				state: p.HGetAll(ctx, k.state),
				stock: p.HGetAll(ctx, k.stock),
				win:   p.HGetAll(ctx, k.wins),
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("load wheels: %w", err)
	}

	out := make([]*wheel.Wheel, len(ids))
	for i, c := range results {
		raw, err := c.def.Bytes()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("load wheel %q: %w", ids[i], err)
		}
		w := &wheel.Wheel{}
		if err := json.Unmarshal(raw, &w.Definition); err != nil {
			return nil, fmt.Errorf("decode wheel %q: %w", ids[i], err)
		}
		state := c.state.Val()
		stock := c.stock.Val()
		wins := c.win.Val()
		w.Enabled = state["enabled"] == "1"
		w.TotalSpins = atoi(state["spins"])
		w.State = make([]wheel.SegmentState, len(w.Segments))
		for j, seg := range w.Segments {
			field := strconv.Itoa(j)
			w.State[j].Wins = atoi(wins[field])
			if seg.Limited() {
				w.State[j].Remaining = new(max(atoi(stock[field]), 0))
			}
		}
		out[i] = w
	}
	return out, nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	k := keysFor(id)
	_, err := s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, k.all()...)
		p.ZRem(ctx, indexKey, id)
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete wheel %q: %w", id, err)
	}
	return nil
}

func (s *Store) SetEnabled(ctx context.Context, id string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	if err := s.rdb.HSet(ctx, keysFor(id).state, "enabled", v).Err(); err != nil {
		return fmt.Errorf("set wheel %q enabled: %w", id, err)
	}
	return nil
}

func (s *Store) Restock(ctx context.Context, id string) error {
	w, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	stock := initialStock(w.Definition)
	if len(stock) == 0 {
		return nil
	}
	if err := s.rdb.HSet(ctx, keysFor(id).stock, stock).Err(); err != nil {
		return fmt.Errorf("restock wheel %q: %w", id, err)
	}
	return nil
}

func (s *Store) Spin(ctx context.Context, id string, r float64, spin wheel.Spin) (int, error) {
	k := keysFor(id)
	res, err := spinScript.Run(ctx, s.rdb,
		[]string{k.def, k.state, k.stock, k.wins, k.history},
		strconv.FormatFloat(r, 'f', -1, 64),
		spin.UserUID,
		spin.UserName,
		spin.Time.UnixMilli(),
		wheel.MaxHistory,
	).Int()
	if err != nil {
		return 0, fmt.Errorf("spin wheel %q: %w", id, err)
	}
	switch res {
	case -1:
		return 0, wheel.ErrNotFound
	case -2:
		return 0, wheel.ErrDisabled
	case -3:
		return 0, wheel.ErrExhausted
	}
	return res, nil
}

type historyRecord struct {
	Index int    `json:"i"`
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Time  int64  `json:"t"`
}

func (s *Store) ListSpins(ctx context.Context, id string, limit int) ([]wheel.Spin, error) {
	w, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	raws, err := s.rdb.LRange(ctx, keysFor(id).history, 0, int64(limit)-1).Result()
	if err != nil {
		return nil, fmt.Errorf("list spins of %q: %w", id, err)
	}
	spins := make([]wheel.Spin, 0, len(raws))
	for _, raw := range raws {
		var rec historyRecord
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			return nil, fmt.Errorf("decode spin of %q: %w", id, err)
		}
		spin := wheel.Spin{
			WheelID:      id,
			SegmentIndex: rec.Index,
			UserUID:      rec.UID,
			UserName:     rec.Name,
			Time:         time.UnixMilli(rec.Time).UTC(),
		}
		if rec.Index >= 0 && rec.Index < len(w.Segments) {
			spin.SegmentLabel = w.Segments[rec.Index].Label
		}
		spins = append(spins, spin)
	}
	return spins, nil
}

func initialStock(def wheel.Definition) map[string]any {
	stock := map[string]any{}
	for i, seg := range def.Segments {
		if seg.Limited() {
			stock[strconv.Itoa(i)] = *seg.Stock
		}
	}
	return stock
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
