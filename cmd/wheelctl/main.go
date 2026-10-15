// Command wheelctl manages and spins wheels from the terminal.
//
//	wheelctl list
//	wheelctl create -name "Prize draw" -mode inventory -s Gold::1 -s Silver::5 -s "Try again::50"
//	wheelctl spin -n 100 -c 8 prize-draw-k2m4xq
//
// Authentication: -token (or WHEEL_TOKEN) is sent as a bearer token. When
// FIREBASE_AUTH_EMULATOR_HOST is set and no token is given, wheelctl signs in
// to the emulator as -email, creating the account on first use.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"

	wheelv1 "github.com/jackmcguire1/wheeloffortune/gen/wheel/v1"
	"github.com/jackmcguire1/wheeloffortune/gen/wheel/v1/wheelv1connect"
)

const usage = `wheelctl manages and spins wheels of fortune.

Usage:
  wheelctl [global flags] <command> [flags] [wheel-id]

Commands:
  list                      list all wheels
  get <id>                  show a wheel and its segments
  create [flags]            create a wheel
  spin [-n N] [-c C] <id>   spin a wheel N times with C concurrent workers
  history [-limit N] <id>   show recent spins
  pause <id>                stop a wheel from being spun
  resume <id>               allow a wheel to be spun again
  restock <id>              reset every segment to its initial stock
  delete <id>               delete a wheel

Global flags:
`

type cli struct {
	client wheelv1connect.WheelServiceClient
}

var errUsage = errors.New("usage")

func main() {
	if err := run(); err != nil {
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	global := flag.NewFlagSet("wheelctl", flag.ExitOnError)
	addr := global.String("addr", env("WHEEL_ADDR", "http://localhost:7755"), "server URL")
	token := global.String("token", os.Getenv("WHEEL_TOKEN"), "bearer token (Firebase ID token)")
	email := global.String("email", env("WHEEL_EMAIL", "cli@wheeloffortune.dev"), "emulator account used when no token is given")
	global.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		global.PrintDefaults()
	}
	_ = global.Parse(os.Args[1:])
	if global.NArg() == 0 {
		global.Usage()
		return errUsage
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if *token == "" {
		if host := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"); host != "" {
			t, err := emulatorSignIn(ctx, host, *email)
			if err != nil {
				return fmt.Errorf("sign in to auth emulator: %w", err)
			}
			*token = t
		}
	}

	var opts []connect.ClientOption
	if *token != "" {
		opts = append(opts, connect.WithInterceptors(bearer(*token)))
	}
	c := cli{client: wheelv1connect.NewWheelServiceClient(http.DefaultClient, *addr, opts...)}

	cmd, args := global.Arg(0), global.Args()[1:]
	switch cmd {
	case "list":
		return c.list(ctx)
	case "get":
		return c.get(ctx, args)
	case "create":
		return c.create(ctx, args)
	case "spin":
		return c.spin(ctx, args)
	case "history":
		return c.history(ctx, args)
	case "pause", "resume":
		return c.setEnabled(ctx, args, cmd == "resume")
	case "restock":
		return c.restock(ctx, args)
	case "delete":
		return c.delete(ctx, args)
	}
	global.Usage()
	return errUsage
}

func (c cli) list(ctx context.Context) error {
	resp, err := c.client.ListWheels(ctx, &wheelv1.ListWheelsRequest{})
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tMODE\tSEGMENTS\tSPINS\tSTATUS")
	for _, w := range resp.GetWheels() {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%s\n", w.GetId(), w.GetName(), modeName(w.GetMode()), len(w.GetSegments()), w.GetTotalSpins(), status(w))
	}
	return tw.Flush()
}

func (c cli) get(ctx context.Context, args []string) error {
	id, err := oneID("get", args)
	if err != nil {
		return err
	}
	resp, err := c.client.GetWheel(ctx, &wheelv1.GetWheelRequest{Id: id})
	if err != nil {
		return err
	}
	printWheel(resp.GetWheel())
	return nil
}

