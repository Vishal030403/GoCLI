package summary

import (
	"sync"
	"time"
)

var (
	mu              sync.Mutex
	session         *ExecutionState
	lastCommandName string
	runSuccess      = true
	summaryOnce     sync.Once
	summaryDone     bool
)

// HasActiveSession reports whether a command is being tracked for summary.
func HasActiveSession() bool {
	mu.Lock()
	defer mu.Unlock()
	return session != nil
}

// MarkFailed marks the current run as unsuccessful.
func MarkFailed() {
	mu.Lock()
	defer mu.Unlock()
	runSuccess = false
}

// ResetSummaryGuard allows a new summary for the next command invocation.
func ResetSummaryGuard() {
	mu.Lock()
	defer mu.Unlock()
	summaryOnce = sync.Once{}
	summaryDone = false
}

// Begin starts a new execution session for a command.
func Begin(command string) {
	ResetSummaryGuard()
	mu.Lock()
	defer mu.Unlock()
	lastCommandName = command
	runSuccess = true
	session = &ExecutionState{
		Command:   command,
		StartTime: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// BeginWithPlan starts a session and registers planned stage names (marked SKIPPED until run).
func BeginWithPlan(command string, plannedStages []string) {
	Begin(command)
	mu.Lock()
	defer mu.Unlock()
	for _, name := range plannedStages {
		session.Stages = append(session.Stages, StageRecord{
			Name:   name,
			Status: StageSkipped,
		})
	}
}

// BeginTunnel initializes tunnel metrics on the active session.
func BeginTunnel(appName, namespace, localPort, targetPort string) {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return
	}
	session.Tunnel = &TunnelSummary{
		AppName:   appName,
		Namespace: namespace,
		LocalPort: localPort,
		TargetPort: targetPort,
		StartTime: time.Now(),
	}
	session.Metadata["app"] = appName
	session.Metadata["namespace"] = namespace
}

// RecordTunnelRequest increments forwarded request count (from port-forward output).
func RecordTunnelRequest() {
	mu.Lock()
	defer mu.Unlock()
	if session != nil && session.Tunnel != nil {
		session.Tunnel.RequestsForwarded++
	}
}

// FinalizeTunnel closes tunnel timing and sets outcome text.
func FinalizeTunnel(outcome string) {
	mu.Lock()
	defer mu.Unlock()
	if session == nil || session.Tunnel == nil {
		return
	}
	session.Tunnel.EndTime = time.Now()
	session.Tunnel.Duration = session.Tunnel.EndTime.Sub(session.Tunnel.StartTime)
	if outcome != "" {
		session.Tunnel.Outcome = outcome
	}
}

// Snapshot returns a copy of the current session state.
func Snapshot() ExecutionState {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return ExecutionState{}
	}
	return session.clone()
}

// Finish marks the session complete and returns the final state.
func Finish(success bool) ExecutionState {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return ExecutionState{Command: lastCommandName, Success: success}
	}
	session.Success = success
	if session.EndTime.IsZero() {
		session.EndTime = time.Now()
	}
	if session.Duration == 0 {
		session.Duration = session.EndTime.Sub(session.StartTime)
	}
	out := session.clone()
	session = nil
	return out
}

// EnsureSession starts a session if none is active.
func EnsureSession(commandName string) {
	mu.Lock()
	defer mu.Unlock()
	if session != nil {
		return
	}
	cmd := commandName
	if cmd == "" {
		cmd = "pipeline"
	}
	lastCommandName = cmd
	runSuccess = true
	session = &ExecutionState{
		Command:   cmd,
		StartTime: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// RecordStage records or updates a stage outcome.
func RecordStage(name string, status StageStatus, message string) {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return
	}
	if status == StageWarning && message != "" {
		session.Warnings = appendUnique(session.Warnings, name+": "+message)
	}
	for i := range session.Stages {
		if session.Stages[i].Name == name {
			session.Stages[i].Status = status
			session.Stages[i].Message = message
			if status == StageFailed {
				session.FailedStage = name
				if message != "" {
					session.Errors = appendUnique(session.Errors, message)
				}
			}
			return
		}
	}
	session.Stages = append(session.Stages, StageRecord{
		Name:    name,
		Status:  status,
		Message: message,
	})
	if status == StageFailed {
		session.FailedStage = name
		if message != "" {
			session.Errors = appendUnique(session.Errors, message)
		}
	}
}

// MarkRemainingSkipped marks stages after a failure as skipped.
func MarkRemainingSkipped() {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return
	}
	foundFailed := false
	for i := range session.Stages {
		if session.Stages[i].Status == StageFailed {
			foundFailed = true
			continue
		}
		if foundFailed && session.Stages[i].Status != StageSuccess && session.Stages[i].Status != StageWarning {
			session.Stages[i].Status = StageSkipped
		}
	}
}

// RecordInfrastructure adds an infrastructure item.
func RecordInfrastructure(name, detail string) {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return
	}
	session.Infrastructure = append(session.Infrastructure, InfrastructureItem{
		Name:   name,
		Detail: detail,
	})
}

// AddWarning records an explicit non-fatal warning (not inferred from stages).
func AddWarning(msg string) {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return
	}
	session.Warnings = appendUnique(session.Warnings, msg)
}

// SetMetadata sets a key-value hint for summary generation.
func SetMetadata(key, value string) {
	mu.Lock()
	defer mu.Unlock()
	if session == nil {
		return
	}
	session.Metadata[key] = value
}

func (e *ExecutionState) clone() ExecutionState {
	out := *e
	out.Stages = append([]StageRecord(nil), e.Stages...)
	out.Warnings = append([]string(nil), e.Warnings...)
	out.Errors = append([]string(nil), e.Errors...)
	out.Infrastructure = append([]InfrastructureItem(nil), e.Infrastructure...)
	out.Metadata = make(map[string]string, len(e.Metadata))
	for k, v := range e.Metadata {
		out.Metadata[k] = v
	}
	if e.Tunnel != nil {
		t := *e.Tunnel
		out.Tunnel = &t
	}
	return out
}

func appendUnique(list []string, item string) []string {
	for _, s := range list {
		if s == item {
			return list
		}
	}
	return append(list, item)
}
