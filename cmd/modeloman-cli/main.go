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
	token := base.String("token", os.Getenv("AUTH_TOKEN"), "optional auth token")
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
	case "list-tasks":
		callList(ctx, conn, rpccontract.MethodListTasks, &emptypb.Empty{})
	case "create-task":
		runCreateTask(ctx, conn, commandArgs)
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
  list-tasks
  create-task --title "..."
  append-changelog --summary "..."
  record-benchmark --workflow "..." --model "..."
`)
}
