package grpcx

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
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

type staticKeyAuth struct {
	principal store.AgentPrincipal
	ok        bool
	err       error
}

func (s staticKeyAuth) AuthenticateAgentKey(rawKey string) (store.AgentPrincipal, bool, error) {
	return s.principal, s.ok, s.err
}

func (s staticKeyAuth) EnsureAgentKey(agentID, rawKey string) (string, bool, error) {
	return "", false, nil
}

type fakeIdempotencyStore struct {
	mu      sync.Mutex
	records map[string]store.IdempotencyRecord
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{
		records: map[string]store.IdempotencyRecord{},
	}
}

func (s *fakeIdempotencyStore) ReserveIdempotencyKey(method, idempotencyKey, requestHash string) (store.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := method + "::" + idempotencyKey
	if record, ok := s.records[key]; ok {
		return record, false, nil
	}
	s.records[key] = store.IdempotencyRecord{
		RequestHash: strings.TrimSpace(requestHash),
		Completed:   false,
	}
	return store.IdempotencyRecord{}, true, nil
}

func (s *fakeIdempotencyStore) CompleteIdempotencyKey(method, idempotencyKey, responseJSON string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := method + "::" + idempotencyKey
	record, ok := s.records[key]
	if !ok {
		return domain.NotFound("idempotency key not found")
	}
	record.ResponseJSON = responseJSON
	record.Completed = true
	s.records[key] = record
	return nil
}

func (s *fakeIdempotencyStore) ReleaseIdempotencyKey(method, idempotencyKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := method + "::" + idempotencyKey
	record, ok := s.records[key]
	if !ok || record.Completed {
		return nil
	}
	delete(s.records, key)
	return nil
}

func mustStruct(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()
	out, err := structpb.NewStruct(value)
	if err != nil {
		t.Fatalf("failed to build struct: %v", err)
	}
	return out
}

func TestAuthInterceptorRejectsPrivateReadWithoutToken(t *testing.T) {
	interceptor := AuthUnaryInterceptor("", false, nil)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodExportState,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %s", status.Code(err))
	}
}

func TestAuthInterceptorRejectsMissingScope(t *testing.T) {
	keyAuth := staticKeyAuth{
		principal: store.AgentPrincipal{
			AgentID: "a1",
			KeyID:   "k1",
			Scopes:  []string{rpccontract.ScopeTasksWrite},
		},
		ok: true,
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-modeloman-token", "agent-key"))
	interceptor := AuthUnaryInterceptor("", false, keyAuth)
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodSetPolicy,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %s", status.Code(err))
	}
}

func TestAuthInterceptorAllowsLegacyTokenWhenEnabled(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-modeloman-token", "legacy-secret"))
	interceptor := AuthUnaryInterceptor("legacy-secret", true, nil)
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodSetPolicy,
	}, func(ctx context.Context, req any) (any, error) {
		principal, ok := principalFromContext(ctx)
		if !ok || principal.KeyID != "legacy_shared_token" {
			t.Fatalf("expected legacy principal in context, got ok=%v key_id=%q", ok, principal.KeyID)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestAuthInterceptorRejectsLegacyTokenWhenDisabled(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-modeloman-token", "legacy-secret"))
	interceptor := AuthUnaryInterceptor("legacy-secret", false, nil)
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodSetPolicy,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %s", status.Code(err))
	}
}

func TestRateLimiterUsesRemoteIPForUnauthenticatedRequests(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(TokenBucketRateLimiterConfig{
		AuthenticatedPerSecond:   100,
		AuthenticatedBurst:       100,
		UnauthenticatedPerSecond: 0.001,
		UnauthenticatedBurst:     1,
		BucketTTL:                time.Minute,
	})
	interceptor := RateLimitUnaryInterceptor(limiter)
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
	})

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodGetHealth,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected first request to pass, got %v", err)
	}

	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodGetHealth,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %s", status.Code(err))
	}
}

