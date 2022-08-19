package wheel

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

var (
	OutOfPrizes = errors.New("out of prizes")
	NotEnabled  = errors.New("wheel not enabled")
)

type State struct {
	Prizes  []int64 `json:"prizes"`
	Spins   int64   `json:"spins"`
	Enabled bool    `json:"enabled"`
}

func (svc *WheelService) Spin(
	ctx context.Context,
	name string,
) (
	i int64,
	remainingPrizes int64,
	err error,
) {
	return svc.Repo.Spin(ctx, name)
}

func (svc *WheelService) Create(
	ctx context.Context,
	name string,
	segments []int64,
) (
	err error,
) {

	// check whether to apply random prize pool for each wheel segment
	if len(segments) != svc.SegmentSize {
		segments = make([]int64, svc.SegmentSize)

		rand.Seed(time.Now().UTC().UnixNano())
		for i := 0; i < svc.SegmentSize; i++ {
			randomValue := rand.Int63n(100)
			segments[i] = randomValue
		}
	}

	return svc.Repo.Create(ctx, name, segments)
}

func (svc *WheelService) Status(ctx context.Context, name string) (*State, error) {
	return svc.Repo.Status(ctx, name)
}

func (svc *WheelService) GetAllWheelNames(ctx context.Context) ([]string, error) {
	return svc.Repo.GetAllWheelNames(ctx)
}
