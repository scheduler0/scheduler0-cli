FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /bin/scheduler0 .

FROM alpine:3.22

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /bin/scheduler0 /usr/local/bin/scheduler0

ENTRYPOINT ["scheduler0"]
