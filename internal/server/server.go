// Package server exposes the wheel Service as a Connect WheelService, which
// speaks Connect, gRPC and gRPC-Web on the same handler.
package server

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	wheelv1 "github.com/jackmcguire1/wheeloffortune/gen/wheel/v1"
	"github.com/jackmcguire1/wheeloffortune/gen/wheel/v1/wheelv1connect"
	"github.com/jackmcguire1/wheeloffortune/internal/auth"
	"github.com/jackmcguire1/wheeloffortune/internal/wheel"
)

type Server struct {
	svc *wheel.Service
	log *slog.Logger
}

var _ wheelv1connect.WheelServiceHandler = (*Server)(nil)

func New(svc *wheel.Service, log *slog.Logger) *Server {
	return &Server{svc: svc, log: log}
}

func (s *Server) CreateWheel(ctx context.Context, req *wheelv1.CreateWheelRequest) (*wheelv1.CreateWheelResponse, error) {
	def := wheel.Definition{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Mode:        modeFromProto(req.GetMode()),
	}
	for _, seg := range req.GetSegments() {
		spec := wheel.SegmentSpec{Label: seg.GetLabel(), Color: seg.GetColor(), Weight: int(seg.GetWeight())}
		if seg.Stock != nil {
			spec.Stock = new(int(seg.GetStock()))
		}
		def.Segments = append(def.Segments, spec)
	}
	w, err := s.svc.Create(ctx, auth.UserFrom(ctx), def)
	if err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	s.log.InfoContext(ctx, "wheel created", "wheel", w.ID, "mode", w.Mode, "owner", w.OwnerUID)
	return &wheelv1.CreateWheelResponse{Wheel: wheelToProto(w)}, nil
}

func (s *Server) GetWheel(ctx context.Context, req *wheelv1.GetWheelRequest) (*wheelv1.GetWheelResponse, error) {
	w, err := s.svc.Get(ctx, req.GetId())
	if err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	return &wheelv1.GetWheelResponse{Wheel: wheelToProto(w)}, nil
}

func (s *Server) ListWheels(ctx context.Context, _ *wheelv1.ListWheelsRequest) (*wheelv1.ListWheelsResponse, error) {
	ws, err := s.svc.List(ctx)
	if err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	resp := &wheelv1.ListWheelsResponse{Wheels: make([]*wheelv1.Wheel, 0, len(ws))}
	for _, w := range ws {
		resp.Wheels = append(resp.Wheels, wheelToProto(w))
	}
	return resp, nil
}

func (s *Server) DeleteWheel(ctx context.Context, req *wheelv1.DeleteWheelRequest) (*wheelv1.DeleteWheelResponse, error) {
	if err := s.svc.Delete(ctx, auth.UserFrom(ctx), req.GetId()); err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	s.log.InfoContext(ctx, "wheel deleted", "wheel", req.GetId())
	return &wheelv1.DeleteWheelResponse{}, nil
}

func (s *Server) SetWheelEnabled(ctx context.Context, req *wheelv1.SetWheelEnabledRequest) (*wheelv1.SetWheelEnabledResponse, error) {
	w, err := s.svc.SetEnabled(ctx, auth.UserFrom(ctx), req.GetId(), req.GetEnabled())
	if err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	return &wheelv1.SetWheelEnabledResponse{Wheel: wheelToProto(w)}, nil
}

func (s *Server) RestockWheel(ctx context.Context, req *wheelv1.RestockWheelRequest) (*wheelv1.RestockWheelResponse, error) {
	w, err := s.svc.Restock(ctx, auth.UserFrom(ctx), req.GetId())
	if err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	return &wheelv1.RestockWheelResponse{Wheel: wheelToProto(w)}, nil
}

