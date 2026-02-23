package grpcx

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/bcrosbie/modeloman/internal/domain"
	"github.com/bcrosbie/modeloman/internal/rpccontract"
	"github.com/bcrosbie/modeloman/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type principalContextKey struct{}

type TokenBucketRateLimiterConfig struct {
	AuthenticatedPerSecond   float64
	AuthenticatedBurst       float64
	UnauthenticatedPerSecond float64
	UnauthenticatedBurst     float64
	BucketTTL                time.Duration
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type TokenBucketRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	config  TokenBucketRateLimiterConfig
}

func NewTokenBucketRateLimiter(config TokenBucketRateLimiterConfig) *TokenBucketRateLimiter {
	if config.AuthenticatedPerSecond <= 0 {
		config.AuthenticatedPerSecond = 20
	}
	if config.AuthenticatedBurst <= 0 {
		config.AuthenticatedBurst = 60
	}
	if config.UnauthenticatedPerSecond <= 0 {
		config.UnauthenticatedPerSecond = 5
	}
	if config.UnauthenticatedBurst <= 0 {
		config.UnauthenticatedBurst = 20
	}
	if config.BucketTTL <= 0 {
		config.BucketTTL = 10 * time.Minute
	}

	return &TokenBucketRateLimiter{
		buckets: map[string]*tokenBucket{},
		config:  config,
	}
}

func (l *TokenBucketRateLimiter) Allow(ctx context.Context) bool {
	if l == nil {
		return true
	}
	identifier, authenticated := limitIdentifier(ctx)
	rate := l.config.UnauthenticatedPerSecond
	burst := l.config.UnauthenticatedBurst
	if authenticated {
		rate = l.config.AuthenticatedPerSecond
		burst = l.config.AuthenticatedBurst
	}

	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.evictExpiredBuckets(now)

	bucket, ok := l.buckets[identifier]
	if !ok {
		l.buckets[identifier] = &tokenBucket{
			tokens:     burst - 1,
			lastRefill: now,
			lastSeen:   now,
		}
		return true
	}

	elapsedSeconds := now.Sub(bucket.lastRefill).Seconds()
	if elapsedSeconds > 0 {
		bucket.tokens = minFloat64(burst, bucket.tokens+(elapsedSeconds*rate))
		bucket.lastRefill = now
	}
	bucket.lastSeen = now
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

func (l *TokenBucketRateLimiter) evictExpiredBuckets(now time.Time) {
	for key, bucket := range l.buckets {
		if now.Sub(bucket.lastSeen) > l.config.BucketTTL {
			delete(l.buckets, key)
		}
	}
}

func RateLimitUnaryInterceptor(limiter *TokenBucketRateLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if limiter == nil || limiter.Allow(ctx) {
			return handler(ctx, req)
		}
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
}

func IdempotencyUnaryInterceptor(idStore store.IdempotencyStore) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if idStore == nil {
			return handler(ctx, req)
		}
		if _, isWriteMethod := rpccontract.WriteMethods[info.FullMethod]; !isWriteMethod {
			return handler(ctx, req)
		}

		idempotencyKey := extractIdempotencyKey(ctx, req)
		if idempotencyKey == "" {
			return handler(ctx, req)
		}

		requestHash, err := idempotencyRequestHash(req)
		if err != nil {
			return nil, err
		}
		record, created, err := idStore.ReserveIdempotencyKey(info.FullMethod, idempotencyKey, requestHash)
		if err != nil {
			return nil, err
		}
		if !created {
			if record.RequestHash != requestHash {
				return nil, domain.Conflict("idempotency_key has already been used with a different request payload")
			}
			if !record.Completed {
				return nil, domain.FailedPrecondition("idempotency key is already in progress")
			}
			decodedResponse, err := decodeIdempotentResponse(record.ResponseJSON)
			if err != nil {
				return nil, err
			}
			return decodedResponse, nil
		}

		response, err := handler(ctx, req)
		if err != nil {
			_ = idStore.ReleaseIdempotencyKey(info.FullMethod, idempotencyKey)
			return nil, err
		}
		encodedResponse, err := encodeIdempotentResponse(response)
		if err != nil {
			_ = idStore.ReleaseIdempotencyKey(info.FullMethod, idempotencyKey)
			return nil, err
		}
		if err := idStore.CompleteIdempotencyKey(info.FullMethod, idempotencyKey, encodedResponse); err != nil {
			return nil, err
		}
		return response, nil
	}
}

func RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (response any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered method=%s panic=%v\n%s", info.FullMethod, recovered, string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func AuthUnaryInterceptor(token string, allowLegacyToken bool, keyAuth store.AgentKeyAuthenticator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !rpccontract.RequiresAuthentication(info.FullMethod) {
			return handler(ctx, req)
		}

		requestToken := extractToken(ctx)
		if requestToken == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authentication token")
		}

		var principal store.AgentPrincipal
		authenticated := false

		if keyAuth != nil {
			authenticatedPrincipal, ok, err := keyAuth.AuthenticateAgentKey(requestToken)
			if err != nil {
				log.Printf("auth validation failure method=%s err=%v", info.FullMethod, err)
				return nil, status.Error(codes.Internal, "authentication subsystem unavailable")
			}
			if ok {
				principal = authenticatedPrincipal
				authenticated = true
			}
		}

		if !authenticated && allowLegacyToken && token != "" && legacyTokenMatch(requestToken, token) {
			principal = store.AgentPrincipal{
				AgentID: "legacy-shared-token",
				KeyID:   "legacy_shared_token",
				Scopes:  append([]string(nil), rpccontract.DefaultAgentKeyScopes...),
			}
			authenticated = true
		}

		if !authenticated {
			return nil, status.Error(codes.Unauthenticated, "invalid authentication token")
		}

		if requiredScope, hasRequiredScope := rpccontract.RequiredScope(info.FullMethod); hasRequiredScope && !hasScope(principal.Scopes, requiredScope) {
			return nil, status.Error(codes.PermissionDenied, "api key scope does not allow this method")
		}
		log.Printf("authenticated method=%s agent_id=%s key_id=%s", info.FullMethod, principal.AgentID, principal.KeyID)
		return handler(withPrincipal(ctx, principal), req)
	}
}

