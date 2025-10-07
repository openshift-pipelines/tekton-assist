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

package analysis

import (
	"fmt"
	"strings"

	"github.com/openshift-pipelines/tekton-assist/pkg/types"
)

// BuildTaskRunPrompt creates a concise user prompt for the LLM from TaskRunDebugInfo.
func BuildTaskRunPrompt(info types.TaskRunDebugInfo) string {
	var b strings.Builder

	fmt.Fprintf(&b, "You are a senior DevOps engineer specializing in Kubernetes, Tekton, and CI/CD pipelines. ")
	fmt.Fprintf(&b, "Analyze this Tekton TaskRun failure and provide specific remediation steps.\n\n")

	fmt.Fprintf(&b, "TASK RUN DETAILS:\n")
	fmt.Fprintf(&b, "- TaskRun: %s\n", info.TaskRun)
	fmt.Fprintf(&b, "- Namespace: %s\n", info.Namespace)
	fmt.Fprintf(&b, "- Status: %s\n", map[bool]string{true: "Succeeded", false: "Failed"}[info.Succeeded])

	if info.FailedStep.Name != "" || info.FailedStep.ExitCode != 0 {
		fmt.Fprintf(&b, "- Failed Step: %s (Exit Code: %d)\n", info.FailedStep.Name, info.FailedStep.ExitCode)
	}

	if (info.Error != types.ErrorInfo{}) {
		fmt.Fprintf(&b, "- Error Type: %s\n", info.Error.Type)
		fmt.Fprintf(&b, "- Error Reason: %s\n", info.Error.Reason)
		if m := strings.TrimSpace(info.Error.Message); m != "" {
			fmt.Fprintf(&b, "- Error Message: %s\n", truncate(m, 600))
		}
	}

	if ls := strings.TrimSpace(info.Error.LogSnippet); ls != "" {
		fmt.Fprintf(&b, "\nRELEVANT LOGS:\n%s\n", truncate(ls, 1200))
	}

	fmt.Fprintf(&b, `
ANALYSIS REQUIREMENTS:
Provide analysis in this exact structure:

[Brief overview of why the job failed, mentioning the primary issues]

Root Cause

[Explanation of what caused this specific issue and why it happens in the context of Tekton/CI/CD]

Solutions

1. [Solution for first issue]
- [Specific actionable step]
- [Another specific step]
- [Example code or configuration change if relevant]

2. [Solution for second issue] 
- [Specific actionable step]
- [Another specific step]
- [Example code or configuration change if relevant]

FORMATTING INSTRUCTIONS:
- Use plain text only, NO markdown symbols (*, **, etc.)
- Use numbered lists for solutions and hyphens for sub-steps
- Include concrete examples and code snippets when relevant
- Focus on immediate fixes and preventive measures
- Keep explanations concise but informative
`)

	return b.String()
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n > 3 {
		return s[:n-3] + "..."
	}
	return s[:n]
}

// buildPipelineRunPrompt creates a prompt for LLM analysis of PipelineRun failures
func BuildPipelineRunPrompt(result *types.PipelineRunDebugInfo) string {
	var prompt strings.Builder

	prompt.WriteString("You are a senior DevOps engineer specializing in Kubernetes, Tekton, and CI/CD pipelines. ")
	prompt.WriteString("Analyze this failed Tekton PipelineRun and provide specific remediation steps.\n\n")

	prompt.WriteString("PIPELINE RUN DETAILS:\n")
	prompt.WriteString(fmt.Sprintf("- PipelineRun: %s/%s\n", result.PipelineRun.Namespace, result.PipelineRun.Name))
	prompt.WriteString(fmt.Sprintf("- Status: %s\n", result.Status.Phase))

	if len(result.Status.Conditions) > 0 {
		prompt.WriteString("\nCONDITIONS:\n")
		for _, cond := range result.Status.Conditions {
			prompt.WriteString(fmt.Sprintf("- %s: %s (%s) - %s\n",
				cond.Type, cond.Status, cond.Reason, cond.Message))
		}
	}

	if len(result.FailedTaskRuns) > 0 {
		prompt.WriteString(fmt.Sprintf("\nFAILED TASKRUNS (%d):\n", len(result.FailedTaskRuns)))
		for _, tr := range result.FailedTaskRuns {
			prompt.WriteString(fmt.Sprintf("- %s: %s - %s\n", tr.Name, tr.Reason, tr.Message))
		}
	} else {
		prompt.WriteString("\nNo TaskRuns were created, indicating a validation or scheduling failure.\n")
	}

	prompt.WriteString(`
ANALYSIS REQUIREMENTS:
Provide analysis in this exact structure:

[Brief overview of why the PipelineRun failed, mentioning the primary issues]

Root Cause

[Explanation of what caused this specific issue in the context of Tekton pipelines]

Solutions

1. [Solution for first issue]
- [Specific actionable step]
- [Another specific step]
- [Example Tekton resource change if relevant]

2. [Solution for second issue] 
- [Specific actionable step]
- [Another specific step]
- [Example Tekton resource change if relevant]

FORMATTING INSTRUCTIONS:
- Use plain text only, NO markdown symbols (*, **, etc.)
- Use numbered lists for solutions and hyphens for sub-steps
- Include concrete examples and YAML snippets when relevant
- Focus on pipeline configuration, resource constraints, and dependencies
- Keep explanations concise but informative
`)

	return prompt.String()
}
