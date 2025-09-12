package analysis

import (
	"fmt"
	"strings"

	"github.com/openshift-pipelines/tekton-assist/pkg/types"
)

// BuildTaskRunPrompt creates a concise user prompt for the LLM from TaskRunDebugInfo.
func BuildTaskRunPrompt(info types.TaskRunDebugInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Analyze this Tekton TaskRun failure and propose fixes.\n")
	fmt.Fprintf(&b, "Provide: root cause, likely failing component, and concrete remediation steps.\n\n")
	fmt.Fprintf(&b, "Context:\n")
	fmt.Fprintf(&b, "- TaskRun: %s\n", info.TaskRun)
	fmt.Fprintf(&b, "- Namespace: %s\n", info.Namespace)
	if info.Succeeded {
		fmt.Fprintf(&b, "- Succeeded: true\n")
	} else {
		fmt.Fprintf(&b, "- Succeeded: false\n")
	}
	if info.FailedStep.Name != "" || info.FailedStep.ExitCode != 0 {
		fmt.Fprintf(&b, "- Failed Step: %s (exitCode=%d)\n", info.FailedStep.Name, info.FailedStep.ExitCode)
	}
	if (info.Error != types.ErrorInfo{}) {
		fmt.Fprintf(&b, "- Error: type=%s status=%s reason=%s\n", info.Error.Type, info.Error.Status, info.Error.Reason)
		if m := strings.TrimSpace(info.Error.Message); m != "" {
			fmt.Fprintf(&b, "- Message: %s\n", truncate(m, 600))
		}
		if ls := strings.TrimSpace(info.Error.LogSnippet); ls != "" {
			fmt.Fprintf(&b, "- Log Snippet:\n%s\n", truncate(ls, 1200))
		}
	}
	fmt.Fprintf(&b, "\nConstraints:\n- Be precise and brief.\n- Output 3-6 bullet points.\n")
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
