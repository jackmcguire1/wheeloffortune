# REDIS - Wheel Of Fortune

A simualted Wheel Of Fortune backed by Redis exposed by a GRCP Server & Client.

## How it works

This project simulates a Wheel Of Fortune by making use of Redis 
to maintain the segments of a wheel i.e. prize allocation and distribution

The server makes use of Redis LUA scripts to synchronously execute user
spins and exposes an API via GRPC to maintain and spin and wheels of fortune!

### How the data is stored:
Segments and Operational Flags of a wheel of fortune are stored in
Redis Keys.

#### Wheel Of Fortune Creation

- When a wheel is CREATED the following keys are generated:

##### wheel of fortune segment(s)
For each defined segment within a wheel of fortune with an allocated prize
a REDIS key like ```'wheel:{wheelname}:segment:{index}'```, data is stored like ```INCRBY {KEY} {VALUE}``` in a TX

#### wheel of fortune Enabled Status
To set the enable status of a wheel of fortune
a REDIS key like ```'wheel:{wheelname}:enabled'```, data is stored like ```SET {KEY} {VALUE} 1```

#### wheel of fortune Total Spins
to init the total spins of a wheel of fortune a
a REDIS key like ```'wheel:{wheelname}:spins'```, data is stored like ```INCRBY {KEY} 0```

#### Wheel Of Fortune Spin

- When a wheel is SPUN the LUA script in ```cmd/server/spin.lua``` is executed:

##### wheel of fortune segment(s)
For each simulated spin of a wheel of fortune, the segment the user lands on is deducted by
a REDIS key like ```'wheel:{wheelname}:segment:{index}'```, data is stored like ```DECR {KEY}```

Example Redis structure:

Refer to [this example](https://github.com/redis-developer/basic-analytics-dashboard-redis-bitmaps-nodejs#how-the-data-is-stored) for a more detailed example of what you need for this section.

### How the data is accessed:
Redis keys are accessed via defined LUA scripts to retrieve and manipulate
relevant keys defined by parameters.

#### Wheel Of Fortune State

- To fetch the current state and prize allocation of a wheel of fortune the
  following LUA script is executed ```cmd/server/state.lua```

##### wheel of fortune segment(s)
For each defined segment within a wheel of fortune we get the state of prize allocation by
a REDIS key like ```'wheel:{wheelname}:segment:{index}'```, data is FETCHED like ```INCRBY {KEY} 0```

##### wheel of fortune total spins
to get the total spins of a wheel of fortune
a REDIS key like ```'wheel:{wheelname}:spins'```, data is FETCHED like ```INCRBY {KEY} 0```


##### wheel of fortune enabled status
to get the active state of a wheel of fortune
a REDIS key like ```'wheel:{wheelname}:enabled'```, data is FETCHED like ```INCRBY {KEY} 0```

#### Wheel Of Fortune Spin

- When a wheel is SPUN the LUA script in ```cmd/server/spin.lua``` is executed:

##### wheel of fortune segment(s)
calculate each segement prize allocation from individual redis keys and accumulate results
in lua.
a REDIS key like ```'wheel:{wheelname}:segment:{index}'```, data is accessed like ```INCRBY {KEY} 0```
.

## How to run it locally?

### GO CLI Server, Client And Admin
```shell
go run wheeloffortune/cmd/server
```
> Observe logs that the server is listening

---
```shell
go run wheeloffortune/cmd/admin
```
> Observe log that admin has created a wheel
---

```shell
go run wheeloffortune/cmd/client
````
> Observe logs that the client is spinning the wheel until all prizes allocated

### VueJS

- make sure 'test' wheel is setup
- run ```docker-compose up -d envoy```
- ```cd ./vue-wheel-forked```
- ```yarn install```
- ```yarn serve ```
> vueJS application should run on port 8081

### Prerequisites
- Go 1.18+
- Docker
- Protoc
- vue-cli
- node 16+


### Local installation

```shell
> go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

> brew install protobuf

>  protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/api.proto
```

## Deployment

This project utilises Redis-Stack in a docker container
> docker-compose up -d redis

Run ENVOY to proxy http -> grpc requests
> docker-compose up -d envoy
