# agentrun-controller

> Kubernetes-native agentic platform built following tektoncd/pipeline conventions
>
> **Commit Attribution**: All commits authored by human contributors

## Project Mindset

This project follows tektoncd/pipeline patterns closely:
- Directory structure mirrors Tekton's layout
- Code generation using k8s.io/code-generator
- Knative injection for dependency management
- Table-driven testing with fake clients
- Security-first pod configuration
- ConfigMap-based feature flags

**MVP Philosophy**: Start standalone, add Tekton integration later
- No CustomRun dependency initially (AgentRun → Pod directly)
- No webhooks (async validation in reconciler)
- Embedded OPA (not sidecar)
- Single container (not multi-container)

## Architecture

### Core Resources

**AgentConfig** (CRD: agent.tekton.dev/v1alpha1)
```yaml
spec:
  serviceAccount: string
  configPVC: string           # Pre-created PVC with prompts, schemas, policies
  maxIterations: int32        # Default: 3, max: 10
  timeout: duration           # Default: 8m
  preHooks: []string          # Hook pod names
  postHooks: []string
  policy:
    opa: string              # strict|permissive
  networkPolicy: string       # strict|permissive
```

**AgentRun** (CRD: agent.tekton.dev/v1alpha1)
```yaml
spec:
  configRef:
    name: string
  goal: string
  context:
    hints: []string
status:
  conditions: []Condition
  phase: string              # Pending|PreHooks|Acting|Reflecting|PostHooks|Succeeded|Failed
  startTime: time
  completionTime: time
  iterations: int32
  results: []AgentResult
```

### Execution Flow

```
1. User creates AgentRun CR
2. Controller reconciles:
   - Phase: Pending → validate AgentConfig exists, create data PVC
   - Phase: PreHooks → launch hook pods, wait for completion
   - Phase: Acting → create agent pod with config+data PVC mounts
   - Phase: Reflecting → wait for pod completion, collect results
   - Phase: PostHooks → launch hook pods for audit
   - Phase: Succeeded/Failed → update status, emit events
3. Garbage collection via owner references
```

### Agent Pod Structure

```yaml
Pod:
  metadata:
    ownerReferences:
      - agentrun (controller: true)
    labels:
      agent.tekton.dev/agentrun: <name>
      agent.tekton.dev/config: <config>
  spec:
    serviceAccountName: <from AgentConfig>
    restartPolicy: Never
    securityContext:
      runAsNonRoot: true
      runAsUser: 65532
      fsGroup: 65532
      seccompProfile:
        type: RuntimeDefault
    containers:
      - name: agent
        image: agentrun-runtime:latest
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop: [ALL]
        volumeMounts:
          - name: config
            mountPath: /workspace/config
            readOnly: true
          - name: data
            mountPath: /workspace/data
          - name: secrets
            mountPath: /workspace/secrets
            readOnly: true
    volumes:
      - name: config
        persistentVolumeClaim:
          claimName: <from AgentConfig>
      - name: data
        persistentVolumeClaim:
          claimName: <generated per AgentRun>
      - name: secrets
        secret:
          secretName: <LLM API keys>
```

### Agent Runtime (inside pod)

**Plan-Act-Reflect Loop**
```go
for iteration := 0; iteration < maxIterations; iteration++ {
    // Plan phase
    planPrompt := buildPrompt(systemPrompt, goal, context, previousResults)
    planResponse := llmProvider.Call(planPrompt)

    // Act phase
    for _, toolCall := range parseToolCalls(planResponse) {
        if !opa.Allow(toolCall) {
            return error("policy violation")
        }

        result := tools.Execute(toolCall)
        results = append(results, result)

        if len(result) > maxOutputSize {
            result = truncate(result)
        }
    }

    // Reflect phase
    reflectPrompt := buildReflectPrompt(goal, results)
    reflectResponse := llmProvider.Call(reflectPrompt)

    if reflectResponse.confident {
        return results
    }
}
```

### Security Model

