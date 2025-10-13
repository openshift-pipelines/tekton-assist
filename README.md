# Tekton Assistant

Tekton Assistant helps explain failed Pipelines/TaskRuns and guide remediation. 

## Tekton Assistant Overview

- **Pipeline Failure Analysis (Explain my failed pipeline)**
  - Retrieves and analyzes logs, status, and events for a failed PipelineRun/TaskRun
  - Identifies the failed step and extracts relevant error messages
  - Produces a concise summary (e.g., "Step 'build' failed due to missing dependency X")
  - Suggests actionable fixes (e.g., permissions, image pull errors)
  - Examples: "Check if secret 'docker-creds' exists in namespace Y", "Verify registry authentication"

## CLI (tkn-assist) Usage

Build locally:
```
make build
./bin/tkn-assist --help
```

Diagnose a TaskRun using Lightspeed service:
```
./bin/tkn-assist taskrun diagnose <taskrun-name> -n <namespace> \
  --lightspeed-url https://localhost:8443 -k \
  --token <BEARER_TOKEN>     # or export LIGHTSPEED_TOKEN
```

Diagnose a PipelineRun:
```
./bin/tkn-assist pipelinerun diagnose <pipelinerun-name> -n <namespace> \
  --lightspeed-url https://localhost:8443 -k
```

Notes:
- Use `-o json` or `-o yaml` for machine-readable output.
- The CLI renders Summary, Analysis, Solutions (if present), References, and Token usage.
- Token resolution order: `--token`, `--token-file`, kubeconfig token, `LIGHTSPEED_TOKEN`.

Build container image with ko:
```
export KO_DOCKER_REPO=ghcr.io/your-org
make image
```
