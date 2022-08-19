package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "strconv"
    "strings"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "wheeloffortune/api"
)

var (
    addr        = flag.String("addr", "localhost:7755", "the address to connect to")
    wheelName   = flag.String("wheel-name", "test", "the name of the wheel of fortune")
    prizesStr   = flag.String("segment-prize-allocation", "10,10,10,10,10", "Wheel of fortune segment prize(s), split by ',' delimeter")
    segmentSize = flag.Int("wheel-segment-size", 5, "number of segments for a wheel")
    override    = flag.Bool("wheel-override", true, "override wheel creation")
)

func main() {
    flag.Parse()

    var segments []int64

    prizes := strings.Split(*prizesStr, ",")
    if len(prizes) > 1 {
        for i := 0; i < len(prizes); i++ {
            v, _ := strconv.ParseInt(prizes[i], 10, 64)
            segments = append(segments, v)
        }
    }
    // Set up a connection to the server.
    conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v \n", err)
    }
    defer conn.Close()

    c := api.NewWheelClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    log.Printf("fetching all avaliable wheels %s \n", *wheelName)
    wheelNameResp, err := c.GetAllWheelNames(ctx, &api.GetAllWheelnamesReq{})
    if err != nil {
        log.Fatalf("failed to fetch all wheel names err:%w", err)
    }
    fmt.Printf("found wheels %v \n ", strings.Join(wheelNameResp.Wheels, ","))

    for _, wName := range wheelNameResp.Wheels {
        if *wheelName == wName && *override == false {
            log.Fatalf("Wheel %s has already been created.", *wheelName)
            return
        }
    }

    resp, err := c.CreateWheel(
        ctx,
        &api.CreateWheelReq{
            Name:     *wheelName,
            Segments: segments,
        },
    )
    if err != nil {
        log.Fatalf("failed to create wheel %s err:%w", *wheelName, err)
    }
    if resp.Message != "success" {
        log.Fatalf("failed to create wheel %s server-msg:%q", *wheelName, resp.Message)
    }

    log.Printf("created wheel %s", *wheelName)

    return
}
