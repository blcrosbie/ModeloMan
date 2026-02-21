package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bcrosbie/modeloman/internal/rpccontract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	base := flag.NewFlagSet("modeloman-cli", flag.ExitOnError)
	addr := base.String("addr", "127.0.0.1:50051", "gRPC address")
	token := base.String("token", os.Getenv("AUTH_TOKEN"), "optional auth token or agent API key")
	_ = base.Parse(os.Args[1:])

	args := base.Args()
	if len(args) == 0 {
		usage()
		return
	}

	command := args[0]
	commandArgs := args[1:]

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	if *token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-modeloman-token", *token)
	}

	switch command {
	case "health":
		callStruct(ctx, conn, rpccontract.MethodGetHealth, &emptypb.Empty{})
	case "summary":
		callStruct(ctx, conn, rpccontract.MethodGetSummary, &emptypb.Empty{})
	case "telemetry-summary":
		callStruct(ctx, conn, rpccontract.MethodGetTelemetrySummary, &emptypb.Empty{})
	case "get-policy":
		callStruct(ctx, conn, rpccontract.MethodGetPolicy, &emptypb.Empty{})
	case "list-policy-caps":
		callList(ctx, conn, rpccontract.MethodListPolicyCaps, &emptypb.Empty{})
	case "list-tasks":
		callList(ctx, conn, rpccontract.MethodListTasks, &emptypb.Empty{})
	case "list-runs":
		runListRuns(ctx, conn, commandArgs)
	case "list-attempts":
		runListAttempts(ctx, conn, commandArgs)
	case "list-events":
		runListEvents(ctx, conn, commandArgs)
	case "leaderboard":
		runLeaderboard(ctx, conn, commandArgs)
	case "create-task":
		runCreateTask(ctx, conn, commandArgs)
	case "start-run":
		runStartRun(ctx, conn, commandArgs)
	case "finish-run":
		runFinishRun(ctx, conn, commandArgs)
	case "record-attempt":
		runRecordAttempt(ctx, conn, commandArgs)
	case "record-event":
		runRecordEvent(ctx, conn, commandArgs)
	case "set-policy":
		runSetPolicy(ctx, conn, commandArgs)
	case "upsert-policy-cap":
		runUpsertPolicyCap(ctx, conn, commandArgs)
	case "delete-policy-cap":
		runDeletePolicyCap(ctx, conn, commandArgs)
	case "append-changelog":
		runAppendChangelog(ctx, conn, commandArgs)
	case "record-benchmark":
		runRecordBenchmark(ctx, conn, commandArgs)
	default:
		usage()
	}
}

