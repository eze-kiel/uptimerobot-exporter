FROM devopsworks/golang-upx:1.16 AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build \
    -o uptimerobot-exporter . && \
    strip uptimerobot-exporter && \
    /usr/local/bin/upx -9 uptimerobot-exporter

FROM gcr.io/distroless/base-debian10

WORKDIR /app

COPY --from=builder /build/uptimerobot-exporter .

EXPOSE 2112

ENTRYPOINT [ "/app/uptimerobot-exporter" ]