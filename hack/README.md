
HTTP check:
List tool groups: curl -s http://localhost:8321/v1/toolgroups | jq
Get the RAG group: curl -s http://localhost:8321/v1/toolgroups/builtin::rag | jq
List tools in a group (if supported): curl -s "http://localhost:8321/v1/tool-runtime/list-tools?tool_group_id=builtin::rag" | jq

➜  tekton-assistant git:(main) ✗ curl -s http://localhost:8321/v1/toolgroups | jq
{
  "data": [
    {
      "identifier": "builtin::rag",
      "provider_resource_id": "builtin::rag",
      "provider_id": "rag-runtime",
      "type": "tool_group",
      "mcp_endpoint": null,
      "args": null
    },
    {
      "identifier": "builtin::websearch",
      "provider_resource_id": "builtin::websearch",
      "provider_id": "tavily-search",
      "type": "tool_group",
      "mcp_endpoint": null,
      "args": null
    }
  ]
}
