package grpcx

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bcrosbie/modeloman/internal/domain"
	"github.com/bcrosbie/modeloman/internal/rpccontract"
	"github.com/bcrosbie/modeloman/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

func AuthUnaryInterceptor(token string, keyAuth store.AgentKeyAuthenticator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if _, isWriteMethod := rpccontract.WriteMethods[info.FullMethod]; !isWriteMethod {
			return handler(ctx, req)
		}

		authEnabled := strings.TrimSpace(token) != "" || keyAuth != nil
		if !authEnabled {
			return handler(ctx, req)
		}

		requestToken := extractToken(ctx)
		if requestToken == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authentication token")
		}

		if keyAuth != nil {
			principal, ok, err := keyAuth.AuthenticateAgentKey(requestToken)
			if err != nil {
				log.Printf("auth validation failure method=%s err=%v", info.FullMethod, err)
				return nil, status.Error(codes.Internal, "authentication subsystem unavailable")
			}
			if ok {
				log.Printf("authenticated write method=%s agent_id=%s key_id=%s", info.FullMethod, principal.AgentID, principal.KeyID)
				return handler(ctx, req)
			}
		}

		if token != "" && requestToken == token {
			return handler(ctx, req)
		}

		return nil, status.Error(codes.Unauthenticated, "invalid authentication token")
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
