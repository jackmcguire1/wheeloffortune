// Package auth authenticates RPC callers from Firebase ID tokens.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	firebase "firebase.google.com/go/v4"
	fbauth "firebase.google.com/go/v4/auth"

	"github.com/jackmcguire1/wheeloffortune/internal/wheel"
)

// Verifier turns a bearer token into a user.
type Verifier interface {
	Verify(ctx context.Context, token string) (*wheel.User, error)
}

// Firebase verifies Firebase Auth ID tokens. When FIREBASE_AUTH_EMULATOR_HOST
// is set the Admin SDK talks to the emulator and accepts its unsigned tokens.
type Firebase struct {
	client *fbauth.Client
}

func NewFirebase(ctx context.Context, projectID string) (*Firebase, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("init firebase auth: %w", err)
	}
	return &Firebase{client: client}, nil
}

func (f *Firebase) Verify(ctx context.Context, token string) (*wheel.User, error) {
	t, err := f.client.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, err
	}
	user := &wheel.User{UID: t.UID, Anonymous: t.Firebase.SignInProvider == "anonymous"}
	for _, claim := range []string{"name", "email"} {
		if v, ok := t.Claims[claim].(string); ok && v != "" {
			user.Name = v
			break
		}
	}
	if user.Name == "" {
		user.Name = "Guest " + shortUID(t.UID)
	}
	return user, nil
}

// Static authenticates every request as the same user. For local development
// with auth turned off.
type Static struct{ User wheel.User }

func (s Static) Verify(context.Context, string) (*wheel.User, error) {
	u := s.User
	return &u, nil
}

type ctxKey struct{}

// UserFrom returns the authenticated caller, or nil for anonymous requests.
func UserFrom(ctx context.Context) *wheel.User {
	u, _ := ctx.Value(ctxKey{}).(*wheel.User)
	return u
}

// WithUser returns a context carrying user.
func WithUser(ctx context.Context, user *wheel.User) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

// NewInterceptor attaches the caller to the request context. Requests without
// a token pass through unauthenticated; requests with an invalid token fail.
// Static verifiers authenticate every request, token or not.
func NewInterceptor(v Verifier) connect.UnaryInterceptorFunc {
	_, static := v.(Static)
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token, ok := strings.CutPrefix(req.Header().Get("Authorization"), "Bearer ")
			token = strings.TrimSpace(token)
			if !ok || token == "" {
				if !static {
					return next(ctx, req)
				}
			}
			user, err := v.Verify(ctx, token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired token"))
			}
			return next(WithUser(ctx, user), req)
		}
	}
}

func shortUID(uid string) string {
	if len(uid) > 4 {
		return uid[:4]
	}
	return uid
}
