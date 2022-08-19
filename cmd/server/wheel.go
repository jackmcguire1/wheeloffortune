package main

import (
    "context"
    "errors"
    "log"

    "wheeloffortune/api"
    "wheeloffortune/dom/wheel"
)

func (svc Server) CreateWheel(ctx context.Context, req *api.CreateWheelReq) (resp *api.CreateWheelResp, err error) {
    if err = svc.wheelSvc.Create(ctx, req.Name, req.Segments); err != nil {
        log.Printf("failed to get wheel status for %s err:%w \n", req.Name, err)
        return
    }
    resp = &api.CreateWheelResp{Message: "success"}
    log.Printf("created wheel %s \n", req.Name)
    return
}

func (svc Server) SpinWheel(ctx context.Context, req *api.SpinWheelReq) (res *api.SpinWheelResp, err error) {
    log.Printf("spinning wheel for %s", req.Name)
    winningIndex, remainingPrizes, err := svc.wheelSvc.Spin(ctx, req.Name)
    if errors.Is(err, wheel.OutOfPrizes) {
        res = &api.SpinWheelResp{
            RemainingPrizes:     0,
            WinningSegmentIndex: 0,
            Error:               "Out Of Prizes",
        }
        return
    }
    if err != nil {
        log.Printf("failed to get wheel status for %s err:%w \n", req.Name, err)
        return
    }
    res = &api.SpinWheelResp{
        RemainingPrizes:     remainingPrizes,
        WinningSegmentIndex: winningIndex,
    }
    log.Printf("spun wheel for %s winning-index:%d total-prizes-left:%d", req.Name, winningIndex, remainingPrizes)

    return
}

func (svc Server) GetWheelStatus(ctx context.Context, req *api.WheelStatusReq) (res *api.WheelStatusResp, err error) {
    state, err := svc.wheelSvc.Status(ctx, req.Name)
    if err != nil {
        log.Printf("failed to get wheel status for %s err:%w \n", req.Name, err)
        return
    }
    res = &api.WheelStatusResp{
        Spins:   state.Spins,
        Prizes:  state.Prizes,
        Enabled: state.Enabled,
    }
    return
}

func (svc Server) GetAllWheelNames(ctx context.Context, req *api.GetAllWheelnamesReq) (res *api.GetAllWheelnamesResp, err error) {
    names, err := svc.wheelSvc.GetAllWheelNames(ctx)
    if err != nil {
        log.Printf("failed to get wheel names %w \n", err)
        return
    }
    res = &api.GetAllWheelnamesResp{
        Wheels: names,
    }

    return
}