func (s *Server) SpinWheel(ctx context.Context, req *wheelv1.SpinWheelRequest) (*wheelv1.SpinWheelResponse, error) {
	spin, w, err := s.svc.Spin(ctx, auth.UserFrom(ctx), req.GetId())
	if err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	s.log.InfoContext(ctx, "wheel spun", "wheel", w.ID, "segment", spin.SegmentIndex, "user", spin.UserUID)
	pw := wheelToProto(w)
	return &wheelv1.SpinWheelResponse{
		SegmentIndex: int32(spin.SegmentIndex),
		Segment:      pw.Segments[spin.SegmentIndex],
		Wheel:        pw,
		Spin:         spinToProto(spin),
	}, nil
}

func (s *Server) ListSpins(ctx context.Context, req *wheelv1.ListSpinsRequest) (*wheelv1.ListSpinsResponse, error) {
	spins, err := s.svc.ListSpins(ctx, req.GetId(), int(req.GetLimit()))
	if err != nil {
		return nil, s.toConnectError(ctx, err)
	}
	resp := &wheelv1.ListSpinsResponse{Spins: make([]*wheelv1.Spin, 0, len(spins))}
	for _, spin := range spins {
		resp.Spins = append(resp.Spins, spinToProto(spin))
	}
	return resp, nil
}

func (s *Server) toConnectError(ctx context.Context, err error) error {
	var ve *wheel.ValidationError
	switch {
	case errors.As(err, &ve):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, wheel.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, wheel.ErrUnauthenticated):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, wheel.ErrPermissionDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, wheel.ErrDisabled), errors.Is(err, wheel.ErrExhausted):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	}
	s.log.ErrorContext(ctx, "internal error", "err", err)
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

func modeFromProto(m wheelv1.WheelMode) wheel.Mode {
	switch m {
	case wheelv1.WheelMode_WHEEL_MODE_WEIGHTED:
		return wheel.ModeWeighted
	case wheelv1.WheelMode_WHEEL_MODE_INVENTORY:
		return wheel.ModeInventory
	default:
		return wheel.ModeEqual
	}
}

func modeToProto(m wheel.Mode) wheelv1.WheelMode {
	switch m {
	case wheel.ModeEqual:
		return wheelv1.WheelMode_WHEEL_MODE_EQUAL
	case wheel.ModeWeighted:
		return wheelv1.WheelMode_WHEEL_MODE_WEIGHTED
	case wheel.ModeInventory:
		return wheelv1.WheelMode_WHEEL_MODE_INVENTORY
	}
	return wheelv1.WheelMode_WHEEL_MODE_UNSPECIFIED
}

func wheelToProto(w *wheel.Wheel) *wheelv1.Wheel {
	probs := w.Probabilities()
	pw := &wheelv1.Wheel{
		Id:          w.ID,
		Name:        w.Name,
		Description: w.Description,
		Mode:        modeToProto(w.Mode),
		Enabled:     w.Enabled,
		TotalSpins:  int32(w.TotalSpins),
		OwnerUid:    w.OwnerUID,
		OwnerName:   w.OwnerName,
		CreateTime:  timestamppb.New(w.CreatedAt),
		Exhausted:   w.Exhausted(),
	}
	for i, spec := range w.Segments {
		seg := &wheelv1.Segment{
			Spec: &wheelv1.SegmentSpec{
				Label:  spec.Label,
				Color:  spec.Color,
				Weight: int32(spec.Weight),
			},
			Wins:        int32(w.State[i].Wins),
			Probability: probs[i],
		}
		if spec.Stock != nil {
			seg.Spec.Stock = new(int32(*spec.Stock))
		}
		if rem := w.State[i].Remaining; rem != nil {
			seg.Remaining = new(int32(*rem))
		}
		pw.Segments = append(pw.Segments, seg)
	}
	return pw
}

func spinToProto(s wheel.Spin) *wheelv1.Spin {
	return &wheelv1.Spin{
		WheelId:      s.WheelID,
		SegmentIndex: int32(s.SegmentIndex),
		SegmentLabel: s.SegmentLabel,
		UserUid:      s.UserUID,
		UserName:     s.UserName,
		SpinTime:     timestamppb.New(s.Time),
	}
}