func runCreateTask(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("create-task", flag.ExitOnError)
	title := flags.String("title", "", "required")
	details := flags.String("details", "", "optional")
	status := flags.String("status", "todo", "todo|in_progress|done|blocked")
	_ = flags.Parse(args)

	if *title == "" {
		log.Fatalf("create-task requires --title")
	}
	request, err := structpb.NewStruct(map[string]any{
		"title":   *title,
		"details": *details,
		"status":  *status,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodCreateTask, request)
}

func runStartRun(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("start-run", flag.ExitOnError)
	workflow := flags.String("workflow", "", "required")
	agentID := flags.String("agent-id", "", "required")
	taskID := flags.String("task-id", "", "optional")
	promptVersion := flags.String("prompt-version", "", "optional")
	modelPolicy := flags.String("model-policy", "", "optional")
	maxRetries := flags.Int64("max-retries", 0, "optional")
	_ = flags.Parse(args)

	if *workflow == "" || *agentID == "" {
		log.Fatalf("start-run requires --workflow and --agent-id")
	}
	request, err := structpb.NewStruct(map[string]any{
		"workflow":       *workflow,
		"agent_id":       *agentID,
		"task_id":        *taskID,
		"prompt_version": *promptVersion,
		"model_policy":   *modelPolicy,
		"max_retries":    *maxRetries,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodStartRun, request)
}

func runFinishRun(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("finish-run", flag.ExitOnError)
	runID := flags.String("run-id", "", "required")
	status := flags.String("status", "completed", "completed|failed|cancelled")
	lastError := flags.String("last-error", "", "optional")
	_ = flags.Parse(args)

	if *runID == "" {
		log.Fatalf("finish-run requires --run-id")
	}
	request, err := structpb.NewStruct(map[string]any{
		"run_id":     *runID,
		"status":     *status,
		"last_error": *lastError,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodFinishRun, request)
}

func runRecordAttempt(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("record-attempt", flag.ExitOnError)
	runID := flags.String("run-id", "", "required")
	attemptNumber := flags.Int64("attempt-number", 1, "required")
	model := flags.String("model", "", "required")
	outcome := flags.String("outcome", "success", "success|failed|timeout|retryable_error|tool_error")
	workflow := flags.String("workflow", "", "optional")
	agentID := flags.String("agent-id", "", "optional")
	providerType := flags.String("provider-type", "api", "optional")
	provider := flags.String("provider", "", "optional")
	promptVersion := flags.String("prompt-version", "", "optional")
	promptHash := flags.String("prompt-hash", "", "optional")
	errorType := flags.String("error-type", "", "optional")
	errorMessage := flags.String("error-message", "", "optional")
	tokensIn := flags.Int64("tokens-in", 0, "optional")
	tokensOut := flags.Int64("tokens-out", 0, "optional")
	costUSD := flags.Float64("cost-usd", 0, "optional")
	latencyMS := flags.Int64("latency-ms", 0, "optional")
	quality := flags.Float64("quality-score", 0, "optional")
	_ = flags.Parse(args)

	if *runID == "" || *model == "" {
		log.Fatalf("record-attempt requires --run-id and --model")
	}
	request, err := structpb.NewStruct(map[string]any{
		"run_id":         *runID,
		"attempt_number": *attemptNumber,
		"workflow":       *workflow,
		"agent_id":       *agentID,
		"provider_type":  *providerType,
		"provider":       *provider,
		"model":          *model,
		"prompt_version": *promptVersion,
		"prompt_hash":    *promptHash,
		"outcome":        *outcome,
		"error_type":     *errorType,
		"error_message":  *errorMessage,
		"tokens_in":      *tokensIn,
		"tokens_out":     *tokensOut,
		"cost_usd":       *costUSD,
		"latency_ms":     *latencyMS,
		"quality_score":  *quality,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodRecordPromptAttempt, request)
}

func runRecordEvent(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("record-event", flag.ExitOnError)
	runID := flags.String("run-id", "", "required")
	eventType := flags.String("event-type", "", "required")
	level := flags.String("level", "info", "info|warn|error")
	message := flags.String("message", "", "optional")
	dataJSON := flags.String("data-json", "", "optional")
	_ = flags.Parse(args)

	if *runID == "" || *eventType == "" {
		log.Fatalf("record-event requires --run-id and --event-type")
	}
	request, err := structpb.NewStruct(map[string]any{
		"run_id":     *runID,
		"event_type": *eventType,
		"level":      *level,
		"message":    *message,
		"data_json":  *dataJSON,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodRecordRunEvent, request)
}

func runSetPolicy(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("set-policy", flag.ExitOnError)
	killSwitch := flags.Bool("kill-switch", false, "true|false")
	reason := flags.String("reason", "", "optional kill switch reason")
	maxCost := flags.Float64("max-cost-per-run", 0, "0 means unlimited")
	maxAttempts := flags.Int64("max-attempts-per-run", 0, "0 means unlimited")
	maxTokens := flags.Int64("max-tokens-per-run", 0, "0 means unlimited")
	maxLatency := flags.Int64("max-latency-ms-per-attempt", 0, "0 means unlimited")
	_ = flags.Parse(args)

	request, err := structpb.NewStruct(map[string]any{
		"kill_switch":                *killSwitch,
		"kill_switch_reason":         *reason,
		"max_cost_per_run_usd":       *maxCost,
		"max_attempts_per_run":       *maxAttempts,
		"max_tokens_per_run":         *maxTokens,
		"max_latency_per_attempt_ms": *maxLatency,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodSetPolicy, request)
}

func runUpsertPolicyCap(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("upsert-policy-cap", flag.ExitOnError)
	id := flags.String("id", "", "optional")
	name := flags.String("name", "", "optional")
	providerType := flags.String("provider-type", "", "optional api|subscription|opensource")
	provider := flags.String("provider", "", "optional")
	model := flags.String("model", "", "optional")
	maxCostRun := flags.Float64("max-cost-run", 0, "0 means inherit global")
	maxAttemptsRun := flags.Int64("max-attempts-run", 0, "0 means inherit global")
	maxTokensRun := flags.Int64("max-tokens-run", 0, "0 means inherit global")
	maxCostAttempt := flags.Float64("max-cost-attempt", 0, "0 means unset")
	maxTokensAttempt := flags.Int64("max-tokens-attempt", 0, "0 means unset")
	maxLatencyAttempt := flags.Int64("max-latency-attempt-ms", 0, "0 means inherit global")
	priority := flags.Int64("priority", 0, "higher wins on same specificity")
	dryRun := flags.Bool("dry-run", false, "log violations without blocking")
	active := flags.Bool("active", true, "true|false")
	_ = flags.Parse(args)

	request, err := structpb.NewStruct(map[string]any{
		"id":                         *id,
		"name":                       *name,
		"provider_type":              *providerType,
		"provider":                   *provider,
		"model":                      *model,
		"max_cost_per_run_usd":       *maxCostRun,
		"max_attempts_per_run":       *maxAttemptsRun,
		"max_tokens_per_run":         *maxTokensRun,
		"max_cost_per_attempt_usd":   *maxCostAttempt,
		"max_tokens_per_attempt":     *maxTokensAttempt,
		"max_latency_per_attempt_ms": *maxLatencyAttempt,
		"priority":                   *priority,
		"dry_run":                    *dryRun,
		"is_active":                  *active,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodUpsertPolicyCap, request)
}

func runDeletePolicyCap(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("delete-policy-cap", flag.ExitOnError)
	id := flags.String("id", "", "required")
	_ = flags.Parse(args)
	if *id == "" {
		log.Fatalf("delete-policy-cap requires --id")
	}
	request, err := structpb.NewStruct(map[string]any{"id": *id})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodDeletePolicyCap, request)
}

func runListRuns(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("list-runs", flag.ExitOnError)
	runID := flags.String("run-id", "", "optional")
	taskID := flags.String("task-id", "", "optional")
	workflow := flags.String("workflow", "", "optional")
	agentID := flags.String("agent-id", "", "optional")
	status := flags.String("status", "", "optional")
	promptVersion := flags.String("prompt-version", "", "optional")
	startedAfter := flags.String("started-after", "", "optional RFC3339")
	startedBefore := flags.String("started-before", "", "optional RFC3339")
	limit := flags.Int64("limit", 0, "optional")
	_ = flags.Parse(args)

	request, err := structpb.NewStruct(map[string]any{
		"run_id":         *runID,
		"task_id":        *taskID,
		"workflow":       *workflow,
		"agent_id":       *agentID,
		"status":         *status,
		"prompt_version": *promptVersion,
		"started_after":  *startedAfter,
		"started_before": *startedBefore,
		"limit":          *limit,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callList(ctx, conn, rpccontract.MethodListRuns, request)
}

func runListAttempts(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("list-attempts", flag.ExitOnError)
	runID := flags.String("run-id", "", "optional")
	workflow := flags.String("workflow", "", "optional")
	agentID := flags.String("agent-id", "", "optional")
	model := flags.String("model", "", "optional")
	outcome := flags.String("outcome", "", "optional")
	promptVersion := flags.String("prompt-version", "", "optional")
	createdAfter := flags.String("created-after", "", "optional RFC3339")
	createdBefore := flags.String("created-before", "", "optional RFC3339")
	limit := flags.Int64("limit", 0, "optional")
	_ = flags.Parse(args)

	request, err := structpb.NewStruct(map[string]any{
		"run_id":         *runID,
		"workflow":       *workflow,
		"agent_id":       *agentID,
		"model":          *model,
		"outcome":        *outcome,
		"prompt_version": *promptVersion,
		"created_after":  *createdAfter,
		"created_before": *createdBefore,
		"limit":          *limit,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callList(ctx, conn, rpccontract.MethodListPromptAttempts, request)
}

func runListEvents(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("list-events", flag.ExitOnError)
	runID := flags.String("run-id", "", "optional")
	eventType := flags.String("event-type", "", "optional")
	level := flags.String("level", "", "optional")
	createdAfter := flags.String("created-after", "", "optional RFC3339")
	createdBefore := flags.String("created-before", "", "optional RFC3339")
	limit := flags.Int64("limit", 0, "optional")
	_ = flags.Parse(args)

	request, err := structpb.NewStruct(map[string]any{
		"run_id":         *runID,
		"event_type":     *eventType,
		"level":          *level,
		"created_after":  *createdAfter,
		"created_before": *createdBefore,
		"limit":          *limit,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callList(ctx, conn, rpccontract.MethodListRunEvents, request)
}

func runLeaderboard(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("leaderboard", flag.ExitOnError)
	workflow := flags.String("workflow", "", "optional")
	model := flags.String("model", "", "optional")
	promptVersion := flags.String("prompt-version", "", "optional")
	windowDays := flags.Int64("window-days", 0, "optional")
	limit := flags.Int64("limit", 20, "optional")
	_ = flags.Parse(args)

	request, err := structpb.NewStruct(map[string]any{
		"workflow":       *workflow,
		"model":          *model,
		"prompt_version": *promptVersion,
		"window_days":    *windowDays,
		"limit":          *limit,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callList(ctx, conn, rpccontract.MethodGetLeaderboard, request)
}

func runAppendChangelog(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("append-changelog", flag.ExitOnError)
	summary := flags.String("summary", "", "required")
	category := flags.String("category", "ops", "platform|policy|model|infra|ops")
	details := flags.String("details", "", "optional")
	actor := flags.String("actor", "", "optional")
	_ = flags.Parse(args)

	if *summary == "" {
		log.Fatalf("append-changelog requires --summary")
	}
	request, err := structpb.NewStruct(map[string]any{
		"summary":  *summary,
		"category": *category,
		"details":  *details,
		"actor":    *actor,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodAppendChangelog, request)
}

func runRecordBenchmark(ctx context.Context, conn grpc.ClientConnInterface, args []string) {
	flags := flag.NewFlagSet("record-benchmark", flag.ExitOnError)
	workflow := flags.String("workflow", "", "required")
	providerType := flags.String("provider-type", "api", "api|subscription|opensource")
	model := flags.String("model", "", "required")
	provider := flags.String("provider", "", "optional")
	tokensIn := flags.Int64("tokens-in", 0, "optional")
	tokensOut := flags.Int64("tokens-out", 0, "optional")
	costUSD := flags.Float64("cost-usd", 0, "optional")
	latencyMS := flags.Int64("latency-ms", 0, "optional")
	quality := flags.Float64("quality-score", 0, "optional")
	notes := flags.String("notes", "", "optional")
	_ = flags.Parse(args)

	if *workflow == "" || *model == "" {
		log.Fatalf("record-benchmark requires --workflow and --model")
	}
	request, err := structpb.NewStruct(map[string]any{
		"workflow":      *workflow,
		"provider_type": *providerType,
		"provider":      *provider,
		"model":         *model,
		"tokens_in":     *tokensIn,
		"tokens_out":    *tokensOut,
		"cost_usd":      *costUSD,
		"latency_ms":    *latencyMS,
		"quality_score": *quality,
		"notes":         *notes,
	})
	if err != nil {
		log.Fatalf("request build error: %v", err)
	}
	callStruct(ctx, conn, rpccontract.MethodRecordBenchmark, request)
}

func callStruct(ctx context.Context, conn grpc.ClientConnInterface, method string, request any) {
	response := &structpb.Struct{}
	if err := conn.Invoke(ctx, method, request, response); err != nil {
		log.Fatalf("rpc error %s: %v", method, err)
	}
	printJSON(response.AsMap())
}

func callList(ctx context.Context, conn grpc.ClientConnInterface, method string, request any) {
	response := &structpb.ListValue{}
	if err := conn.Invoke(ctx, method, request, response); err != nil {
		log.Fatalf("rpc error %s: %v", method, err)
	}
	printJSON(response.AsSlice())
}

func printJSON(value any) {
	serialized, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatalf("encode error: %v", err)
	}
	fmt.Println(string(serialized))
}

func usage() {
	fmt.Print(`ModeloMan gRPC CLI

Usage:
  modeloman-cli [--addr 127.0.0.1:50051] [--token ...] <command> [flags]

Commands:
  health
  summary
  telemetry-summary
  get-policy
  list-policy-caps
  list-tasks
  list-runs [--workflow "..." --status "..."]
  list-attempts [--run-id "..."]
  list-events [--run-id "..."]
  leaderboard [--workflow "..." --window-days 14 --limit 20]
  create-task --title "..."
  start-run --workflow "..." --agent-id "..."
  finish-run --run-id "..." --status completed|failed|cancelled
  record-attempt --run-id "..." --attempt-number 1 --model "..." --outcome success|failed|timeout|retryable_error|tool_error
  record-event --run-id "..." --event-type "..."
  set-policy --kill-switch false --max-cost-per-run 2.5 --max-attempts-per-run 8 --max-tokens-per-run 50000
  upsert-policy-cap --name "expensive-model" --provider-type api --provider openai --model gpt-5 --max-cost-run 5 --max-cost-attempt 0.8 --priority 50
  delete-policy-cap --id "cap_..."
  append-changelog --summary "..."
  record-benchmark --workflow "..." --model "..."
`)
}
