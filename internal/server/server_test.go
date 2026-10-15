package server_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	wheelv1 "github.com/jackmcguire1/wheeloffortune/gen/wheel/v1"
	"github.com/jackmcguire1/wheeloffortune/gen/wheel/v1/wheelv1connect"
	"github.com/jackmcguire1/wheeloffortune/internal/auth"
	"github.com/jackmcguire1/wheeloffortune/internal/server"
	"github.com/jackmcguire1/wheeloffortune/internal/wheel"
	"github.com/jackmcguire1/wheeloffortune/internal/wheel/redisstore"
)

// tokenVerifier treats the bearer token as the user id.
type tokenVerifier struct{}

func (tokenVerifier) Verify(_ context.Context, token string) (*wheel.User, error) {
	if token == "bad" {
		return nil, errors.New("bad token")
	}
	return &wheel.User{UID: token, Name: token}, nil
}

func newClient(t *testing.T) func(token string) wheelv1connect.WheelServiceClient {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	svc := wheel.NewService(redisstore.New(rdb))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mux := http.NewServeMux()
	mux.Handle(wheelv1connect.NewWheelServiceHandler(
		server.New(svc, log),
		connect.WithInterceptors(auth.NewInterceptor(tokenVerifier{})),
	))
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return func(token string) wheelv1connect.WheelServiceClient {
		var opts []connect.ClientOption
		if token != "" {
			opts = append(opts, connect.WithInterceptors(connect.UnaryInterceptorFunc(
				func(next connect.UnaryFunc) connect.UnaryFunc {
					return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
						req.Header().Set("Authorization", "Bearer "+token)
						return next(ctx, req)
					}
				})))
		}
		return wheelv1connect.NewWheelServiceClient(ts.Client(), ts.URL, opts...)
	}
}

func code(err error) connect.Code { return connect.CodeOf(err) }

func TestWheelLifecycle(t *testing.T) {
	client := newClient(t)
	ctx := t.Context()
	alice, bob, anon := client("alice"), client("bob"), client("")

	createReq := &wheelv1.CreateWheelRequest{
		Name: "Raffle",
		Mode: wheelv1.WheelMode_WHEEL_MODE_INVENTORY,
		Segments: []*wheelv1.SegmentSpec{
			{Label: "Bike", Stock: new(int32(1))},
			{Label: "Mug", Stock: new(int32(1))},
		},
	}
	if _, err := anon.CreateWheel(ctx, createReq); code(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous create: got %v, want unauthenticated", err)
	}
	if _, err := client("bad").ListWheels(ctx, &wheelv1.ListWheelsRequest{}); code(err) != connect.CodeUnauthenticated {
		t.Fatalf("bad token: got %v, want unauthenticated", err)
	}
	if _, err := alice.CreateWheel(ctx, &wheelv1.CreateWheelRequest{Name: "x"}); code(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid create: got %v, want invalid_argument", err)
	}

	created, err := alice.CreateWheel(ctx, createReq)
	if err != nil {
		t.Fatal(err)
	}
	w := created.GetWheel()
	if w.GetOwnerUid() != "alice" || w.GetMode() != wheelv1.WheelMode_WHEEL_MODE_INVENTORY || w.GetSegments()[0].GetProbability() != 0.5 {
		t.Fatalf("unexpected wheel %v", w)
	}

	// Public reads.
	if _, err := anon.GetWheel(ctx, &wheelv1.GetWheelRequest{Id: w.GetId()}); err != nil {
		t.Fatal(err)
	}
	if _, err := anon.GetWheel(ctx, &wheelv1.GetWheelRequest{Id: "nope"}); code(err) != connect.CodeNotFound {
		t.Fatalf("missing wheel: got %v, want not_found", err)
	}

	if _, err := anon.SpinWheel(ctx, &wheelv1.SpinWheelRequest{Id: w.GetId()}); code(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous spin: got %v, want unauthenticated", err)
	}
	for range 2 {
		resp, err := bob.SpinWheel(ctx, &wheelv1.SpinWheelRequest{Id: w.GetId()})
		if err != nil {
			t.Fatal(err)
		}
		if resp.GetSpin().GetUserUid() != "bob" || resp.GetSegment().GetSpec().GetLabel() != resp.GetSpin().GetSegmentLabel() {
			t.Fatalf("unexpected spin %v", resp)
		}
	}
	if _, err := bob.SpinWheel(ctx, &wheelv1.SpinWheelRequest{Id: w.GetId()}); code(err) != connect.CodeFailedPrecondition {
		t.Fatalf("exhausted spin: got %v, want failed_precondition", err)
	}

	if _, err := bob.RestockWheel(ctx, &wheelv1.RestockWheelRequest{Id: w.GetId()}); code(err) != connect.CodePermissionDenied {
		t.Fatalf("non owner restock: got %v, want permission_denied", err)
	}
	restocked, err := alice.RestockWheel(ctx, &wheelv1.RestockWheelRequest{Id: w.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	if restocked.GetWheel().GetExhausted() || restocked.GetWheel().GetTotalSpins() != 2 {
		t.Fatalf("unexpected wheel after restock %v", restocked.GetWheel())
	}

	spins, err := anon.ListSpins(ctx, &wheelv1.ListSpinsRequest{Id: w.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	if len(spins.GetSpins()) != 2 {
		t.Fatalf("got %d spins, want 2", len(spins.GetSpins()))
	}

	if _, err := alice.DeleteWheel(ctx, &wheelv1.DeleteWheelRequest{Id: w.GetId()}); err != nil {
		t.Fatal(err)
	}
	list, err := anon.ListWheels(ctx, &wheelv1.ListWheelsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.GetWheels()) != 0 {
		t.Fatalf("deleted wheel still listed")
	}
}
