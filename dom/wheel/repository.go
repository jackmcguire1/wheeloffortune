package wheel

import "context"

type Repository interface {
	Spin(context.Context, string) (int64, int64, error)
	Create(ctx context.Context, name string, segments []int64) error
	Status(context.Context, string) (*State, error)
	GetAllWheelNames(ctx context.Context) ([]string, error)
}