**Controller RBAC** (ClusterRole)
```yaml
rules:
  - apiGroups: [agent.tekton.dev]
    resources: [agentconfigs, agentruns]
    verbs: [get, list, watch]
  - apiGroups: [agent.tekton.dev]
    resources: [agentruns/status]
    verbs: [get, update, patch]
  - apiGroups: [""]
    resources: [pods, pods/log]
    verbs: [get, list, create, delete, watch]
  - apiGroups: [""]
    resources: [persistentvolumeclaims]
    verbs: [get, list, create, delete]
  - apiGroups: [""]
    resources: [secrets, serviceaccounts]
    verbs: [get, list]
  - apiGroups: [rbac.authorization.k8s.io]
    resources: [roles, rolebindings]
    verbs: [get, create, delete]
  - apiGroups: [networking.k8s.io]
    resources: [networkpolicies]
    verbs: [get, create, delete]
  - apiGroups: [""]
    resources: [events]
    verbs: [create, update, patch]
```

**Agent RBAC** (generated Role per AgentRun)
```yaml
rules:
  - apiGroups: [""]
    resources: [pods, services, endpoints, events]
    verbs: [get, list, watch]
  - apiGroups: [""]
    resources: [pods/log]
    verbs: [get, list]
  - apiGroups: [apps]
    resources: [deployments, replicasets, statefulsets, daemonsets]
    verbs: [get, list, watch]
```

**NetworkPolicy** (generated per AgentRun)
```yaml
spec:
  podSelector:
    matchLabels:
      agent.tekton.dev/agentrun: <name>
  policyTypes: [Ingress, Egress]
  ingress: []  # deny all
  egress:
    - to:  # Kubernetes API
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
    - to:  # DNS
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53
    - to:  # LLM endpoints (from AgentConfig)
        - podSelector: {}
      ports:
        - protocol: TCP
          port: 443
```

**OPA Policy** (embedded in agent runtime)
```rego
package agent.tools

default allow = false

allow {
    input.tool == "k8s_get_resources"
    input.namespace in data.allowed_namespaces
}

allow {
    input.tool == "k8s_get_logs"
    input.namespace in data.allowed_namespaces
}

deny[msg] {
    input.tool == "k8s_get_resources"
    count(input.labelSelector) > 0
    not valid_label_selector(input.labelSelector)
    msg := "invalid label selector"
}
```

### Tools (MVP)

All tools implemented in pkg/tools/ as Go packages.

**k8s_get_resources**
```json
{
  "name": "k8s_get_resources",
  "schema": {
    "namespace": "string (required)",
    "resourceType": "enum[pods,deployments,services,replicasets] (required)",
    "labelSelector": "string (optional)",
    "limit": "int (default: 100, max: 100)"
  }
}
```

**k8s_get_logs**
```json
{
  "name": "k8s_get_logs",
  "schema": {
    "namespace": "string (required)",
    "pod": "string (required)",
    "container": "string (optional)",
    "tailLines": "int (default: 500, max: 500)",
    "sinceSeconds": "int (default: 900, max: 900)"
  }
}
```

**k8s_describe** (Phase 5)
```json
{
  "name": "k8s_describe",
  "schema": {
    "namespace": "string (required)",
    "resourceType": "string (required)",
    "name": "string (required)"
  },
  "note": "Redacts Secret data from output"
}
```

**k8s_events** (Phase 5)
```json
{
  "name": "k8s_events",
  "schema": {
    "namespace": "string (required)",
    "sinceSeconds": "int (default: 7200, max: 7200)"
  }
}
```

### Observability

**Structured Logging** (Knative zap-based)
```go
logger.Infow("Reconciling AgentRun",
    "namespace", ar.Namespace,
    "name", ar.Name,
    "phase", ar.Status.Phase,
    "iteration", iteration)
```

**Metrics** (Prometheus)
```
agentrun_total{status="succeeded|failed|timeout"}
agentrun_duration_seconds{phase="prehooks|acting|reflecting|posthooks"}
agentrun_iterations{config="<name>"}
agentrun_tool_calls_total{tool="<name>"}
agentrun_llm_tokens_total{provider="claude|gemini",direction="in|out"}
```

