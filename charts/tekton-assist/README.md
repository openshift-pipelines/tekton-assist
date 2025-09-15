Tekton Assist Helm Chart

This chart is used to deploy Tekton Assist.

## Usage

```
helm repo add tektoncd "git+https://github.com/openshift-pipelines/tekton-assist@charts?ref=main"
helm install tekton-assist tektoncd/tekton-assist
-n openshift-pipelines
--set tekton-assist.openai.apiKey=$GEMINI_API_KEY
--set tekton-assist.openai.provider=gemini
--set tekton-assist.openai.model=gemini-2.5-flash
--set tekton-assist.openai.baseURL=https://generativelanguage.googleapis.com/v1beta/openai/
--set tekton-assist.openai.temperature=0.2
--set tekton-assist.openai.maxTokens=400
--set tekton-assist.openai.timeout=30s
--set tekton-assist.openai.debug=false
 
```

## Test the deployment

```
curl -s "http://localhost:8080/taskrun/explainFailure?namespace=default&taskrun=pipelinerun-go-golangci-lint" | jq
```

## Uninstall

```
helm uninstall tekton-assist
```
