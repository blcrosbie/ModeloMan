package rpccontract

const (
	ServiceName = "modeloman.v1.ModeloManHub"
)

const (
	MethodGetHealth           = "/" + ServiceName + "/GetHealth"
	MethodGetSummary          = "/" + ServiceName + "/GetSummary"
	MethodExportState         = "/" + ServiceName + "/ExportState"
	MethodCreateTask          = "/" + ServiceName + "/CreateTask"
	MethodUpdateTask          = "/" + ServiceName + "/UpdateTask"
	MethodDeleteTask          = "/" + ServiceName + "/DeleteTask"
	MethodListTasks           = "/" + ServiceName + "/ListTasks"
	MethodCreateNote          = "/" + ServiceName + "/CreateNote"
	MethodListNotes           = "/" + ServiceName + "/ListNotes"
	MethodAppendChangelog     = "/" + ServiceName + "/AppendChangelog"
	MethodListChangelog       = "/" + ServiceName + "/ListChangelog"
	MethodRecordBenchmark     = "/" + ServiceName + "/RecordBenchmark"
	MethodListBenchmarks      = "/" + ServiceName + "/ListBenchmarks"
	MethodStartRun            = "/" + ServiceName + "/StartRun"
	MethodFinishRun           = "/" + ServiceName + "/FinishRun"
	MethodListRuns            = "/" + ServiceName + "/ListRuns"
	MethodRecordPromptAttempt = "/" + ServiceName + "/RecordPromptAttempt"
	MethodListPromptAttempts  = "/" + ServiceName + "/ListPromptAttempts"
	MethodRecordRunEvent      = "/" + ServiceName + "/RecordRunEvent"
	MethodListRunEvents       = "/" + ServiceName + "/ListRunEvents"
	MethodGetTelemetrySummary = "/" + ServiceName + "/GetTelemetrySummary"
	MethodGetPolicy           = "/" + ServiceName + "/GetPolicy"
	MethodSetPolicy           = "/" + ServiceName + "/SetPolicy"
	MethodGetLeaderboard      = "/" + ServiceName + "/GetLeaderboard"
	MethodListPolicyCaps      = "/" + ServiceName + "/ListPolicyCaps"
	MethodUpsertPolicyCap     = "/" + ServiceName + "/UpsertPolicyCap"
	MethodDeletePolicyCap     = "/" + ServiceName + "/DeletePolicyCap"
)

var WriteMethods = map[string]struct{}{
	MethodCreateTask:          {},
	MethodUpdateTask:          {},
	MethodDeleteTask:          {},
	MethodCreateNote:          {},
	MethodAppendChangelog:     {},
	MethodRecordBenchmark:     {},
	MethodStartRun:            {},
	MethodFinishRun:           {},
	MethodRecordPromptAttempt: {},
	MethodRecordRunEvent:      {},
	MethodSetPolicy:           {},
	MethodUpsertPolicyCap:     {},
	MethodDeletePolicyCap:     {},
}