**Traces** (OpenTelemetry)
```
Span: ReconcileAgentRun
  ├─ Span: PreHooks
  ├─ Span: Acting
  │   ├─ Span: Plan (iteration=0)
  │   ├─ Span: Tool.k8s_get_resources
  │   ├─ Span: Tool.k8s_get_logs
  │   └─ Span: Reflect (iteration=0)
  └─ Span: PostHooks
```

## Directory Structure

Following tektoncd/pipeline conventions:

```
agentrun-controller/
├── cmd/
│   ├── controller/              # Main controller binary
│   │   └── main.go
│   └── agent/                   # Agent runtime binary (runs in pod)
│       └── main.go
├── pkg/
│   ├── apis/
│   │   ├── agent/
│   │   │   └── v1alpha1/
│   │   │       ├── doc.go                    # Package docs + markers
│   │   │       ├── register.go               # Scheme registration
│   │   │       ├── agentconfig_types.go
│   │   │       ├── agentconfig_defaults.go
│   │   │       ├── agentconfig_validation.go
│   │   │       ├── agentrun_types.go
│   │   │       ├── agentrun_defaults.go
│   │   │       ├── agentrun_validation.go
│   │   │       └── zz_generated.deepcopy.go  # Generated
│   │   └── config/
│   │       ├── config.go                     # Config struct
│   │       ├── feature_flags.go
│   │       ├── defaults.go
│   │       └── store.go                      # ConfigStore
│   ├── client/                                # Generated (DO NOT EDIT)
│   │   ├── clientset/
│   │   ├── informers/
│   │   ├── listers/
│   │   └── injection/
│   ├── reconciler/
│   │   └── agentrun/
│   │       ├── controller.go                 # Controller setup
│   │       ├── agentrun.go                   # ReconcileKind
│   │       └── agentrun_test.go
│   ├── agent/                                # Agent runtime logic
│   │   ├── loop.go                           # Plan-act-reflect
│   │   ├── provider.go                       # LLM interface
│   │   └── opa.go                            # Policy enforcement
│   ├── tools/                                # Tool implementations
│   │   ├── registry.go
│   │   ├── k8s/
│   │   │   ├── get_resources.go
│   │   │   ├── get_logs.go
│   │   │   ├── describe.go
│   │   │   └── events.go
│   │   └── limiter.go                        # Output size caps
│   ├── providers/
│   │   ├── claude/
│   │   │   └── client.go
│   │   └── gemini/
│   │       └── client.go
│   ├── security/
│   │   ├── rbac.go                           # Role generator
│   │   ├── networkpolicy.go                  # NetworkPolicy generator
│   │   └── redaction.go                      # PII scrubber
│   └── pod/
│       └── builder.go                        # Pod spec builder
├── config/
│   ├── 100-namespace.yaml
│   ├── 200-clusterrole.yaml
│   ├── 200-serviceaccount.yaml
│   ├── 201-clusterrolebinding.yaml
│   ├── 300-agentconfig.yaml                  # CRD
│   ├── 300-agentrun.yaml                     # CRD
│   └── 400-controller.yaml                   # Deployment
├── test/
│   ├── controller.go                         # Test utilities
│   ├── testdata/                             # Fixtures
│   └── e2e/
│       └── agentrun_test.go
├── hack/
│   ├── update-codegen.sh                     # Code generation
│   ├── update-deps.sh
│   └── boilerplate.go.txt
├── docs/
│   ├── installation.md
│   ├── agentconfig.md
│   ├── agentrun.md
│   ├── security.md
│   └── tools.md
├── examples/
│   ├── agentconfig-sample.yaml
│   ├── agentrun-sample.yaml
│   └── prompts/
├── charts/
│   └── agentrun-controller/
├── Makefile
├── go.mod
├── LICENSE
└── README.md
```

