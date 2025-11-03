# Claude-Powered PipelineRun Agent Example

This example demonstrates how to use AgentRun to create Tekton PipelineRuns based on natural language intent, powered by Claude.

## Overview

The agent analyzes your goal, examines available Pipelines in the cluster, and creates appropriate PipelineRuns with the correct parameters. This example includes:

- **AgentConfig**: Configuration for the agent runtime
- **AgentRun**: Example goals for the agent to achieve
- **OPA Policy**: Security policy allowing PipelineRun creation
- **RBAC**: Permissions for the agent to interact with Tekton resources
- **Sample Pipelines**: Example Pipelines that the agent can run

## Prerequisites

1. **Kubernetes cluster** (kind, minikube, or real cluster)
2. **Tekton Pipelines** installed:
   ```bash
   kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
   ```
3. **agentrun-controller** installed
4. **Claude API key** from Anthropic

## Setup Instructions

### 1. Install Tekton Pipelines

```bash
# Install Tekton Pipelines CRDs and controller
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml

# Wait for Tekton to be ready
kubectl wait --for=condition=ready pod -l app=tekton-pipelines-controller -n tekton-pipelines --timeout=120s
```

### 2. Create the Claude API Key Secret

```bash
# Replace with your actual Claude API key
kubectl create secret generic claude-api-key \
  --from-literal=CLAUDE_API_KEY='sk-ant-api03-YOUR_KEY_HERE' \
  -n default
```

Or apply the secret YAML after editing it:

```bash
# Edit 00-secret.yaml with your API key
vim 00-secret.yaml

# Apply it
kubectl apply -f 00-secret.yaml
```

### 3. Create the Config PVC and Copy Files

```bash
# Create the PVC
kubectl apply -f 02-config-pvc.yaml

# Create a temporary pod to copy files
kubectl run config-copier --image=busybox --restart=Never --rm -i \
  --overrides='
{
  "spec": {
    "volumes": [{
      "name": "config",
      "persistentVolumeClaim": {"claimName": "agent-config-pvc"}
    }],
    "containers": [{
      "name": "config-copier",
      "image": "busybox",
      "command": ["sh"],
      "volumeMounts": [{
        "name": "config",
        "mountPath": "/workspace/config"
      }]
    }]
  }
}' \
  -- sh -c '
mkdir -p /workspace/config/prompts /workspace/config/guardrails
cat > /workspace/config/prompts/system.txt << "EOF"
You are an intelligent Kubernetes agent that helps users manage their Tekton Pipelines.

Your role is to:
1. Understand the user'\''s intent about what Pipeline they want to run
2. Analyze the available Pipelines in the cluster
3. Create appropriate PipelineRuns with the correct parameters

Available tools:
- k8s_get_resources: List Kubernetes resources (pods, deployments, services, pipelines)
- k8s_get_logs: Get logs from pods
- tekton_create_pipelinerun: Create a Tekton PipelineRun

When creating PipelineRuns:
- Always use descriptive names with timestamps or unique identifiers
- Pass appropriate parameters based on the Pipeline'\''s parameter schema
- Configure workspaces if the Pipeline requires them

Be concise and action-oriented.
EOF
cat > /workspace/config/guardrails/policy.rego << "EOF"
package agent.tools
default allow = false
allow { input.tool == "k8s_get_resources"; input.namespace == "default" }
allow { input.tool == "k8s_get_logs"; input.namespace == "default" }
allow { input.tool == "tekton_create_pipelinerun"; input.namespace == "default" }
EOF
ls -la /workspace/config/prompts/
ls -la /workspace/config/guardrails/
'
```

### 4. Apply RBAC and AgentConfig

```bash
# Create ServiceAccount and RBAC for the agent
kubectl apply -f 04-rbac.yaml

# Create AgentConfig
kubectl apply -f 03-agentconfig.yaml

# Optional: Create sample Pipelines for testing
kubectl apply -f 06-sample-pipeline.yaml
```

### 5. Run the Agent

