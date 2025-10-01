# Tekton Assistant

Tekton Assistant helps explain failed Pipelines/TaskRuns and guide remediation. 

## Tekton Assistant Overview

- **Pipeline Failure Analysis (Explain my failed pipeline)**
  - Retrieves and analyzes logs, status, and events for a failed PipelineRun/TaskRun
  - Identifies the failed step and extracts relevant error messages
  - Produces a concise summary (e.g., "Step 'build' failed due to missing dependency X")
  - Suggests actionable fixes (e.g., permissions, image pull errors)
  - Examples: "Check if secret 'docker-creds' exists in namespace Y", "Verify registry authentication"

## Flow chart
The following diagram illustrates the request flow for the tekton-assist service:

```mermaid
graph TD
    A[Client Request] --> B[GET /taskrun/explainFailure?taskrun=&namespace=]
    B --> C{Validate Parameters}
    C -->|Missing params| D[Return HTTP 400 Error]
    C -->|Valid params| E[Create Inspector Instance]
    
    E --> F[Inspect TaskRun<br/>namespace, taskrunName]
    F --> G{Fetch TaskRun Details}
    G -->|Success| H[Get TaskRun Result]
    G -->|Failure| I[Return HTTP 500 Error]
    
    H --> J{LLM Available?}
    J -->|Yes| K[Build LLM Prompt<br/>with TaskRun data]
    K --> L[Call LLM.Analyze<br/>with 45s timeout]
    L --> M{LLM Analysis Success?}
    M -->|Yes| N[Get Analysis Text]
    M -->|No| O[Log Error, Store Error Message]
    J -->|No| P[Skip LLM Analysis]
    
    N --> Q[Prepare JSON Response]
    O --> Q
    P --> Q
    
    Q --> R[Encode Response JSON]
    R --> S[Return HTTP 200 Response]
    
    style A fill:#e1f5fe
    style S fill:#e8f5e8
    style D fill:#ffebee
    style I fill:#ffebee

```

## Using Gemini LLM

### Build for local testing
```
go build ./cmd/diagnose
```

### Use command line for local testing
```
OPENAI_API_KEY="$GEMINI_API_KEY" go run ./cmd/diagnose serve \
  --openai-base-url "https://generativelanguage.googleapis.com/v1beta/openai/" \
  --openai-model "gemini-2.5-flash" --debug
```

### Deploy for local testing
- Create the openai secret
```
kubectl create secret generic openai-api-key --from-literal=openai-api-key=xxx -n openshift-pipelines
```

- deploy on kind cluster
```
KO_DOCKER_REPO=kind.local make apply
```

-deploy on openshift cluster
```
KO_DOCKER_REPO=ttl.sh make apply
```

### Developer convenience
```
# Run e2e tests against kind using the in-cluster mock OpenAI server
KO_DOCKER_REPO=kind.local go test -v ./test/...
```

### Test the deployment

- Use tekton-assist command line tool to diagnose a taskrun
```
go run ./cmd/tkn-assist taskrun diagnose pipelinerun-go-golangci-lint
```

- Use curl to diagnose a taskrun
```
curl -s "http://localhost:8080/taskrun/explainFailure?namespace=default&name=pipelinerun-go-golangci-lint" | jq
```