## Development Phases

### Phase 0: Foundation

**CRD Definitions**
- `api/v1alpha1/agentconfig_types.go`
  - Spec: serviceAccount, configPVC, maxIterations, timeout, preHooks, postHooks
  - Status: conditions
- `api/v1alpha1/agentrun_types.go`
  - Spec: configRef, goal, context
  - Status: conditions, phase, startTime, completionTime, iterations, results
- `*_validation.go` for both (use apis.FieldError)
- `*_defaults.go` for both (context-based)
- Kubebuilder markers: +genclient, +k8s:deepcopy-gen, +optional

**Code Generation**
- `hack/update-codegen.sh` using k8s.io/code-generator
- Generate: deepcopy, client, informer, lister
- Knative injection generation
- Add to Makefile: `make generated`
- Commit generated code

**RBAC**
- Controller ClusterRole (see Security Model)
- ServiceAccount
- ClusterRoleBinding
- Agent Role template in pkg/security/rbac.go

**Project Setup**
- Go module initialized
- Directory structure created
- LICENSE, .gitignore, Makefile
- go.mod with dependencies:
  - k8s.io/client-go
  - k8s.io/apimachinery
  - knative.dev/pkg
  - github.com/open-policy-agent/opa

### Phase 1: Core Loop

**Controller**
- `cmd/controller/main.go` entry point
- `pkg/reconciler/agentrun/controller.go`
  - Knative injection setup
  - Watch AgentRuns, Pods
  - ConfigStore for feature flags
- `pkg/reconciler/agentrun/agentrun.go`
  - ReconcileKind implementation
  - State machine: check Done → Cancelled → TimedOut → Reconcile
  - Phase transitions with status updates
  - Event emission on changes
  - DeepCopy before state for comparison

**Pod Builder**
- `pkg/pod/builder.go`
  - Build pod spec from AgentRun + AgentConfig
  - Apply security contexts
  - Owner references (AgentRun as controller)
  - Volume mounts (config PVC read-only, data PVC, secret)
  - Labels for NetworkPolicy

**Agent Runtime**
- `cmd/agent/main.go` entry point (runs in pod)
- `pkg/agent/loop.go`
  - Plan-act-reflect loop
  - Load system prompt from /workspace/config/prompts/system.txt
  - Max iterations from AgentConfig
  - Timeout enforcement per iteration
  - Parse LLM responses for tool calls

**LLM Provider**
- `pkg/providers/claude/client.go`
  - Read API key from /workspace/secrets/CLAUDE_API_KEY
  - Messages API with streaming
  - Sampling: temp=0.2, top_p=0.3
  - Exponential backoff on errors
  - Token counting

**Tools**
- `pkg/tools/registry.go` (tool lookup)
- `pkg/tools/k8s/get_resources.go`
  - List pods, deployments, services, replicasets
  - Namespace-scoped, max 100 results
  - Label selector support
- `pkg/tools/k8s/get_logs.go`
  - Tail max 500 lines, since ≤ 15min
  - Container selection
- `pkg/tools/limiter.go` (output size caps)

