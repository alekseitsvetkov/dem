FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/poller ./cmd/poller/ \
 && go build -o /bin/downloader ./cmd/downloader/ \
 && go build -o /bin/parser ./cmd/parser/ \
 && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.2

FROM alpine:3.21
# 7zip for RAR extraction (available in main repo, unlike unrar)
# cloudscraper for Cloudflare bypass
RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories \
 && apk add --no-cache p7zip python3 py3-pip \
 && pip3 install --break-system-packages cloudscraper \
 && ln -sf /usr/bin/7za /usr/local/bin/7z

COPY --from=builder /bin/poller /bin/downloader /bin/parser /go/bin/migrate /usr/local/bin/
COPY sql/migrations /sql/migrations
