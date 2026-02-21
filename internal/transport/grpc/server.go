package grpcx

import (
	"context"
	"encoding/json"

	"github.com/bcrosbie/modeloman/internal/domain"
	"github.com/bcrosbie/modeloman/internal/rpccontract"
	"github.com/bcrosbie/modeloman/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type HubRPCServer interface {
	GetHealth(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	GetSummary(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	ExportState(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	CreateTask(context.Context, *structpb.Struct) (*structpb.Struct, error)
	UpdateTask(context.Context, *structpb.Struct) (*structpb.Struct, error)
	DeleteTask(context.Context, *structpb.Struct) (*structpb.Struct, error)
	ListTasks(context.Context, *emptypb.Empty) (*structpb.ListValue, error)
	CreateNote(context.Context, *structpb.Struct) (*structpb.Struct, error)
	ListNotes(context.Context, *emptypb.Empty) (*structpb.ListValue, error)
	AppendChangelog(context.Context, *structpb.Struct) (*structpb.Struct, error)
	ListChangelog(context.Context, *emptypb.Empty) (*structpb.ListValue, error)
	RecordBenchmark(context.Context, *structpb.Struct) (*structpb.Struct, error)
	ListBenchmarks(context.Context, *emptypb.Empty) (*structpb.ListValue, error)
	StartRun(context.Context, *structpb.Struct) (*structpb.Struct, error)
	FinishRun(context.Context, *structpb.Struct) (*structpb.Struct, error)
	ListRuns(context.Context, *structpb.Struct) (*structpb.ListValue, error)
	RecordPromptAttempt(context.Context, *structpb.Struct) (*structpb.Struct, error)
	ListPromptAttempts(context.Context, *structpb.Struct) (*structpb.ListValue, error)
	RecordRunEvent(context.Context, *structpb.Struct) (*structpb.Struct, error)
	ListRunEvents(context.Context, *structpb.Struct) (*structpb.ListValue, error)
	GetTelemetrySummary(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	GetPolicy(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	SetPolicy(context.Context, *structpb.Struct) (*structpb.Struct, error)
	GetLeaderboard(context.Context, *structpb.Struct) (*structpb.ListValue, error)
	ListPolicyCaps(context.Context, *emptypb.Empty) (*structpb.ListValue, error)
	UpsertPolicyCap(context.Context, *structpb.Struct) (*structpb.Struct, error)
	DeletePolicyCap(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

type HubHandler struct {
	hub *service.HubService
}

func NewHubHandler(hub *service.HubService) *HubHandler {
	return &HubHandler{hub: hub}
}

func RegisterHubServer(server *grpc.Server, handler HubRPCServer) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: rpccontract.ServiceName,
		HandlerType: (*HubRPCServer)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "GetHealth", Handler: getHealthHandler},
			{MethodName: "GetSummary", Handler: getSummaryHandler},
			{MethodName: "ExportState", Handler: exportStateHandler},
			{MethodName: "CreateTask", Handler: createTaskHandler},
			{MethodName: "UpdateTask", Handler: updateTaskHandler},
			{MethodName: "DeleteTask", Handler: deleteTaskHandler},
			{MethodName: "ListTasks", Handler: listTasksHandler},
			{MethodName: "CreateNote", Handler: createNoteHandler},
			{MethodName: "ListNotes", Handler: listNotesHandler},
			{MethodName: "AppendChangelog", Handler: appendChangelogHandler},
			{MethodName: "ListChangelog", Handler: listChangelogHandler},
			{MethodName: "RecordBenchmark", Handler: recordBenchmarkHandler},
			{MethodName: "ListBenchmarks", Handler: listBenchmarksHandler},
			{MethodName: "StartRun", Handler: startRunHandler},
			{MethodName: "FinishRun", Handler: finishRunHandler},
			{MethodName: "ListRuns", Handler: listRunsHandler},
			{MethodName: "RecordPromptAttempt", Handler: recordPromptAttemptHandler},
			{MethodName: "ListPromptAttempts", Handler: listPromptAttemptsHandler},
			{MethodName: "RecordRunEvent", Handler: recordRunEventHandler},
			{MethodName: "ListRunEvents", Handler: listRunEventsHandler},
			{MethodName: "GetTelemetrySummary", Handler: getTelemetrySummaryHandler},
			{MethodName: "GetPolicy", Handler: getPolicyHandler},
			{MethodName: "SetPolicy", Handler: setPolicyHandler},
			{MethodName: "GetLeaderboard", Handler: getLeaderboardHandler},
			{MethodName: "ListPolicyCaps", Handler: listPolicyCapsHandler},
			{MethodName: "UpsertPolicyCap", Handler: upsertPolicyCapHandler},
			{MethodName: "DeletePolicyCap", Handler: deletePolicyCapHandler},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "proto/modeloman/v1/hub.proto",
	}, handler)
}

func (h *HubHandler) GetHealth(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	return toStruct(h.hub.Health())
}

func (h *HubHandler) GetSummary(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	summary, err := h.hub.Summary()
	if err != nil {
		return nil, err
	}
	return toStruct(summary)
}

func (h *HubHandler) ExportState(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	state, err := h.hub.ExportState()
	if err != nil {
		return nil, err
	}
	return toStruct(state)
}

func (h *HubHandler) CreateTask(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.CreateTaskRequest](request)
	if err != nil {
		return nil, err
	}
	created, err := h.hub.CreateTask(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(created)
}

func (h *HubHandler) UpdateTask(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.UpdateTaskRequest](request)
	if err != nil {
		return nil, err
	}
	updated, err := h.hub.UpdateTask(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(updated)
}

func (h *HubHandler) DeleteTask(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.DeleteTaskRequest](request)
	if err != nil {
		return nil, err
	}
	if err := h.hub.DeleteTask(decoded); err != nil {
		return nil, err
	}
	return toStruct(map[string]any{"ok": true})
}

func (h *HubHandler) ListTasks(_ context.Context, _ *emptypb.Empty) (*structpb.ListValue, error) {
	items, err := h.hub.ListTasks()
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) CreateNote(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.CreateNoteRequest](request)
	if err != nil {
		return nil, err
	}
	created, err := h.hub.CreateNote(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(created)
}

func (h *HubHandler) ListNotes(_ context.Context, _ *emptypb.Empty) (*structpb.ListValue, error) {
	items, err := h.hub.ListNotes()
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) AppendChangelog(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.AppendChangelogRequest](request)
	if err != nil {
		return nil, err
	}
	created, err := h.hub.AppendChangelog(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(created)
}

func (h *HubHandler) ListChangelog(_ context.Context, _ *emptypb.Empty) (*structpb.ListValue, error) {
	items, err := h.hub.ListChangelog()
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) RecordBenchmark(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.RecordBenchmarkRequest](request)
	if err != nil {
		return nil, err
	}
	recorded, err := h.hub.RecordBenchmark(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(recorded)
}

func (h *HubHandler) ListBenchmarks(_ context.Context, _ *emptypb.Empty) (*structpb.ListValue, error) {
	items, err := h.hub.ListBenchmarks()
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) StartRun(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.StartRunRequest](request)
	if err != nil {
		return nil, err
	}
	created, err := h.hub.StartRun(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(created)
}

func (h *HubHandler) FinishRun(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.FinishRunRequest](request)
	if err != nil {
		return nil, err
	}
	updated, err := h.hub.FinishRun(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(updated)
}

func (h *HubHandler) ListRuns(_ context.Context, request *structpb.Struct) (*structpb.ListValue, error) {
	decoded, err := decodeStruct[service.ListRunsRequest](request)
	if err != nil {
		return nil, err
	}
	items, err := h.hub.ListRuns(decoded)
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) RecordPromptAttempt(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.RecordPromptAttemptRequest](request)
	if err != nil {
		return nil, err
	}
	recorded, err := h.hub.RecordPromptAttempt(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(recorded)
}

func (h *HubHandler) ListPromptAttempts(_ context.Context, request *structpb.Struct) (*structpb.ListValue, error) {
	decoded, err := decodeStruct[service.ListPromptAttemptsRequest](request)
	if err != nil {
		return nil, err
	}
	items, err := h.hub.ListPromptAttempts(decoded)
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) RecordRunEvent(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.RecordRunEventRequest](request)
	if err != nil {
		return nil, err
	}
	recorded, err := h.hub.RecordRunEvent(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(recorded)
}

func (h *HubHandler) ListRunEvents(_ context.Context, request *structpb.Struct) (*structpb.ListValue, error) {
	decoded, err := decodeStruct[service.ListRunEventsRequest](request)
	if err != nil {
		return nil, err
	}
	items, err := h.hub.ListRunEvents(decoded)
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) GetTelemetrySummary(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	summary, err := h.hub.TelemetrySummary()
	if err != nil {
		return nil, err
	}
	return toStruct(summary)
}

func (h *HubHandler) GetPolicy(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	policy, err := h.hub.GetPolicy()
	if err != nil {
		return nil, err
	}
	return toStruct(policy)
}

func (h *HubHandler) SetPolicy(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.SetPolicyRequest](request)
	if err != nil {
		return nil, err
	}
	policy, err := h.hub.SetPolicy(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(policy)
}

func (h *HubHandler) GetLeaderboard(_ context.Context, request *structpb.Struct) (*structpb.ListValue, error) {
	decoded, err := decodeStruct[service.LeaderboardRequest](request)
	if err != nil {
		return nil, err
	}
	items, err := h.hub.Leaderboard(decoded)
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) ListPolicyCaps(_ context.Context, _ *emptypb.Empty) (*structpb.ListValue, error) {
	items, err := h.hub.ListPolicyCaps()
	if err != nil {
		return nil, err
	}
	return toList(items)
}

func (h *HubHandler) UpsertPolicyCap(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.UpsertPolicyCapRequest](request)
	if err != nil {
		return nil, err
	}
	item, err := h.hub.UpsertPolicyCap(decoded)
	if err != nil {
		return nil, err
	}
	return toStruct(item)
}

func (h *HubHandler) DeletePolicyCap(_ context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	decoded, err := decodeStruct[service.DeletePolicyCapRequest](request)
	if err != nil {
		return nil, err
	}
	if err := h.hub.DeletePolicyCap(decoded); err != nil {
		return nil, err
	}
	return toStruct(map[string]any{"ok": true})
}

func toStruct(value any) (*structpb.Struct, error) {
	serialized, err := json.Marshal(value)
	if err != nil {
		return nil, domain.Internal("failed to encode response", err)
	}

	decoded := map[string]any{}
	if err := json.Unmarshal(serialized, &decoded); err != nil {
		return nil, domain.Internal("failed to shape response object", err)
	}
	result, err := structpb.NewStruct(decoded)
	if err != nil {
		return nil, domain.Internal("failed to convert response to protobuf struct", err)
	}
	return result, nil
}

func toList(value any) (*structpb.ListValue, error) {
	serialized, err := json.Marshal(value)
	if err != nil {
		return nil, domain.Internal("failed to encode response list", err)
	}

	decoded := []any{}
	if err := json.Unmarshal(serialized, &decoded); err != nil {
		return nil, domain.Internal("failed to shape response list", err)
	}
	result, err := structpb.NewList(decoded)
	if err != nil {
		return nil, domain.Internal("failed to convert response to protobuf list", err)
	}
	return result, nil
}

func decodeStruct[T any](input *structpb.Struct) (T, error) {
	var out T
	serialized, err := json.Marshal(input.AsMap())
	if err != nil {
		return out, domain.InvalidArgument("request payload could not be encoded")
	}
	if err := json.Unmarshal(serialized, &out); err != nil {
		return out, domain.InvalidArgument("request payload shape is invalid")
	}
	return out, nil
}

func getHealthHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).GetHealth(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodGetHealth}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).GetHealth(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func getSummaryHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).GetSummary(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodGetSummary}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).GetSummary(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func exportStateHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ExportState(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodExportState}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ExportState(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func createTaskHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).CreateTask(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodCreateTask}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).CreateTask(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func updateTaskHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).UpdateTask(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodUpdateTask}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).UpdateTask(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func deleteTaskHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).DeleteTask(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodDeleteTask}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).DeleteTask(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listTasksHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListTasks(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListTasks}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListTasks(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func createNoteHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).CreateNote(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodCreateNote}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).CreateNote(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listNotesHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListNotes(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListNotes}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListNotes(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func appendChangelogHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).AppendChangelog(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodAppendChangelog}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).AppendChangelog(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listChangelogHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListChangelog(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListChangelog}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListChangelog(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func recordBenchmarkHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).RecordBenchmark(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodRecordBenchmark}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).RecordBenchmark(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listBenchmarksHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListBenchmarks(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListBenchmarks}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListBenchmarks(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func startRunHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).StartRun(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodStartRun}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).StartRun(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func finishRunHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).FinishRun(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodFinishRun}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).FinishRun(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listRunsHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListRuns(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListRuns}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListRuns(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func recordPromptAttemptHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).RecordPromptAttempt(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodRecordPromptAttempt}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).RecordPromptAttempt(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listPromptAttemptsHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListPromptAttempts(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListPromptAttempts}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListPromptAttempts(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func recordRunEventHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).RecordRunEvent(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodRecordRunEvent}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).RecordRunEvent(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listRunEventsHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListRunEvents(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListRunEvents}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListRunEvents(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func getTelemetrySummaryHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).GetTelemetrySummary(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodGetTelemetrySummary}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).GetTelemetrySummary(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func getPolicyHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).GetPolicy(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodGetPolicy}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).GetPolicy(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func setPolicyHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).SetPolicy(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodSetPolicy}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).SetPolicy(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func getLeaderboardHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).GetLeaderboard(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodGetLeaderboard}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).GetLeaderboard(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func listPolicyCapsHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(emptypb.Empty)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).ListPolicyCaps(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodListPolicyCaps}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).ListPolicyCaps(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, request, info, handler)
}

func upsertPolicyCapHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).UpsertPolicyCap(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodUpsertPolicyCap}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).UpsertPolicyCap(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func deletePolicyCapHandler(
	srv any,
	ctx context.Context,
	decoder func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HubRPCServer).DeletePolicyCap(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: rpccontract.MethodDeletePolicyCap}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(HubRPCServer).DeletePolicyCap(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}