func TestRateLimiterUsesKeyIDForAuthenticatedRequests(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(TokenBucketRateLimiterConfig{
		AuthenticatedPerSecond:   0.001,
		AuthenticatedBurst:       1,
		UnauthenticatedPerSecond: 100,
		UnauthenticatedBurst:     100,
		BucketTTL:                time.Minute,
	})
	interceptor := RateLimitUnaryInterceptor(limiter)
	ctxA := withPrincipal(context.Background(), store.AgentPrincipal{KeyID: "key-a"})
	ctxB := withPrincipal(context.Background(), store.AgentPrincipal{KeyID: "key-b"})

	_, err := interceptor(ctxA, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodSetPolicy,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected first key-a request to pass, got %v", err)
	}

	_, err = interceptor(ctxA, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodSetPolicy,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected key-a to be rate limited, got %s", status.Code(err))
	}

	_, err = interceptor(ctxB, nil, &grpc.UnaryServerInfo{
		FullMethod: rpccontract.MethodSetPolicy,
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected separate key-b bucket to pass, got %v", err)
	}
}

func TestIdempotencyInterceptorReturnsStoredResponseOnReplay(t *testing.T) {
	idStore := newFakeIdempotencyStore()
	interceptor := IdempotencyUnaryInterceptor(idStore)
	request, err := structpb.NewStruct(map[string]any{
		"idempotency_key": "req-1",
		"title":           "task title",
	})
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	handlerCalls := 0
	info := &grpc.UnaryServerInfo{FullMethod: rpccontract.MethodCreateTask}
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalls++
		return mustStruct(t, map[string]any{
			"id":    "task_123",
			"title": "task title",
		}), nil
	}

	first, err := interceptor(context.Background(), request, info, handler)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	second, err := interceptor(context.Background(), request, info, handler)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if handlerCalls != 1 {
		t.Fatalf("expected handler to run once, ran %d times", handlerCalls)
	}
	firstStruct := first.(*structpb.Struct).AsMap()
	secondStruct := second.(*structpb.Struct).AsMap()
	if firstStruct["id"] != secondStruct["id"] {
		t.Fatalf("expected replayed response to match first response")
	}
}

func TestIdempotencyInterceptorRejectsMismatchedReplay(t *testing.T) {
	idStore := newFakeIdempotencyStore()
	interceptor := IdempotencyUnaryInterceptor(idStore)

	firstRequest, err := structpb.NewStruct(map[string]any{
		"idempotency_key": "req-2",
		"title":           "original",
	})
	if err != nil {
		t.Fatalf("failed to build first request: %v", err)
	}
	secondRequest, err := structpb.NewStruct(map[string]any{
		"idempotency_key": "req-2",
		"title":           "different",
	})
	if err != nil {
		t.Fatalf("failed to build second request: %v", err)
	}

	info := &grpc.UnaryServerInfo{FullMethod: rpccontract.MethodCreateTask}
	handler := func(ctx context.Context, req any) (any, error) {
		return mustStruct(t, map[string]any{"id": "task_x"}), nil
	}

	if _, err := interceptor(context.Background(), firstRequest, info, handler); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	_, err = interceptor(context.Background(), secondRequest, info, handler)
	appErr, ok := domain.AsAppError(err)
	if !ok || appErr.Code != domain.CodeConflict {
		t.Fatalf("expected conflict AppError, got %#v", err)
	}
}

func TestIdempotencyInterceptorReleasesKeyOnHandlerError(t *testing.T) {
	idStore := newFakeIdempotencyStore()
	interceptor := IdempotencyUnaryInterceptor(idStore)
	request, err := structpb.NewStruct(map[string]any{
		"idempotency_key": "req-3",
		"title":           "retryable",
	})
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	info := &grpc.UnaryServerInfo{FullMethod: rpccontract.MethodCreateTask}
	handlerCalls := 0
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalls++
		if handlerCalls == 1 {
			return nil, domain.Internal("temporary failure", nil)
		}
		return mustStruct(t, map[string]any{"id": "task_ok"}), nil
	}

	if _, err := interceptor(context.Background(), request, info, handler); err == nil {
		t.Fatalf("expected first call to fail")
	}
	if _, err := interceptor(context.Background(), request, info, handler); err != nil {
		t.Fatalf("expected retry to succeed after release, got %v", err)
	}
	if handlerCalls != 2 {
		t.Fatalf("expected handler to run twice, ran %d times", handlerCalls)
	}
}
