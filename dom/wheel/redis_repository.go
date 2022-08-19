package wheel

import (
    "context"
    "fmt"
    "log"
    "math/rand"
    "strings"
    "time"

    "github.com/go-redis/redis/v9"
)

const (
    wheelSegFmt         = "wheel:%s:segment:%d"
    missingLuaScriptErr = "NOSCRIPT No matching script. Please use EVAL."
)

type RedisRepository struct {
    Client *redis.Client

    SpinScript string
    spinSHA    string

    StateScript string
    stateSHA    string

    SegmentSize int
}

type RedisRepoParams struct {
    Host        string
    SpinScript  string
    StateScript string
    SegmentSize int
}

func NewRedisRepository(ctx context.Context, params *RedisRepoParams) (*RedisRepository, error) {
    client := redis.NewClient(&redis.Options{
        Addr:    params.Host,
        Network: "tcp",
    })

    repo := &RedisRepository{
        Client:      client,
        SpinScript:  params.SpinScript,
        StateScript: params.StateScript,
        SegmentSize: params.SegmentSize,
    }

    err := repo.LoadScripts(ctx)
    if err != nil {
        return nil, err
    }

    return repo, nil
}

func (repo *RedisRepository) LoadScripts(ctx context.Context) error {
    spinSha, err := repo.Client.ScriptLoad(ctx, repo.SpinScript).Result()
    if err != nil {
        return fmt.Errorf("failed to load spin script into redis err:%w", err)
    }
    repo.spinSHA = spinSha

    stateSha, err := repo.Client.ScriptLoad(ctx, repo.StateScript).Result()
    if err != nil {
        return fmt.Errorf("failed to load script state script into redis err:%w", err)
    }
    repo.stateSHA = stateSha

    return nil
}

func (repo *RedisRepository) Spin(
    ctx context.Context,
    name string,
) (
    i int64,
    remainingPrizes int64,
    err error,
) {
    rand.Seed(time.Now().UTC().UnixNano())

    state, err := repo.Status(ctx, name)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to fetch current state of wheel err:%w", err)
    }

    if !state.Enabled {
        log.Printf("wheel is not enabled :%s \n", name)
        return 0, 0, NotEnabled
    }

    randomValue := rand.Float32()
    values, err := repo.Client.EvalSha(
        ctx,
        repo.spinSHA,
        []string{name},
        randomValue,
        repo.SegmentSize,
    ).Result()
    if err != nil {
        if strings.Contains(err.Error(), missingLuaScriptErr) {
            if err = repo.LoadScripts(ctx); err != nil {
                log.Printf("wheel state script is not in redis")

                return repo.Spin(ctx, name)
            }
        }
    }

    results := values.([]interface{})
    if len(results) < 2 {
        err = fmt.Errorf(
            "invalid response from redis upon determining winning segment wheel:%q",
            name,
        )
        return
    }

    if len(results) == 3 {
        log.Printf("wheel of fortune name:%s has returned more values than expected")
    }

    i = results[0].(int64) - 1
    remainingPrizes = results[1].(int64)

    if remainingPrizes == 0 {
        return 0, 0, OutOfPrizes
    }

    return
}

func (repo *RedisRepository) Create(
    ctx context.Context,
    name string,
    segments []int64,
) (
    err error,
) {
    tx := repo.Client.TxPipeline()

    for i, segment := range segments {
        key := fmt.Sprintf(wheelSegFmt, name, i+1)
        tx.IncrBy(ctx, key, segment)
    }

    _, err = tx.Set(ctx, fmt.Sprintf("wheel:%s:enabled", name), 1, 0).Result()
    if err != nil {
        return fmt.Errorf("failed to set wheel as enabled err:%w", err)
    }

    results, err := tx.Exec(ctx)
    if err != nil {
        err = fmt.Errorf("failed to create wheel:%s", err)
        return
    }

    errCount := 0
    for _, resp := range results {
        if resp.Err() == nil {
            continue
        }

        log.Print(resp.Err())
        errCount++
    }

    if errCount == 0 {
        return
    }

    return fmt.Errorf("%d errors occured when creating wheel:%q", errCount, name)
}

func (repo *RedisRepository) Status(ctx context.Context, name string) (*State, error) {
    val, err := repo.Client.EvalSha(
        ctx,
        repo.stateSHA,
        []string{name},
        repo.SegmentSize,
    ).Result()
    if err != nil {
        if strings.Contains(err.Error(), missingLuaScriptErr) {
            if err = repo.LoadScripts(ctx); err != nil {
                log.Printf("wheel state script is not in redis")

                return nil, fmt.Errorf("failed to init wheel state script when fetching wheel state err:%w", err)
            }
            return repo.Status(ctx, name)
        }
    }

    results := val.([]interface{})

    spins := results[0].(int64)
    prizes := results[1].([]interface{})
    enabled := results[2].(int64) == 1
    prizeTotals := []int64{}
    for _, amount := range prizes {
        prizeTotals = append(prizeTotals, amount.(int64))
    }

    return &State{
        Spins:   spins,
        Prizes:  prizeTotals,
        Enabled: enabled,
    }, nil
}

func (repo *RedisRepository) GetAllWheelNames(ctx context.Context) ([]string, error) {
    keys, err := repo.Client.Keys(ctx, "wheel:*:enabled").Result()
    if err != nil {
        return nil, err
    }
    var names []string
    for _, key := range keys {
        names = append(names, strings.Split(key, ":")[1])
    }
    return names, nil
}