func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		started := time.Now()
		response, err := handler(ctx, req)
		log.Printf("grpc method=%s duration=%s code=%s", info.FullMethod, time.Since(started), status.Code(err))
		return response, err
	}
}

func ErrorUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		response, err := handler(ctx, req)
		if err == nil {
			return response, nil
		}

		if status.Code(err) != codes.Unknown {
			return nil, err
		}

		return nil, mapError(err)
	}
}

func mapError(err error) error {
	var appError *domain.AppError
	if errors.As(err, &appError) {
		switch appError.Code {
		case domain.CodeInvalidArgument:
			return status.Error(codes.InvalidArgument, appError.Message)
		case domain.CodeNotFound:
			return status.Error(codes.NotFound, appError.Message)
		case domain.CodeConflict:
			return status.Error(codes.AlreadyExists, appError.Message)
		case domain.CodeUnauthenticated:
			return status.Error(codes.Unauthenticated, appError.Message)
		case domain.CodeFailedPrecondition:
			return status.Error(codes.FailedPrecondition, appError.Message)
		case domain.CodeResourceExhausted:
			return status.Error(codes.ResourceExhausted, appError.Message)
		default:
			return status.Error(codes.Internal, appError.Message)
		}
	}

	return status.Error(codes.Internal, "internal server error")
}

func extractToken(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	token := strings.TrimSpace(first(md.Get("x-modeloman-token")))
	if token != "" {
		return token
	}

	authHeader := strings.TrimSpace(first(md.Get("authorization")))
	const bearer = "Bearer "
	if strings.HasPrefix(authHeader, bearer) {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, bearer))
	}
	return ""
}

func first(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if strings.TrimSpace(scope) == required {
			return true
		}
	}
	return false
}

func legacyTokenMatch(requestToken, expectedToken string) bool {
	requestHash := sha256.Sum256([]byte(requestToken))
	expectedHash := sha256.Sum256([]byte(expectedToken))
	return subtle.ConstantTimeCompare(requestHash[:], expectedHash[:]) == 1
}

func extractIdempotencyKey(ctx context.Context, req any) string {
	requestStruct, ok := req.(*structpb.Struct)
	if ok {
		requestMap := requestStruct.AsMap()
		if raw, exists := requestMap["idempotency_key"]; exists {
			key, _ := raw.(string)
			key = strings.TrimSpace(key)
			if key != "" {
				return key
			}
		}
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(first(md.Get("x-idempotency-key")))
}

func idempotencyRequestHash(req any) (string, error) {
	requestStruct, ok := req.(*structpb.Struct)
	if !ok {
		return "", domain.InvalidArgument("idempotency is only supported for object request payloads")
	}

	sanitized := map[string]any{}
	for key, value := range requestStruct.AsMap() {
		if key == "idempotency_key" {
			continue
		}
		sanitized[key] = value
	}
	serialized, err := json.Marshal(sanitized)
	if err != nil {
		return "", domain.Internal("failed to encode idempotency request payload", err)
	}
	hash := sha256.Sum256(serialized)
	return hex.EncodeToString(hash[:]), nil
}

func encodeIdempotentResponse(response any) (string, error) {
	responseStruct, ok := response.(*structpb.Struct)
	if !ok {
		return "", domain.Internal("idempotency requires object responses for write RPCs", nil)
	}
	serialized, err := json.Marshal(responseStruct.AsMap())
	if err != nil {
		return "", domain.Internal("failed to encode idempotent response", err)
	}
	return string(serialized), nil
}

func decodeIdempotentResponse(responseJSON string) (*structpb.Struct, error) {
	responseJSON = strings.TrimSpace(responseJSON)
	if responseJSON == "" {
		return nil, domain.Internal("stored idempotent response is empty", nil)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(responseJSON), &decoded); err != nil {
		return nil, domain.Internal("failed to decode stored idempotent response", err)
	}
	response, err := structpb.NewStruct(decoded)
	if err != nil {
		return nil, domain.Internal("stored idempotent response payload is invalid", err)
	}
	return response, nil
}

func withPrincipal(ctx context.Context, principal store.AgentPrincipal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func principalFromContext(ctx context.Context) (store.AgentPrincipal, bool) {
	value := ctx.Value(principalContextKey{})
	principal, ok := value.(store.AgentPrincipal)
	return principal, ok
}

func limitIdentifier(ctx context.Context) (string, bool) {
	if principal, ok := principalFromContext(ctx); ok && strings.TrimSpace(principal.KeyID) != "" {
		return "key:" + principal.KeyID, true
	}
	return "ip:" + remoteIP(ctx), false
}

func remoteIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "unknown"
	}
	address := strings.TrimSpace(p.Addr.String())
	host, _, err := net.SplitHostPort(address)
	if err == nil && host != "" {
		return host
	}
	if address == "" {
		return "unknown"
	}
	return address
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
