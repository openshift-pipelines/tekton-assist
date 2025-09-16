# tkn-assist

`tkn-assist` is a CLI tool that provides AI-powered diagnosis and troubleshooting for Tekton TaskRuns and PipelineRuns. It can be used as a standalone tool or as a plugin for the `tkn` CLI.

## Overview

The `tkn-assist` CLI helps developers and operators quickly identify and resolve issues with failed Tekton executions by:

- Analyzing TaskRun and PipelineRun status, logs, and events
- Using AI to identify root causes of failures
- Providing actionable recommendations for fixing issues
- Supporting multiple output formats (text, JSON, YAML)

## Installation

### As a tkn Plugin

To use `tkn-assist` as a plugin for the `tkn` CLI:

1. Build the binary:
   ```bash
   go build -o tkn-assist ./cmd/tkn-assist
   ```

2. Place the binary in your PATH with the name `tkn-assist`

3. Verify the plugin is available:
   ```bash
   tkn plugin list
   ```

4. Use the plugin:
   ```bash
   tkn assist taskrun diagnose my-failed-taskrun
   ```

### As a Standalone Tool

You can also use `tkn-assist` as a standalone CLI tool:

```bash
go build -o tkn-assist ./cmd/tkn-assist
./tkn-assist taskrun diagnose my-failed-taskrun
```

## Usage

### Basic Commands

```bash
# Diagnose a failed TaskRun
tkn-assist taskrun diagnose my-failed-taskrun

# Diagnose in a specific namespace
tkn-assist taskrun diagnose my-taskrun -n my-namespace

# Get JSON output
tkn-assist taskrun diagnose my-taskrun -o json

# Get YAML output
tkn-assist taskrun diagnose my-taskrun -o yaml

# Use custom API base URL
tkn-assist taskrun diagnose my-taskrun --base-url http://my-api-server:8080

# Set custom timeout
tkn-assist taskrun diagnose my-taskrun --timeout 60s
```

### Global Flags

- `--namespace, -n`: Kubernetes namespace
- `--kubeconfig`: Path to kubeconfig file
- `--context`: Kubernetes context to use
- `--verbose, -v`: Enable verbose output

### TaskRun Diagnosis Options

- `--output, -o`: Output format (text, json, yaml) - default: text
- `--base-url`: Tekton Assistant API base URL - default: http://localhost:8080
- `--timeout`: Timeout for API requests - default: 30s

## Examples

### Diagnose a Failed TaskRun (Text Format)

```bash
tkn-assist taskrun diagnose test-failed-image-pull -n default
```

Output:
```
TaskRun Diagnosis Report
========================

TaskRun: test-failed-image-pull
Namespace: default
Succeeded: ❌ No
Failed Step: failing-image-step
Exit Code: 1

Error Details:
Type: Succeeded
Status: False
Reason: TaskRunImagePullFailed
Message: the step "failing-image-step" in TaskRun "test-failed-image-pull" failed to pull the image "". The pod errored with the message: "Back-off pulling image "nonexistent-registry.example.com/missing-image:v1.0.0": ErrImagePull..."
```

### JSON Output

```bash
tkn-assist taskrun diagnose test-failed-image-pull -n default -o json
```

Output:
```json
{
  "debug": {
    "error": {
      "log_snippet": "the step \"failing-image-step\" in TaskRun...",
      "message": "the step \"failing-image-step\" in TaskRun...",
      "reason": "TaskRunImagePullFailed",
      "status": "False",
      "type": "Succeeded"
    },
    "failed_step": {
      "exit_code": 1,
      "name": "failing-image-step"
    },
    "namespace": "default",
    "succeeded": false,
    "taskrun": "test-failed-image-pull"
  }
}
```

### YAML Output

```bash
tkn-assist taskrun diagnose test-failed-image-pull -n default -o yaml
```

### Verbose Mode

```bash
tkn-assist taskrun diagnose my-taskrun -n default -v
```

Shows additional debugging information:
```
Diagnosing TaskRun: my-taskrun
Namespace: default
Output format: text
Connecting to API at: http://localhost:8080
Calling API: /taskrun/explainFailure?namespace=default&taskrun=my-taskrun
```

### Custom API Server

```bash
tkn-assist taskrun diagnose my-taskrun --base-url https://tekton-assist.my-cluster.com
```

## API Integration

The CLI connects to the Tekton Assistant API server to perform AI-powered analysis. The server:

1. Fetches TaskRun status, logs, and events from Kubernetes
2. Analyzes the failure using AI models
3. Returns structured diagnosis information with recommendations

**API Endpoint**: `GET /taskrun/explainFailure?namespace={namespace}&taskrun={taskrun}`

**Default Base URL**: `http://localhost:8080`

## Requirements

- Kubernetes cluster with Tekton Pipelines installed
- Tekton Assistant API server deployed and accessible
- Network connectivity to the API server

## Configuration

The tool uses standard Kubernetes client configuration:

1. `--kubeconfig` flag
2. `KUBECONFIG` environment variable  
3. In-cluster configuration (when running in a pod)
4. `~/.kube/config` (default)

## Error Handling

The CLI provides clear error messages for common issues:

- **Connection refused**: API server not running or unreachable
- **Timeout**: Request took longer than specified timeout
- **404/500 errors**: TaskRun not found or server errors
- **Authentication errors**: Invalid credentials or permissions

## Output Formats

### Text Format (Default)
Human-readable format with structured sections, emojis, and clear organization.

### JSON Format (`-o json`)
Pretty-printed JSON suitable for:
- Programmatic processing
- Integration with other tools
- Debugging and development

### YAML Format (`-o yaml`)
YAML output for:
- Configuration files
- GitOps workflows
- Human-readable structured data

## Contributing

This tool is part of the Tekton Assistant project. See the main project README for contribution guidelines.

## License

Licensed under the Apache License, Version 2.0. See LICENSE file for details.