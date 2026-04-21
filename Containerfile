FROM caddy:2.11-builder AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN xcaddy build --with github.com/sn0wcrack/caddy-file-ip=/build

FROM caddy:2.11

COPY --from=builder /build/caddy /usr/bin/caddy

EXPOSE 80 443 2019

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
