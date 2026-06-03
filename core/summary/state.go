package summary

import (
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	session *ExecutionState
)

// Begin starts a new execution session for a command.
func Begin(command string) {
	mu.Lock()
	defer mu.Unlock()
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
		return ExecutionState{}
	}
	session.Success = success
	session.EndTime = time.Now()
	session.Duration = session.EndTime.Sub(session.StartTime)
	out := session.clone()
	session = nil
	return out
}

// EnsureSession starts a session if none is active (uses commandName when provided).
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
			if status == StageWarning && message != "" {
				session.Warnings = appendUnique(session.Warnings, name+": "+message)
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

// MarkRemainingSkipped marks all planned SKIPPED stages after a failure.
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
		if foundFailed && session.Stages[i].Status == StageSkipped {
			// already skipped placeholder — keep
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

// AddWarning records a non-fatal warning.
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
