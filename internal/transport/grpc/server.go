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
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "proto/modeloman/v1/hub.proto",
	}, handler)
}

func (h *HubHandler) GetHealth(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	return toStruct(h.hub.Health())
}

func (h *HubHandler) GetSummary(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	return toStruct(h.hub.Summary())
}

func (h *HubHandler) ExportState(_ context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	return toStruct(h.hub.ExportState())
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
	return toList(h.hub.ListTasks())
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
	return toList(h.hub.ListNotes())
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
	return toList(h.hub.ListChangelog())
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
	return toList(h.hub.ListBenchmarks())
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
