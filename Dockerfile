# Created with Strapit
FROM    golang:1.18-buster as builder
WORKDIR /app
COPY    . ./
RUN     go build -ldflags="-w -s" -o service ./cmd/gcs-proxy

FROM    debian:buster-slim
RUN     set -x && \
		apt-get update && \
		DEBIAN_FRONTEND=noninteractive apt-get install -y \
			ca-certificates && \
			rm -rf /var/lib/apt/lists/*
COPY    --from=builder /app/service /app/service

CMD     ["/app/service"]
