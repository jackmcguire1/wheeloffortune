package main

import (
    "context"
    _ "embed"
    "flag"
    "fmt"
    "log"
    "net"

    "wheeloffortune/dom/wheel"

    "google.golang.org/grpc"

    "wheeloffortune/api"
)

type Server struct {
    api.UnimplementedWheelServer
    wheelSvc wheel.Service
}

var (
    //go:embed spin.lua
    SpinScript []byte

    //go:embed state.lua
    StateScript []byte

    port        = flag.Int("port", 7755, "port of network")
    redisHost   = flag.String("redis-host", "localhost:6379", "redis host")
    segmentSize = flag.Int("wheel-segment-size", 5, "number of segements for a wheel")
)

func init() {
    flag.Parse()
}

func main() {
    ctx := context.Background()

    redisWheelRepo, err := wheel.NewRedisRepository(ctx, &wheel.RedisRepoParams{
        Host:        *redisHost,
        SpinScript:  string(SpinScript),
        StateScript: string(StateScript),
        SegmentSize: *segmentSize,
    })
    if err != nil {
        log.Fatal(fmt.Errorf("failed to setup wheel redis repository err:%w", err))
    }
    wheelSvc := wheel.New(&wheel.Resources{
        SegmentSize: *segmentSize,
        Repo:        redisWheelRepo,
    })

    svc := Server{
        UnimplementedWheelServer: api.UnimplementedWheelServer{},
        wheelSvc:                 wheelSvc,
    }

    listen, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    s := grpc.NewServer()

    api.RegisterWheelServer(s, svc)

    fmt.Printf("server listening: %d \n", *port)
    if err := s.Serve(listen); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
