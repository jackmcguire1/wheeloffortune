package wheel

import "context"

type Service interface {
	Spin(context.Context, string) (int64, int64, error)
	Create(context.Context, string, []int64) (err error)
	Status(context.Context, string) (*State, error)
	GetAllWheelNames(ctx context.Context) ([]string, error)
}

type Resources struct {
	SegmentSize int
	Repo        Repository
}

type WheelService struct {
	*Resources
}

func New(r *Resources) *WheelService {
	return &WheelService{
		r,
	}
}
