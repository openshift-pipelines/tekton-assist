package types

// TaskRunDebugInfo represents a distilled view of a TaskRun's outcome
// and the primary failure signal if it did not succeed.
type TaskRunDebugInfo struct {
	TaskRun    string    `json:"taskrun"`
	Namespace  string    `json:"namespace"`
	Succeeded  bool      `json:"succeeded"`
	FailedStep StepInfo  `json:"failed_step,omitempty"`
	Error      ErrorInfo `json:"error,omitempty"`
}

type StepInfo struct {
	Name     string `json:"name"`
	ExitCode int32  `json:"exit_code"`
}

type ErrorInfo struct {
	Type       string `json:"type"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	LogSnippet string `json:"log_snippet"`
}