func (c cli) create(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	name := fs.String("name", "", "wheel name (required)")
	desc := fs.String("description", "", "wheel description")
	mode := fs.String("mode", "equal", "equal, weighted or inventory")
	var segs segmentFlags
	fs.Var(&segs, "s", `segment as "label[:weight[:stock]]", repeatable; empty stock means unlimited`)
	_ = fs.Parse(args)

	m, ok := map[string]wheelv1.WheelMode{
		"equal":     wheelv1.WheelMode_WHEEL_MODE_EQUAL,
		"weighted":  wheelv1.WheelMode_WHEEL_MODE_WEIGHTED,
		"inventory": wheelv1.WheelMode_WHEEL_MODE_INVENTORY,
	}[*mode]
	if !ok {
		return fmt.Errorf("unknown mode %q", *mode)
	}
	resp, err := c.client.CreateWheel(ctx, &wheelv1.CreateWheelRequest{
		Name:        *name,
		Description: *desc,
		Mode:        m,
		Segments:    segs,
	})
	if err != nil {
		return err
	}
	fmt.Printf("created %s\n\n", resp.GetWheel().GetId())
	printWheel(resp.GetWheel())
	return nil
}

func (c cli) spin(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("spin", flag.ExitOnError)
	n := fs.Int("n", 1, "number of spins")
	workers := fs.Int("c", 1, "concurrent workers")
	_ = fs.Parse(args)
	id, err := oneID("spin", fs.Args())
	if err != nil {
		return err
	}

	jobs := make(chan struct{})
	var (
		mu      sync.Mutex
		wins    = map[string]int{}
		stopErr error
		wg      sync.WaitGroup
	)
	for range max(*workers, 1) {
		wg.Go(func() {
			for range jobs {
				resp, err := c.client.SpinWheel(ctx, &wheelv1.SpinWheelRequest{Id: id})
				mu.Lock()
				if err != nil {
					if stopErr == nil {
						stopErr = err
					}
				} else {
					wins[resp.GetSegment().GetSpec().GetLabel()]++
					if *n == 1 {
						fmt.Printf("🎉 %s\n", resp.GetSegment().GetSpec().GetLabel())
					}
				}
				mu.Unlock()
			}
		})
	}
	for range *n {
		mu.Lock()
		stopped := stopErr != nil
		mu.Unlock()
		if stopped {
			break
		}
		jobs <- struct{}{}
	}
	close(jobs)
	wg.Wait()

	if *n > 1 {
		total := 0
		for _, v := range wins {
			total += v
		}
		fmt.Printf("%d spins\n", total)
		labels := slices.SortedFunc(maps.Keys(wins), func(a, b string) int { return wins[b] - wins[a] })
		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		for _, label := range labels {
			v := wins[label]
			fmt.Fprintf(tw, "  %s\t%d\t%.1f%%\n", label, v, 100*float64(v)/float64(max(total, 1)))
		}
		_ = tw.Flush()
	}
	var ce *connect.Error
	if errors.As(stopErr, &ce) && ce.Code() == connect.CodeFailedPrecondition {
		fmt.Printf("stopped: %s\n", ce.Message())
		return nil
	}
	return stopErr
}

func (c cli) history(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("history", flag.ExitOnError)
	limit := fs.Int("limit", 20, "number of spins")
	_ = fs.Parse(args)
	id, err := oneID("history", fs.Args())
	if err != nil {
		return err
	}
	resp, err := c.client.ListSpins(ctx, &wheelv1.ListSpinsRequest{Id: id, Limit: int32(*limit)})
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TIME\tUSER\tPRIZE")
	for _, s := range resp.GetSpins() {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", s.GetSpinTime().AsTime().Local().Format(time.DateTime), s.GetUserName(), s.GetSegmentLabel())
	}
	return tw.Flush()
}

func (c cli) setEnabled(ctx context.Context, args []string, enabled bool) error {
	id, err := oneID("pause/resume", args)
	if err != nil {
		return err
	}
	resp, err := c.client.SetWheelEnabled(ctx, &wheelv1.SetWheelEnabledRequest{Id: id, Enabled: enabled})
	if err != nil {
		return err
	}
	fmt.Printf("%s is %s\n", id, status(resp.GetWheel()))
	return nil
}

