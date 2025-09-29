// Copyright 2025 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import "time"

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

// PipelineRunDebugInfo represents a distilled view of a PipelineRun's outcome
// and associated failed TaskRuns if any exist.
type PipelineRunDebugInfo struct {
	PipelineRun    PipelineRunMetadata `json:"pipelineRun"`
	Status         PipelineRunStatus   `json:"status"`
	FailedTaskRuns []TaskRunSummary    `json:"failedTaskRuns"`
	Analysis       string              `json:"analysis"`
}

// PipelineRunMetadata contains basic metadata about the PipelineRun
type PipelineRunMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	UID         string            `json:"uid"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// PipelineRunStatus contains the status information of a PipelineRun
type PipelineRunStatus struct {
	Phase           string                 `json:"phase"`
	StartTime       *time.Time             `json:"startTime,omitempty"`
	CompletionTime  *time.Time             `json:"completionTime,omitempty"`
	DurationSeconds int64                  `json:"durationSeconds"`
	Conditions      []PipelineRunCondition `json:"conditions"`
}

// PipelineRunCondition represents a condition of the PipelineRun
type PipelineRunCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
}

// TaskRunSummary contains summary information about a failed TaskRun
type TaskRunSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}
