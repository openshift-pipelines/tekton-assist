# Tekton Assistant

Tekton Assistant helps explain failed Pipelines/TaskRuns and guide remediation. 

## Tekton Assistant Overview

- **Pipeline Failure Analysis (Explain my failed pipeline)**
  - Retrieves and analyzes logs, status, and events for a failed PipelineRun/TaskRun
  - Identifies the failed step and extracts relevant error messages
  - Produces a concise summary (e.g., "Step 'build' failed due to missing dependency X")
  - Suggests actionable fixes (e.g., permissions, image pull errors)
  - Examples: "Check if secret 'docker-creds' exists in namespace Y", "Verify registry authentication"

## Using Gemini LLM

### Build
```
go build ./cmd/diagnose
```

### Usage
```
OPENAI_API_KEY="$GEMINI_API_KEY" ./diagnose serve \
  --openai-base-url "https://generativelanguage.googleapis.com/v1beta/openai/" \
  --openai-model "gemini-2.5-flash" --debug
```
### Test
```
curl -s "http://localhost:8080/taskrun/explainFailure?namespace=default&taskrun_name=pipelinerun-go-golangci-lint" | jq
```