**OPA Policy**
- `pkg/agent/opa.go`
  - Embed OPA library
  - Load policies from /workspace/config/guardrails/*.rego
  - Validate tool calls before execution
  - Fail closed on policy errors

**Testing**
- `test/controller.go`
  - SeedTestData() helper
  - Fake client setup
- `pkg/reconciler/agentrun/agentrun_test.go`
  - Table-driven tests
  - Test cases: successful run, timeout, policy violation, LLM error

**Config Management**
- `pkg/apis/config/config.go`
  - Config struct with FeatureFlags, Defaults
- `pkg/apis/config/store.go`
  - ConfigStore with ConfigMap watching
- ConfigMap: agentrun-config
  ```yaml
  data:
    feature-flags: |
      enable-webhooks: false
    defaults: |
      default-service-account: "default"
      default-timeout: "8m"
  ```

### Phase 2: Security Hardening

**PreHook: llm-security-check**
- `pkg/reconciler/agentrun/prehooks.go`
  - Create Pod running security check
  - Wait for completion before agent pod
- Simple Go binary:
  - Scan prompts for blocklisted patterns
  - Verify tool schemas
  - Check RBAC consistency
  - Output validation report to /workspace/data/prehook-report.json

**PostHook: audit-bundle**
- `pkg/reconciler/agentrun/posthooks.go`
  - Create Pod collecting audit data
  - Wait for completion after agent pod
- Simple Go binary:
  - Read agent logs
  - Collect tool calls from /workspace/data/tool-calls.json
  - Collect policy decisions from /workspace/data/policy-decisions.json
  - Generate summary
  - Write to /workspace/data/audit-bundle.json

**NetworkPolicy Generation**
- `pkg/security/networkpolicy.go`
  - GenerateNetworkPolicy(agentRun, agentConfig)
  - Default deny + egress whitelist
  - Create in reconciler before agent pod
  - Delete via owner references

**RBAC Generation**
- `pkg/security/rbac.go`
  - GenerateRole(agentRun, agentConfig)
  - Read-only verbs only
  - Scoped to safe resources
  - Create RoleBinding to ServiceAccount from AgentConfig
  - Delete via owner references

**Prompt Redaction**
- `pkg/security/redaction.go`
  - ScrubSecrets() - replace ${SECRET_*} with placeholders
  - ScrubPII() - regex for tokens, IPs, emails
  - Apply to LLM inputs and tool outputs

**Tool Output Caps**
- `pkg/tools/limiter.go`
  - Enforce max output size per tool
  - Truncate with markers
  - Log when caps hit

### Phase 3: Observability

**Structured Logging**
- Use knative.dev/pkg/logging
- Context-aware logger
- Fields: run_id, agent, phase, iteration, tool, latency_ms, tokens_in/out
- Log tool calls with decision_hash, policy_verdict

**Metrics**
- Use knative.dev/pkg/metrics
- Define metrics (see Observability section)
- Record in reconciler and agent runtime
- Expose on :9090/metrics

**Tracing**
- Use go.opentelemetry.io/otel
- Span per reconciliation
- Child spans: PreHooks, Acting (plan, tool, reflect), PostHooks
- Export to in-cluster collector
- Configuration via OTEL_EXPORTER_OTLP_ENDPOINT

**Status Reporting**
- AgentRunStatus.Conditions (Succeeded, Failed, TimedOut)
- AgentRunStatus.Phase tracking
- AgentRunStatus.Results (summary)
- Timestamp all transitions
- Emit Kubernetes events on changes

**Audit Bundle Storage**
- PostHook writes to /workspace/data/audit-bundle.json
- Controller copies to persistent storage (optional)
- Includes: input hashes, tool calls, policy decisions, summary

### Phase 4: Packaging

**Helm Chart**
- `charts/agentrun-controller/Chart.yaml`
- Templates:
  - Namespace
  - CRDs
  - ClusterRole, ServiceAccount, ClusterRoleBinding
  - Deployment (controller)
  - ConfigMap (agentrun-config)
  - Service (metrics)
- Values:
  - image, replicas, resources
  - featureFlags, defaults
- NOTES.txt with quickstart
- Test: helm install --dry-run

**OLM Bundle**
- `bundle/manifests/`
  - ClusterServiceVersion
  - CRDs
  - RBAC
- `bundle/metadata/`
  - annotations.yaml
- `bundle.Dockerfile`
- Build: operator-sdk bundle validate
- Test on OpenShift

**Documentation**
- `docs/installation.md` (minikube, OpenShift, Helm, OLM)
- `docs/agentconfig.md` (CRD reference)
- `docs/agentrun.md` (CRD reference)
- `docs/security.md` (RBAC, NetworkPolicy, OPA)
- `docs/tools.md` (available tools, schemas)
- `examples/` (sample manifests, prompts)

**Release Process**
- Container image build (ko or docker)
- Push to registry
- GitHub release
- Helm chart versioning

### Phase 5: Extensions

**Additional Tools**
- `pkg/tools/k8s/describe.go` (with Secret redaction)
- `pkg/tools/k8s/events.go` (time filtering)
- Update tool registry
- Tests

**Gemini Provider**
- `pkg/providers/gemini/client.go`
- Match Claude's sampling params
- Provider selection in AgentConfig:
  ```yaml
  spec:
    provider: claude|gemini
  ```

**Provider Interface**
- `pkg/agent/provider.go`
  ```go
  type Provider interface {
      Call(ctx context.Context, prompt string) (Response, error)
  }
  ```

## Configuration Example

**AgentConfig**
```yaml
apiVersion: agent.tekton.dev/v1alpha1
kind: AgentConfig
metadata:
  name: cluster-diagnostics
spec:
  serviceAccount: agent-sa
  configPVC: agent-config-pvc
  maxIterations: 3
  timeout: 8m
  preHooks:
    - llm-security-check
  postHooks:
    - audit-bundle
  policy:
    opa: strict
  networkPolicy: strict
  provider: claude
```

**AgentRun**
```yaml
apiVersion: agent.tekton.dev/v1alpha1
kind: AgentRun
metadata:
  name: debug-deployment-001
spec:
  configRef:
    name: cluster-diagnostics
  goal: "Diagnose why deployment 'api-server' is failing in namespace 'production'"
  context:
    hints:
      - "Check recent events"
      - "Review pod logs"
```

**Config PVC Content**
```
/workspace/config/
├── prompts/
│   ├── system.txt              # System prompt
│   ├── planner.txt             # Planning prompt template
│   └── reflector.txt           # Reflection prompt template
├── guardrails/
│   ├── tool-policy.rego        # OPA policies
│   └── blocklists.json
├── tools/
│   └── tools.json              # Tool allowlist + schemas
└── providers/
    └── claude.json             # Provider config
```

## Non-Goals (MVP)

- Write actions (create, patch, delete resources)
- Dynamic tool installation
- External web search
- User-provided code execution
- Vector database integration
- Automatic AgentRun creation on events
- Webhooks (async validation instead)
- Multi-container pods (single container)
- OPA sidecar (embedded instead)
- Tekton CustomRun integration (standalone first)

## Dependencies

- Kubernetes 1.27+
- Go 1.21+
- k8s.io/client-go v0.28+
- k8s.io/apimachinery v0.28+
- knative.dev/pkg v0.0.0-latest
- github.com/open-policy-agent/opa v0.55+
- go.opentelemetry.io/otel v1.16+

## Installation

### Minikube
```bash
# Start cluster
minikube start

# Install controller
make install    # Install CRDs
make deploy     # Deploy controller

# Create config PVC
kubectl apply -f examples/config-pvc.yaml

# Create AgentConfig
kubectl apply -f examples/agentconfig-sample.yaml

# Create Secret with API key
kubectl create secret generic claude-api-key \
  --from-literal=CLAUDE_API_KEY=sk-...

# Run AgentRun
kubectl apply -f examples/agentrun-sample.yaml

# Check status
kubectl get agentrun -w
```

### OpenShift
```bash
# Install via OLM
oc apply -f bundle/manifests/

# Or via Helm
helm install agentrun-controller charts/agentrun-controller
```

## Future Enhancements

**Tekton Integration**
- Implement CustomRun controller wrapping AgentRun
- Users can reference AgentRun from Pipeline
- Reuse Tekton hooks (Tasks instead of Pods)

**Vector Database**
- RAG over Tekton Results DB
- Historical failure pattern matching
- Read-only index in MVP, write-back later

**Multi-Tenant Isolation**
- Per-namespace AgentConfigs
- Network and secret isolation
- Quotas and limits

**Scoped Write Actions**
- Add single write tool (e.g., rollout_restart)
- Human approval required
- Namespace allowlist

## License

Apache 2.0

## Repository

https://github.com/waveywaves/agentrun-controller
