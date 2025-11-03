# AgentRun Controller

⚠️ **Status: Proof of Concept** - Not production ready

Kubernetes controller that runs LLM-powered agents to execute natural language goals against your cluster.

## Prerequisites

- Kubernetes 1.27+
- Claude API key
- Tekton Pipelines (for example)

## Quick Start

```bash
# 1. Install CRDs and controller
make install
make deploy

# 2. Install Tekton (required for example)
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/buildpacks/0.6/buildpacks.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.9/git-clone.yaml

# 3. Create Claude API key secret
kubectl create secret generic claude-api-key \
  --from-literal=CLAUDE_API_KEY='sk-ant-api03-YOUR_KEY_HERE'

# 4. Deploy example config and RBAC
kubectl apply -f examples/claude-pipelinerun-agent/02-config-pvc.yaml
kubectl apply -f examples/claude-pipelinerun-agent/04-rbac.yaml
kubectl apply -f examples/claude-pipelinerun-agent/03-agentconfig.yaml
kubectl apply -f examples/claude-pipelinerun-agent/06-sample-pipeline.yaml

# 5. Run an agent
kubectl apply -f examples/claude-pipelinerun-agent/05-agentrun.yaml

# 6. Watch it work
kubectl get agentrun -w
kubectl logs -f -l agent.tekton.dev/agentrun=create-pipelinerun-example
```

## What it does

Creates an agent pod that:
1. Reads your natural language goal
2. Plans actions using Claude
3. Executes Kubernetes tools (get resources, get logs, create PipelineRuns)
4. Validates all actions with OPA policies
5. Reports results

## Example Goal

```yaml
apiVersion: agent.tekton.dev/v1alpha1
kind: AgentRun
metadata:
  name: create-pipelinerun-example
spec:
  configRef:
    name: pipeline-agent
  goal: |
    Create a PipelineRun for building a container image.
    Use the 'buildpacks' Pipeline with these parameters:
    - image: docker.io/myorg/myapp:v1.0.0
    - source-url: https://github.com/myorg/myapp
```

## Architecture

See [claude.md](./claude.md) for detailed design and roadmap.

## License

Apache 2.0