```bash
# Create an AgentRun to have the agent create a PipelineRun
kubectl apply -f 05-agentrun.yaml

# Watch the AgentRun status
kubectl get agentrun -w

# Check the agent pod logs
kubectl logs -l agent.tekton.dev/agentrun=create-build-pipeline -f

# Verify the PipelineRun was created
kubectl get pipelineruns
```

## How It Works

1. **User creates an AgentRun** with a natural language goal (e.g., "Create a PipelineRun for building my app")

2. **Controller creates an agent pod** with:
   - Claude LLM integration
   - Access to Kubernetes API (via RBAC)
   - OPA policy enforcement
   - System prompts from the config PVC

3. **Agent executes plan-act-reflect loop**:
   - **Plan**: Understand the goal and determine what information is needed
   - **Act**: Use tools to gather information (list Pipelines) and create PipelineRuns
   - **Reflect**: Check if goal is achieved or if more actions are needed

4. **OPA enforces security**:
   - Only allows tools specified in the policy
   - Restricts namespace access
   - Validates tool inputs

5. **PipelineRun is created** based on the agent's analysis

## Example Goals

### Simple Pipeline Execution
```yaml
goal: "Run the buildpacks Pipeline to build docker.io/myorg/app:v2.0.0 from https://github.com/myorg/app"
```

### Conditional Execution
```yaml
goal: |
  Check if there are any failed PipelineRuns in the last hour.
  If yes, create a new PipelineRun of the same Pipeline with the same parameters.
```

### Parameterized Deployment
```yaml
goal: |
  Deploy the myapp application to the staging namespace with 2 replicas.
  Use the deploy Pipeline if it exists, otherwise tell me what Pipelines are available.
```

## Customization

### Modify the System Prompt

Edit the ConfigMap in `02-config-pvc.yaml` to change how the agent behaves:

```yaml
data:
  system.txt: |
    You are a specialized CI/CD agent...
    [your custom instructions]
```

### Adjust OPA Policy

Edit `01-policy.rego` to add more restrictions or allow additional tools:

```rego
# Allow only specific Pipeline names
allow {
    input.tool == "tekton_create_pipelinerun"
    input.pipelineName in ["buildpacks", "deploy", "test"]
}
```

### Change LLM Provider

Edit the AgentConfig to use Gemini instead:

```yaml
spec:
  provider: gemini
```

Then create a secret with `GEMINI_API_KEY` instead.

## Troubleshooting

### Agent pod fails to start
```bash
# Check pod events
kubectl describe pod -l agent.tekton.dev/agentrun=create-build-pipeline

# Common issues:
# - Missing Claude API key secret
# - RBAC permissions not configured
# - Config PVC not mounted correctly
```

### Agent cannot create PipelineRuns
```bash
# Check agent logs for policy violations
kubectl logs -l agent.tekton.dev/agentrun=create-build-pipeline

# Verify RBAC
kubectl auth can-i create pipelineruns --as=system:serviceaccount:default:pipeline-agent-sa
```

### OPA policy denies everything
```bash
# Check policy file in PVC
kubectl exec -it <agent-pod> -- cat /workspace/config/guardrails/policy.rego

# Test policy locally with opa CLI
opa eval -d policy.rego -i input.json 'data.agent.tools.allow'
```

## Cleanup

```bash
# Delete AgentRuns
kubectl delete agentrun --all

# Delete AgentConfig
kubectl delete agentconfig pipeline-agent-config

# Delete RBAC
kubectl delete -f 04-rbac.yaml

# Delete PVC
kubectl delete pvc agent-config-pvc

# Delete secret
kubectl delete secret claude-api-key
```

## Security Considerations

- **API Key Protection**: Claude API key is stored in a Kubernetes Secret, never in config files
- **RBAC Least Privilege**: Agent only has permissions to read resources and create PipelineRuns
- **OPA Policy Enforcement**: All tool calls are validated before execution
- **Network Policy**: Agent pod network access is restricted (if networkPolicy: strict)
- **Read-only Config**: System prompts and policies are mounted read-only

## Next Steps

- Add more sophisticated goals combining multiple Pipelines
- Implement pre-hooks for additional validation
- Use post-hooks for audit logging
- Integrate with Tekton Triggers for event-driven agent execution
- Add vector database for learning from past PipelineRun executions
