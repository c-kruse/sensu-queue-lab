FROM golang:1.21-alpine as build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
RUN go build -o /bin/nsqload ./cmd/nsqload \
    && go build -o /bin/nsqsub ./cmd/nsqsub

FROM nsqio/nsq as run

COPY --from=build /bin/nsqload /bin/nsqload
COPY --from=build /bin/nsqsub /bin/nsqsub
