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

const (
	ScopeTasksWrite     = "tasks:write"
	ScopeTelemetryWrite = "telemetry:write"
	ScopePolicyWrite    = "policy:write"
	ScopeAdminRead      = "admin:read"
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

var PublicReadMethods = map[string]struct{}{
	MethodGetHealth:           {},
	MethodGetLeaderboard:      {},
	MethodGetTelemetrySummary: {},
}

var PrivateReadMethods = map[string]struct{}{
	MethodGetSummary:         {},
	MethodExportState:        {},
	MethodListTasks:          {},
	MethodListNotes:          {},
	MethodListChangelog:      {},
	MethodListBenchmarks:     {},
	MethodListRuns:           {},
	MethodListPromptAttempts: {},
	MethodListRunEvents:      {},
	MethodGetPolicy:          {},
	MethodListPolicyCaps:     {},
}

var MethodScopes = map[string]string{
	MethodGetSummary:         ScopeAdminRead,
	MethodExportState:        ScopeAdminRead,
	MethodListTasks:          ScopeAdminRead,
	MethodListNotes:          ScopeAdminRead,
	MethodListChangelog:      ScopeAdminRead,
	MethodListBenchmarks:     ScopeAdminRead,
	MethodListRuns:           ScopeAdminRead,
	MethodListPromptAttempts: ScopeAdminRead,
	MethodListRunEvents:      ScopeAdminRead,
	MethodGetPolicy:          ScopeAdminRead,
	MethodListPolicyCaps:     ScopeAdminRead,

	MethodCreateTask:      ScopeTasksWrite,
	MethodUpdateTask:      ScopeTasksWrite,
	MethodDeleteTask:      ScopeTasksWrite,
	MethodCreateNote:      ScopeTasksWrite,
	MethodAppendChangelog: ScopeTasksWrite,

	MethodRecordBenchmark:     ScopeTelemetryWrite,
	MethodStartRun:            ScopeTelemetryWrite,
	MethodFinishRun:           ScopeTelemetryWrite,
	MethodRecordPromptAttempt: ScopeTelemetryWrite,
	MethodRecordRunEvent:      ScopeTelemetryWrite,

	MethodSetPolicy:       ScopePolicyWrite,
	MethodUpsertPolicyCap: ScopePolicyWrite,
	MethodDeletePolicyCap: ScopePolicyWrite,
}

var DefaultAgentKeyScopes = []string{
	ScopeTasksWrite,
	ScopeTelemetryWrite,
	ScopePolicyWrite,
	ScopeAdminRead,
}

func RequiresAuthentication(fullMethod string) bool {
	if _, ok := WriteMethods[fullMethod]; ok {
		return true
	}
	_, ok := PrivateReadMethods[fullMethod]
	return ok
}

func RequiredScope(fullMethod string) (string, bool) {
	scope, ok := MethodScopes[fullMethod]
	return scope, ok
}
