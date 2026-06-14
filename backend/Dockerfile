FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata build-base

WORKDIR /app

COPY go.mod go.sum* ./
RUN if [ -f go.sum ]; then go mod download; else go mod tidy && go mod download; fi

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build \
    -ldflags "-s -w -X main.buildStamp=$(date -u '+%Y-%m-%d_%I:%M:%S%p') -X main.gitHash=$(git rev-parse HEAD 2>/dev/null || echo 'dev')" \
    -tags "netgo" \
    -o /app/server \
    ./cmd/server/main.go

FROM scratch AS final

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

WORKDIR /app

COPY --from=builder /app/server /app/server
COPY --from=builder /app/config/model_params.yaml /app/config/model_params.yaml
COPY --from=builder /app/.env.example /app/.env

USER 1000:1000

EXPOSE 8080 6060 9090

ENV TZ=Asia/Shanghai

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/server", "--healthcheck"] || exit 1

ENTRYPOINT ["/app/server"]
