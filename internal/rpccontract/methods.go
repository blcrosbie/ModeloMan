package rpccontract

const (
	ServiceName = "modeloman.v1.ModeloManHub"
)

const (
	MethodGetHealth       = "/" + ServiceName + "/GetHealth"
	MethodGetSummary      = "/" + ServiceName + "/GetSummary"
	MethodExportState     = "/" + ServiceName + "/ExportState"
	MethodCreateTask      = "/" + ServiceName + "/CreateTask"
	MethodUpdateTask      = "/" + ServiceName + "/UpdateTask"
	MethodDeleteTask      = "/" + ServiceName + "/DeleteTask"
	MethodListTasks       = "/" + ServiceName + "/ListTasks"
	MethodCreateNote      = "/" + ServiceName + "/CreateNote"
	MethodListNotes       = "/" + ServiceName + "/ListNotes"
	MethodAppendChangelog = "/" + ServiceName + "/AppendChangelog"
	MethodListChangelog   = "/" + ServiceName + "/ListChangelog"
	MethodRecordBenchmark = "/" + ServiceName + "/RecordBenchmark"
	MethodListBenchmarks  = "/" + ServiceName + "/ListBenchmarks"
)

var WriteMethods = map[string]struct{}{
	MethodCreateTask:      {},
	MethodUpdateTask:      {},
	MethodDeleteTask:      {},
	MethodCreateNote:      {},
	MethodAppendChangelog: {},
	MethodRecordBenchmark: {},
}
