package telemetry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mmconfig "github.com/bcrosbie/modeloman/internal/mm/config"
	"github.com/bcrosbie/modeloman/internal/rpccontract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Client struct {
	conn          *grpc.ClientConn
	token         string
	requestTO     time.Duration
	retryAttempts int
}

type StartRunInput struct {
	Workflow      string
	AgentID       string
	PromptVersion string
	ModelPolicy   string
}

type AttemptInput struct {
	RunID         string
	AttemptNumber int64
	Workflow      string
	AgentID       string
	Model         string
	PromptVersion string
	PromptHash    string
	Outcome       string
	ErrorMessage  string
	LatencyMS     int64
}

type EventInput struct {
	RunID     string
	EventType string
	Level     string
	Message   string
	Data      map[string]any
}

type FinishRunInput struct {
	RunID     string
	Status    string
	LastError string
}

func New(cfg mmconfig.Config, token string) (*Client, error) {
	cred := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}))
	if cfg.GRPCInsecure || strings.HasPrefix(cfg.GRPCAddr, "127.0.0.1:") || strings.HasPrefix(cfg.GRPCAddr, "localhost:") {
		cred = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	conn, err := grpc.NewClient(
		cfg.GRPCAddr,
		cred,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                25 * time.Second,
			Timeout:             6 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", cfg.GRPCAddr, err)
	}

	// Trigger initial connect attempt on startup.
	conn.Connect()

	return &Client{
		conn:          conn,
		token:         strings.TrimSpace(token),
		requestTO:     cfg.RequestTimeout,
		retryAttempts: cfg.RetryAttempts,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) StartRun(ctx context.Context, input StartRunInput) (string, error) {
	response, err := c.invokeStruct(ctx, rpccontract.MethodStartRun, map[string]any{
		"workflow":       strings.TrimSpace(input.Workflow),
		"agent_id":       strings.TrimSpace(input.AgentID),
		"prompt_version": strings.TrimSpace(input.PromptVersion),
		"model_policy":   strings.TrimSpace(input.ModelPolicy),
	})
	if err != nil {
		return "", err
	}
	runID, _ := response["id"].(string)
	if strings.TrimSpace(runID) == "" {
		return "", fmt.Errorf("start run response missing id")
	}
	return runID, nil
}

func (c *Client) RecordPromptAttempt(ctx context.Context, input AttemptInput) error {
	_, err := c.invokeStruct(ctx, rpccontract.MethodRecordPromptAttempt, map[string]any{
		"run_id":         strings.TrimSpace(input.RunID),
		"attempt_number": input.AttemptNumber,
		"workflow":       strings.TrimSpace(input.Workflow),
		"agent_id":       strings.TrimSpace(input.AgentID),
		"provider_type":  "api",
		"provider":       "wrapped-cli",
		"model":          strings.TrimSpace(input.Model),
		"prompt_version": strings.TrimSpace(input.PromptVersion),
		"prompt_hash":    strings.TrimSpace(input.PromptHash),
		"outcome":        strings.TrimSpace(input.Outcome),
		"error_type":     "",
		"error_message":  strings.TrimSpace(input.ErrorMessage),
		"tokens_in":      int64(0),
		"tokens_out":     int64(0),
		"cost_usd":       0.0,
		"latency_ms":     input.LatencyMS,
		"quality_score":  0.0,
	})
	return err
}

func (c *Client) RecordRunEvent(ctx context.Context, input EventInput) error {
	level := strings.TrimSpace(input.Level)
	if level == "" {
		level = "info"
	}
	payload := ""
	if input.Data != nil {
		raw, _ := json.Marshal(input.Data)
		payload = string(raw)
	}
	_, err := c.invokeStruct(ctx, rpccontract.MethodRecordRunEvent, map[string]any{
		"run_id":     strings.TrimSpace(input.RunID),
		"event_type": strings.TrimSpace(input.EventType),
		"level":      level,
		"message":    strings.TrimSpace(input.Message),
		"data_json":  payload,
	})
	return err
}

func (c *Client) FinishRun(ctx context.Context, input FinishRunInput) error {
	_, err := c.invokeStruct(ctx, rpccontract.MethodFinishRun, map[string]any{
		"run_id":     strings.TrimSpace(input.RunID),
		"status":     strings.TrimSpace(input.Status),
		"last_error": strings.TrimSpace(input.LastError),
	})
	return err
}

func (c *Client) invokeStruct(ctx context.Context, method string, payload map[string]any) (map[string]any, error) {
	request, err := structpb.NewStruct(payload)
	if err != nil {
		return nil, err
	}

	attempts := c.retryAttempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, c.requestTO)
		callCtx = c.withAuth(callCtx)

		response := &structpb.Struct{}
		invokeErr := c.conn.Invoke(callCtx, method, request, response)
		cancel()
		if invokeErr == nil {
			return response.AsMap(), nil
		}
		lastErr = invokeErr
		if !isRetryable(invokeErr) || attempt == attempts {
			break
		}
		time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
	}
	return nil, lastErr
}

func (c *Client) withAuth(ctx context.Context) context.Context {
	if strings.TrimSpace(c.token) == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-modeloman-token", c.token)
}

func isRetryable(err error) bool {
	code := status.Code(err)
	switch code {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
