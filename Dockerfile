# syntax=docker/dockerfile:1
FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/wheelctl ./cmd/wheelctl

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/ /usr/local/bin/
EXPOSE 7755
ENTRYPOINT ["/usr/local/bin/server"]
