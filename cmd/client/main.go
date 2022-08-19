package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "math/rand"
    "strings"
    "sync"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "wheeloffortune/api"
)

var (
    addr      = flag.String("addr", "localhost:7755", "the address to connect to")
    wheelName = flag.String("wheel-name", "test", "the name of the wheel of fortune")
    workers   = flag.Int("num-of-workers", 2, "the number of clients")
)

func main() {
    flag.Parse()

    conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v \n", err)
    }
    defer conn.Close()
    c := api.NewWheelClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

runWorkers:
    wg := sync.WaitGroup{}
    wg.Add(*workers)

    wheelNameResp, err := c.GetAllWheelNames(ctx, &api.GetAllWheelnamesReq{})
    if err != nil {
        log.Printf("failed to fetch all wheel names err:%w \n", err)
        return
    }
    fmt.Printf("found wheels %v \n ", strings.Join(wheelNameResp.Wheels, ","))

    var valid bool
    for _, wName := range wheelNameResp.Wheels {
        if *wheelName == wName {
            valid = true
            break
        }
    }

    if !valid {
        log.Fatalf("wheel %s has not been setup", *wheelName)
    }

    var finish bool
    for i := 0; i < *workers; i++ {

        go func() {
            defer wg.Done()

            // random sleep between workers
            time.Sleep(time.Duration(rand.Int31n(60)) * time.Millisecond)

            // Set up a connection to the server.
            conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
            if err != nil {
                log.Fatalf("did not connect: %v \n", err)
            }
            defer conn.Close()
            c := api.NewWheelClient(conn)

            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
            defer cancel()

            log.Printf("fetching wheel status for %s \n", *wheelName)
            state, err := c.GetWheelStatus(ctx, &api.WheelStatusReq{Name: *wheelName})
            if err != nil {
                log.Fatal(fmt.Errorf("failed to get wheel state for %s err:%w", *wheelName, err))
            }
            fmt.Printf(
                "got wheel state for %s total-spins:%d prizes-remaining:%+v \n",
                *wheelName,
                state.Spins,
                state.Prizes,
            )

            outOfPrizes := true
            for _, prizes := range state.Prizes {
                if prizes != 0 {
                    outOfPrizes = false
                    break
                }
            }

            if outOfPrizes || !state.Enabled {
                fmt.Printf("wheel %q is out of prizes or not enabled \n", *wheelName)
                finish = true
                return
            }

            fmt.Printf("spinning wheel %s", *wheelName)
            res, err := c.SpinWheel(ctx, &api.SpinWheelReq{Name: *wheelName})
            if err != nil {
                if strings.Contains(err.Error(), "out of prizes") {
                    finish = true
                    fmt.Printf("wheel %q is out of prizes \n", *wheelName)
                    return
                }
                if strings.Contains(err.Error(), "wheel not enabled") {
                    finish = true
                    fmt.Printf("wheel %q is not enabled \n", *wheelName)
                    return
                }
                log.Fatal(fmt.Errorf("failed to get wheel spin result for %s err:%w \n", *wheelName, err))
            }
            fmt.Printf(
                "got prize for segment %d - total remaining prizes:%d \n",
                res.WinningSegmentIndex,
                res.RemainingPrizes,
            )
        }()

    }
    wg.Wait()
    if !finish {
        goto runWorkers
    }

    return
}
