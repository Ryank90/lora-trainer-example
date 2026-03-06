FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /lora-trainer ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /lora-trainer /usr/local/bin/lora-trainer
COPY configs/config.example.yaml /etc/lora-trainer/config.yaml

EXPOSE 8080
ENTRYPOINT ["lora-trainer"]
CMD ["-config", "/etc/lora-trainer/config.yaml"]
