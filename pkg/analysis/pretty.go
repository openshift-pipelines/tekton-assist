package analysis

import (
	"fmt"
	"strings"

	"github.com/openshift-pipelines/tekton-assist/pkg/types"
)

// RenderPrettyReport builds a human-readable demo report from TaskRunDebugInfo and analysis text.
// This does not change any server behavior; it is intended for ad-hoc/demo output.
func RenderPrettyReport(info types.TaskRunDebugInfo, analysisText string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Tekton TaskRun Failure Report**\n")
	fmt.Fprintf(&b, "**TaskRun:** %s\n", valueOrDash(info.TaskRun))
	fmt.Fprintf(&b, "**Namespace:** %s\n", valueOrDash(info.Namespace))
	if info.Succeeded {
		fmt.Fprintf(&b, "**Succeeded:** ✅ Yes\n")
	} else {
		fmt.Fprintf(&b, "**Succeeded:** ❌ No\n")
	}
	if info.FailedStep.Name != "" {
		fmt.Fprintf(&b, "**Failed Step:** %s\n", info.FailedStep.Name)
	}
	if info.FailedStep.ExitCode != 0 {
		fmt.Fprintf(&b, "**Exit Code:** %d\n", info.FailedStep.ExitCode)
	}
	fmt.Fprintf(&b, "**Error Details:**\n")
	fmt.Fprintf(&b, "**Type:** %s\n", valueOrDash(info.Error.Type))
	fmt.Fprintf(&b, "**Status:** %s\n", valueOrDash(info.Error.Status))
	fmt.Fprintf(&b, "**Reason:** %s\n", valueOrDash(info.Error.Reason))
	if m := strings.TrimSpace(info.Error.Message); m != "" {
		fmt.Fprintf(&b, "**Message:** %s\n", m)
	}
	if ls := strings.TrimSpace(info.Error.LogSnippet); ls != "" {
		fmt.Fprintf(&b, "**Log Snippet:**\n%s\n", ls)
	}
	fmt.Fprintf(&b, "**Analysis & Suggested Remediation:**\n")
	if strings.TrimSpace(analysisText) != "" {
		fmt.Fprintf(&b, "%s\n", analysisText)
	} else {
		fmt.Fprintf(&b, "(not available)\n")
	}
	return b.String()
}

// RenderPrettyReportANSI prints the same report but with ANSI bold styling for terminals.
func RenderPrettyReportANSI(info types.TaskRunDebugInfo, analysisText string) string {
	const (
		bold  = "\x1b[1m"
		reset = "\x1b[0m"
	)
	var b strings.Builder
	fmt.Fprintf(&b, "%sTekton TaskRun Failure Report%s\n", bold, reset)
	fmt.Fprintf(&b, "%sTaskRun:%s %s\n", bold, reset, valueOrDash(info.TaskRun))
	fmt.Fprintf(&b, "%sNamespace:%s %s\n", bold, reset, valueOrDash(info.Namespace))
	if info.Succeeded {
		fmt.Fprintf(&b, "%sSucceeded:%s ✅ Yes\n", bold, reset)
	} else {
		fmt.Fprintf(&b, "%sSucceeded:%s ❌ No\n", bold, reset)
	}
	if info.FailedStep.Name != "" {
		fmt.Fprintf(&b, "%sFailed Step:%s %s\n", bold, reset, info.FailedStep.Name)
	}
	if info.FailedStep.ExitCode != 0 {
		fmt.Fprintf(&b, "%sExit Code:%s %d\n", bold, reset, info.FailedStep.ExitCode)
	}
	fmt.Fprintf(&b, "%sError Details:%s\n", bold, reset)
	fmt.Fprintf(&b, "%sType:%s %s\n", bold, reset, valueOrDash(info.Error.Type))
	fmt.Fprintf(&b, "%sStatus:%s %s\n", bold, reset, valueOrDash(info.Error.Status))
	fmt.Fprintf(&b, "%sReason:%s %s\n", bold, reset, valueOrDash(info.Error.Reason))
	if m := strings.TrimSpace(info.Error.Message); m != "" {
		fmt.Fprintf(&b, "%sMessage:%s %s\n", bold, reset, m)
	}
	if ls := strings.TrimSpace(info.Error.LogSnippet); ls != "" {
		fmt.Fprintf(&b, "%sLog Snippet:%s\n%s\n", bold, reset, ls)
	}
	fmt.Fprintf(&b, "%sAnalysis & Suggested Remediation:%s\n", bold, reset)
	if strings.TrimSpace(analysisText) != "" {
		fmt.Fprintf(&b, "%s\n", analysisText)
	} else {
		fmt.Fprintf(&b, "(not available)\n")
	}
	return b.String()
}

func valueOrDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}