func (c cli) restock(ctx context.Context, args []string) error {
	id, err := oneID("restock", args)
	if err != nil {
		return err
	}
	resp, err := c.client.RestockWheel(ctx, &wheelv1.RestockWheelRequest{Id: id})
	if err != nil {
		return err
	}
	printWheel(resp.GetWheel())
	return nil
}

func (c cli) delete(ctx context.Context, args []string) error {
	id, err := oneID("delete", args)
	if err != nil {
		return err
	}
	if _, err := c.client.DeleteWheel(ctx, &wheelv1.DeleteWheelRequest{Id: id}); err != nil {
		return err
	}
	fmt.Printf("deleted %s\n", id)
	return nil
}

func printWheel(w *wheelv1.Wheel) {
	fmt.Printf("%s (%s)\n", w.GetName(), w.GetId())
	if w.GetDescription() != "" {
		fmt.Println(w.GetDescription())
	}
	fmt.Printf("mode: %s  spins: %d  status: %s  owner: %s\n\n", modeName(w.GetMode()), w.GetTotalSpins(), status(w), w.GetOwnerName())
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tLABEL\tWEIGHT\tREMAINING\tWINS\tCHANCE")
	for i, s := range w.GetSegments() {
		remaining := "∞"
		if s.Remaining != nil {
			remaining = fmt.Sprintf("%d/%d", s.GetRemaining(), s.GetSpec().GetStock())
		}
		fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%d\t%.1f%%\n", i, s.GetSpec().GetLabel(), s.GetSpec().GetWeight(), remaining, s.GetWins(), 100*s.GetProbability())
	}
	_ = tw.Flush()
}

func modeName(m wheelv1.WheelMode) string {
	return strings.ToLower(strings.TrimPrefix(m.String(), "WHEEL_MODE_"))
}

func status(w *wheelv1.Wheel) string {
	switch {
	case w.GetExhausted():
		return "exhausted"
	case !w.GetEnabled():
		return "paused"
	}
	return "live"
}

func oneID(cmd string, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%s: expected exactly one wheel id", cmd)
	}
	return args[0], nil
}

// segmentFlags parses repeated -s "label[:weight[:stock]]" flags.
type segmentFlags []*wheelv1.SegmentSpec

func (s *segmentFlags) String() string { return fmt.Sprint(len(*s), " segments") }

func (s *segmentFlags) Set(v string) error {
	parts := strings.Split(v, ":")
	if len(parts) > 3 {
		return fmt.Errorf("segment %q: want label[:weight[:stock]]", v)
	}
	seg := &wheelv1.SegmentSpec{Label: parts[0], Weight: 1}
	if len(parts) > 1 && parts[1] != "" {
		w, err := strconv.ParseInt(parts[1], 10, 32)
		if err != nil {
			return fmt.Errorf("segment %q: bad weight: %w", v, err)
		}
		seg.Weight = int32(w)
	}
	if len(parts) > 2 && parts[2] != "" {
		n, err := strconv.ParseInt(parts[2], 10, 32)
		if err != nil {
			return fmt.Errorf("segment %q: bad stock: %w", v, err)
		}
		seg.Stock = new(int32(n))
	}
	*s = append(*s, seg)
	return nil
}

func bearer(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	}
}

// emulatorSignIn signs in to the Firebase Auth emulator with a fixed dev
// password, creating the account if needed, and returns an ID token.
func emulatorSignIn(ctx context.Context, host, email string) (string, error) {
	const password = "wheelctl-dev"
	call := func(method string) (string, error) {
		url := fmt.Sprintf("http://%s/identitytoolkit.googleapis.com/v1/accounts:%s?key=demo-api-key", host, method)
		body, _ := json.Marshal(map[string]any{
			"email": email, "password": password, "returnSecureToken": true, "displayName": "wheelctl",
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		var out struct {
			IDToken string `json:"idToken"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return "", err
		}
		if out.IDToken == "" {
			return "", errors.New(out.Error.Message)
		}
		return out.IDToken, nil
	}
	token, err := call("signInWithPassword")
	if err != nil && strings.Contains(err.Error(), "EMAIL_NOT_FOUND") {
		token, err = call("signUp")
	}
	return token, err
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
