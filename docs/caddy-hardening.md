# Caddy gRPC Hardening

This deployment uses a dedicated `grpc.modeloman.com` vhost for gRPC control-plane traffic.

## Required behavior

- TLS 1.2+ only.
- ALPN negotiated as `h2` on the gRPC hostname.
- Only gRPC traffic proxied on `grpc.modeloman.com`.
- Non-gRPC requests to `grpc.modeloman.com` return `404`.
- Plain HTTP to `grpc.modeloman.com` returns `404`.

## Config location

- `Caddyfile`

## Verification

1. Confirm ALPN is `h2`:
```bash
openssl s_client -alpn h2 -connect grpc.modeloman.com:443 -servername grpc.modeloman.com < /dev/null 2>/dev/null | grep -i "ALPN protocol"
```

Expected output includes `ALPN protocol: h2`.

2. Confirm non-gRPC traffic is rejected:
```bash
curl -i https://grpc.modeloman.com/
```

Expected status: `404`.

3. Confirm plaintext HTTP is denied:
```bash
curl -i http://grpc.modeloman.com/
```

Expected status: `404`.

4. Confirm reflection is unavailable in prod:
```bash
grpcurl grpc.modeloman.com:443 list
```

Expected result: failure unless reflection is intentionally enabled in a non-production environment.
