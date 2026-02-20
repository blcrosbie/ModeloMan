# Error Handling

## Domain Error Type
All business logic returns `*domain.AppError` where possible:
- `invalid_argument`
- `not_found`
- `conflict`
- `unauthenticated`
- `internal`

## Mapping to gRPC Codes
- `invalid_argument` -> `InvalidArgument`
- `not_found` -> `NotFound`
- `conflict` -> `AlreadyExists`
- `unauthenticated` -> `Unauthenticated`
- `internal` -> `Internal`

Any non-domain error is treated as `Internal`.

## Interceptor Order
Configured order:
1. panic recovery
2. auth
3. logging
4. error mapping

This ensures:
- panics never leak stack traces to clients
- auth guard applies before writes
- logs record final mapped gRPC status

## Authentication Behavior
If `AUTH_TOKEN` is unset:
- no method requires auth

If `AUTH_TOKEN` is set:
- write methods require metadata token
- read methods are open

Write methods are centrally defined in `internal/rpccontract/methods.go`.
