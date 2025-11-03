# Tekton Pipeline Repository Structure and Conventions Research

**Research Date**: 2025-10-01
**Target Repository**: https://github.com/tektoncd/pipeline
**Purpose**: Inform agentrun-controller implementation with Tekton best practices

## Executive Summary

This research examines the tektoncd/pipeline repository to identify patterns, conventions, and best practices for implementing a CustomRun controller. The findings are organized by implementation area with specific recommendations for the agentrun-controller project.

---

## 1. Directory Structure

### Tekton Pipeline Standard Layout

```
tektoncd/pipeline/
├── cmd/                          # Binary entry points
│   ├── controller/               # Main controller binary
│   ├── webhook/                  # Webhook server binary
│   ├── entrypoint/               # Task entrypoint binary
│   ├── resolvers/                # Resolution service
│   └── events/                   # Event processing
├── pkg/
│   ├── apis/                     # API definitions
│   │   ├── config/               # Configuration APIs
│   │   ├── pipeline/             # Main pipeline APIs
│   │   │   ├── v1/               # Stable v1 API
│   │   │   ├── v1beta1/          # Beta API
│   │   │   └── v1alpha1/         # Alpha API (deprecated)
│   │   ├── resolution/           # Resolution APIs
│   │   └── run/                  # Run-related shared APIs
│   │       ├── v1alpha1/
│   │       └── v1beta1/
│   ├── client/                   # Generated clients
│   │   ├── clientset/            # Typed clientsets
│   │   ├── informers/            # Shared informers
│   │   ├── listers/              # Resource listers
│   │   ├── injection/            # Knative injection
│   │   ├── resolution/           # Resolution client (hand-written)
│   │   └── resource/             # Resource client (hand-written)
│   ├── reconciler/               # Controllers
│   │   ├── pipelinerun/
│   │   ├── taskrun/
│   │   ├── resolutionrequest/
│   │   ├── notifications/        # Event controllers
│   │   │   └── customrun/        # CustomRun notification controller
│   │   ├── events/
│   │   └── volumeclaim/          # PVC lifecycle
│   ├── pod/                      # Pod creation utilities
│   └── workspace/                # Workspace handling
├── config/                       # Kubernetes manifests
│   ├── 100-namespace/
│   ├── 200-clusterrole.yaml      # RBAC definitions
│   ├── 200-role.yaml
│   ├── 200-serviceaccount.yaml
│   ├── 201-clusterrolebinding.yaml
│   ├── 201-rolebinding.yaml
│   ├── 300-crds/                 # CRD manifests
│   ├── 500-webhooks.yaml         # Webhook configurations
│   ├── resolvers/                # Resolver configs
│   └── webhook*.yaml             # Webhook-specific configs
├── test/                         # Test resources
│   ├── conformance/              # API conformance tests
│   ├── custom-task-ctrls/        # Custom task test controllers
│   ├── testdata/                 # Test fixtures
│   └── upgrade/                  # Upgrade tests
├── hack/                         # Development scripts
│   ├── update-codegen.sh         # Code generation
│   ├── update-deps.sh            # Dependency updates
│   └── ...
├── docs/                         # Documentation
│   ├── tasks.md
│   ├── pipelines.md
│   ├── customruns.md             # CustomRun documentation
│   └── ...
├── examples/                     # Example manifests
├── Makefile                      # Build targets
└── PROJECT                       # Project metadata
```

### Recommendation for agentrun-controller

```
agentrun-controller/
├── cmd/
│   ├── controller/               # Main CustomRun controller
│   └── webhook/                  # Webhook server (if needed)
├── pkg/
│   ├── apis/
│   │   └── agent/
│   │       └── v1alpha1/         # Start with v1alpha1
│   │           ├── agentconfig_types.go
│   │           ├── agentconfig_defaults.go
│   │           ├── agentconfig_validation.go
│   │           ├── agentrun_types.go
│   │           ├── agentrun_defaults.go
│   │           ├── agentrun_validation.go
│   │           ├── register.go
│   │           ├── doc.go
│   │           └── zz_generated.deepcopy.go
│   ├── client/                   # Generated (DO NOT EDIT manually)
│   │   ├── clientset/
│   │   ├── informers/
│   │   └── listers/
│   ├── reconciler/
│   │   └── agentrun/             # Main reconciler
│   │       ├── controller.go     # Controller setup
│   │       ├── agentrun.go       # Reconciliation logic
│   │       └── agentrun_test.go
│   ├── agent/                    # Agent runtime logic
│   │   ├── loop.go               # Plan-act-reflect loop
│   │   ├── provider.go           # LLM provider interface
│   │   └── tools/                # Tool implementations
│   ├── security/                 # Security components
│   │   ├── opa.go                # OPA integration
│   │   └── rbac.go               # RBAC generation
│   └── pod/                      # Pod creation utilities
├── config/
│   ├── 100-namespace.yaml
│   ├── 200-clusterrole.yaml
│   ├── 200-serviceaccount.yaml
│   ├── 201-clusterrolebinding.yaml
│   ├── 300-agentconfig.yaml      # CRD
│   ├── 300-agentrun.yaml         # CRD
│   ├── 400-controller.yaml       # Deployment
│   └── 500-webhook.yaml          # If using webhooks
├── test/
│   ├── testdata/
│   └── e2e/
├── hack/
│   ├── update-codegen.sh
│   └── update-deps.sh
├── docs/
├── examples/
└── Makefile
```

---

## 2. CRD Definitions and API Patterns

### Tekton Patterns Observed

#### File Naming Convention
- `{resource}_types.go` - Type definitions
- `{resource}_defaults.go` - Default values
- `{resource}_validation.go` - Validation logic
- `{resource}_conversion.go` - Version conversions
- `{resource}_types_test.go` - Type tests
- `zz_generated.deepcopy.go` - Generated deep copy (DO NOT EDIT)
- `register.go` - API group registration
- `doc.go` - Package documentation

#### CustomRun Type Structure

From `pkg/apis/pipeline/v1beta1/customrun_types.go`:

```go
// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// CustomRun represents a single execution of a Custom Task.
type CustomRun struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // +optional
    Spec CustomRunSpec `json:"spec,omitempty"`

    // +optional
    Status CustomRunStatus `json:"status,omitempty"`
}

type CustomRunSpec struct {
    // CustomRef references a custom task type
    // +optional
    CustomRef *TaskRef `json:"customRef,omitempty"`

    // CustomSpec embeds the custom task spec
    // +optional
    CustomSpec *EmbeddedCustomRunSpec `json:"customSpec,omitempty"`

    // Params
    // +optional
    // +listType=atomic
    Params Params `json:"params,omitempty"`

    // ServiceAccountName
    // +optional
    ServiceAccountName string `json:"serviceAccountName,omitempty"`

    // Timeout
    // +optional
    Timeout *metav1.Duration `json:"timeout,omitempty"`

    // Workspaces
    // +optional
    // +listType=atomic
    Workspaces []WorkspaceBinding `json:"workspaces,omitempty"`

    // Retries
    // +optional
    Retries int `json:"retries,omitempty"`

    // Status (allows users to cancel)
    // +optional
    Status CustomRunSpecStatus `json:"status,omitempty"`
}
```

#### Important Kubebuilder Markers

```go
// At package level (doc.go):
// +k8s:deepcopy-gen=package
// +groupName=tekton.dev

// On types:
// +genclient                           // Generate typed client
// +genreconciler                       // Generate reconciler skeleton (Knative)
// +k8s:deepcopy-gen:interfaces=...     // Generate DeepCopy methods
// +k8s:openapi-gen=true                // Include in OpenAPI schema

// On fields:
// +optional                            // Not required
// +listType=atomic                     // List is treated as single unit
// +listType=map                        // List is map with keys
// +listMapKey=name                     // Key field for map lists
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=64
```

### Validation Patterns

From `pkg/apis/pipeline/v1beta1/customrun_validation.go`:

```go
func (r *CustomRun) Validate(ctx context.Context) *apis.FieldError {
    if err := validate.ObjectMetadata(r.GetObjectMeta()); err != nil {
        return err.ViaField("metadata")
    }
    return r.Spec.Validate(ctx)
}

func (rs *CustomRunSpec) Validate(ctx context.Context) *apis.FieldError {
    // Ensure exactly one of CustomRef or CustomSpec
    if rs.CustomRef == nil && rs.CustomSpec == nil {
        return apis.ErrMissingOneOf("customRef", "customSpec")
    }
    if rs.CustomRef != nil && rs.CustomSpec != nil {
        return apis.ErrMultipleOneOf("customRef", "customSpec")
    }

    // Validate CustomRef
    if rs.CustomRef != nil {
        if rs.CustomRef.APIVersion == "" {
            return apis.ErrMissingField("customRef.apiVersion")
        }
        if rs.CustomRef.Kind == "" {
            return apis.ErrMissingField("customRef.kind")
        }
    }

    // Validate workspaces
    if err := validateWorkspaceBindings(rs.Workspaces); err != nil {
        return err
    }

    return nil
}
```

#### Validation Testing Pattern

From `pkg/apis/pipeline/v1beta1/customrun_validation_test.go`:

```go
func TestCustomRun_Invalid(t *testing.T) {
    tests := []struct {
        name      string
        customRun *v1beta1.CustomRun
        want      *apis.FieldError
    }{
        {
            name: "missing customRef and customSpec",
            customRun: &v1beta1.CustomRun{
                ObjectMeta: metav1.ObjectMeta{Name: "test"},
                Spec: v1beta1.CustomRunSpec{},
            },
            want: apis.ErrMissingOneOf("spec.customRef", "spec.customSpec"),
        },
        {
            name: "both customRef and customSpec",
            customRun: &v1beta1.CustomRun{
                ObjectMeta: metav1.ObjectMeta{Name: "test"},
                Spec: v1beta1.CustomRunSpec{
                    CustomRef: &v1beta1.TaskRef{
                        APIVersion: "example.dev/v1",
                        Kind:       "Example",
                    },
                    CustomSpec: &v1beta1.EmbeddedCustomRunSpec{
                        APIVersion: "example.dev/v1",
                        Kind:       "Example",
                    },
                },
            },
            want: apis.ErrMultipleOneOf("spec.customRef", "spec.customSpec"),
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            err := tc.customRun.Validate(context.Background())
            if d := cmp.Diff(tc.want.Error(), err.Error()); d != "" {
                t.Error(d)
            }
        })
    }
}
```

### Defaulting Patterns

From `pkg/apis/pipeline/v1beta1/customrun_defaults.go`:

```go
func (r *CustomRun) SetDefaults(ctx context.Context) {
    r.Spec.SetDefaults(ctx)
}

func (rs *CustomRunSpec) SetDefaults(ctx context.Context) {
    cfg := config.FromContextOrDefaults(ctx)
    defaultSA := cfg.Defaults.DefaultServiceAccount

    if rs.ServiceAccountName == "" && defaultSA != "" {
        rs.ServiceAccountName = defaultSA
    }
}
```

Key pattern: Load configuration from context (injected by webhooks/controllers).

### Recommendations for AgentRun

1. **Follow Tekton's file naming convention exactly**
   - `agentconfig_types.go`, `agentconfig_defaults.go`, `agentconfig_validation.go`
   - `agentrun_types.go`, `agentrun_defaults.go`, `agentrun_validation.go`

2. **Use consistent kubebuilder markers**
   ```go
   // +genclient
   // +genreconciler
   // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
   ```

3. **Implement validation asynchronously**
   - Validation in webhooks should be lightweight
   - Deeper validation happens in reconciler
   - Use `apis.FieldError` for consistent error reporting

4. **Use table-driven tests**
   - Clear test case naming
   - Comprehensive coverage of edge cases
   - Use `cmp.Diff()` for error comparison

---

## 3. Controller Implementation Patterns

### Controller Setup

From `pkg/reconciler/pipelinerun/controller.go`:

```go
func NewController(opts *pipeline.Options, clock clock.PassiveClock) func(context.Context, configmap.Watcher) *controller.Impl {
    return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
        // Get clients
        kubeclient := kubeclient.Get(ctx)
        pipelineclient := pipelineclient.Get(ctx)

        // Get informers
        pipelineRunInformer := pipelineruninformer.Get(ctx)
        taskRunInformer := taskruninformer.Get(ctx)
        podInformer := podinformer.Get(ctx)

        // Create config store
        configStore := config.NewStore(logger.Named("config-store"), configmap.TypeFilter(&Metrics{}, &FeatureFlags{}))
        configStore.WatchConfigs(cmw)

        // Create reconciler
        c := &Reconciler{
            KubeClientSet:     kubeclient,
            PipelineClientSet: pipelineclient,
            Images:            opts.Images,
            Clock:             clock,
            // ... other fields
        }

        // Create controller implementation
        impl := pipelinerunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
            return controller.Options{
                AgentName:         pipeline.PipelineRunControllerName,
                ConfigStore:       configStore,
                PromoteFilterFunc: filterManagedByTekton,
            }
        })

        // Add event handlers
        pipelineRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
            FilterFunc: filterManagedByTekton,
            Handler:    controller.HandleAll(impl.Enqueue),
        })

        // Watch TaskRuns owned by PipelineRuns
        taskRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
            FilterFunc: controller.FilterController(&v1beta1.PipelineRun{}),
            Handler:    controller.HandleAll(impl.EnqueueControllerOf),
        })

        // Watch Pods owned by TaskRuns (indirect ownership)
        podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
            FilterFunc: controller.FilterController(&v1beta1.TaskRun{}),
            Handler:    controller.HandleAll(impl.EnqueueLabelOfClusterScopedResource("", pipeline.PipelineRunLabelKey)),
        })

        return impl
    }
}
```

#### Key Patterns:
1. **Closure-based controller factory** - Returns function that creates controller
2. **Knative injection** - Gets clients/informers from context
3. **ConfigStore** - Watches ConfigMaps for feature flags, metrics config
4. **Multiple informers** - Watch related resources (owner references, labels)
5. **Filter functions** - Limit events to relevant resources

### Reconciliation Logic

From `pkg/reconciler/pipelinerun/pipelinerun.go`:

```go
func (c *Reconciler) ReconcileKind(ctx context.Context, pr *v1.PipelineRun) pkgreconciler.Event {
    logger := logging.FromContext(ctx)
    ctx = cloudevent.ToContext(ctx, c.CloudEventClient)
    ctx = metrics.ToContext(ctx, c.Metrics)

    // Before state (for comparison)
    before := pr.DeepCopy()

    // Check if done
    if pr.IsDone() {
        logger.Info("PipelineRun is done")
        return c.finishReconcile(ctx, pr, before)
    }

    // Check if cancelled
    if pr.IsCancelled() {
        return c.handleCancellation(ctx, pr)
    }

    // Check timeout
    if pr.HasTimedOut(ctx, c.Clock) {
        return c.handleTimeout(ctx, pr)
    }

    // Start if not started
    if !pr.HasStarted() {
        pr.Status.InitializeConditions(ctx)
        pr.Status.StartTime = &metav1.Time{Time: c.Clock.Now()}
    }

    // Reconcile the pipeline run
    if err := c.reconcile(ctx, pr); err != nil {
        logger.Errorf("Reconcile error: %v", err)
        return err
    }

    return c.finishReconcile(ctx, pr, before)
}

func (c *Reconciler) finishReconcile(ctx context.Context, pr *v1.PipelineRun, before *v1.PipelineRun) error {
    // Update status
    if _, err := c.updateStatus(ctx, pr); err != nil {
        return err
    }

    // Emit events if status changed
    if !equality.Semantic.DeepEqual(before.Status, pr.Status) {
        c.Recorder.Event(pr, corev1.EventTypeNormal, "Succeeded", "PipelineRun completed")
        cloudevent.EmitCloudEvents(ctx, pr)
    }

    return nil
}
```

#### Key Patterns:
1. **DeepCopy before state** - Compare before/after for events
2. **State checks first** - Done? Cancelled? Timed out?
3. **Initialize conditions** - On first reconciliation
4. **Separate update method** - Status updates isolated
5. **Event emission** - Only when status changes
6. **Structured logging** - Context-aware logger

### CustomRun Notification Controller

From `pkg/reconciler/notifications/customrun/controller.go`:

```go
func NewController(ctx context.Context) *controller.Impl {
    logger := logging.FromContext(ctx)
    customRunInformer := customruninformer.Get(ctx)

    r := &Reconciler{
        cloudEventClient: cloudeventclient.Get(ctx),
        cacheClient:      cache.Get(ctx),
    }

    impl := customrunreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
        return controller.Options{
            AgentName:         pipeline.CustomRunControllerName,
            SkipStatusUpdates: true,  // Read-only controller
        }
    })

    customRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

    return impl
}
```

From `pkg/reconciler/notifications/customrun/customrun.go`:

```go
func (r *Reconciler) ReconcileKind(ctx context.Context, customRun *v1beta1.CustomRun) pkgreconciler.Event {
    logger := logging.FromContext(ctx)

    // Make a copy to avoid mutating cache
    customRunEvents := customRun.DeepCopy()

    // Potentially emit cloud events
    if configs.FeatureFlags.SendCloudEventsForRuns {
        condition := customRunEvents.Status.GetCondition(apis.ConditionSucceeded)
        logger.Debugf("Emitting cloudevent for %s, condition: %s", customRunEvents.Name, condition)
        events.EmitCloudEvents(ctx, customRunEvents)
    }

    return nil
}
```

**Note**: This is a **read-only controller** that only observes and emits events. It does NOT update CustomRun status - that's the responsibility of the custom task controller.

### Recommendations for AgentRun Controller

1. **Use Knative injection pattern**
   - Get clients/informers from context
   - Use `genreconciler` to generate boilerplate

2. **Implement ReconcileKind**
   ```go
   func (r *Reconciler) ReconcileKind(ctx context.Context, ar *v1alpha1.AgentRun) pkgreconciler.Event
   ```

3. **State machine approach**
   - Check Done/Cancelled/TimedOut first
   - Initialize conditions on first reconcile
   - Track phases: PreHooks → Planning → Acting → Reflecting → PostHooks

4. **Status updates**
   - Separate method for status updates
   - Use DeepCopy to compare before/after
   - Emit events only on changes

5. **Watch related resources**
   - Watch CustomRuns owned by AgentRun
   - Watch Pods created by agent
   - Use owner references for garbage collection

---

## 4. Webhook Setup

### Webhook Server Structure

From `cmd/webhook/main.go`:

```go
func main() {
    // Parse flags
    flag.Parse()

    ctx := signals.NewContext()

    // Get webhook name from env
    serviceName := os.Getenv("WEBHOOK_SERVICE_NAME")
    if serviceName == "" {
        serviceName = "webhook.pipeline.tekton.dev"
    }

    // Start webhook server
    sharedmain.WebhookMainWithConfig(ctx, "webhook",
        sharedmain.ParseAndGetConfigOrDie(),
        certificates.NewController,
        webhook.NewDefaultingAdmissionController,
        webhook.NewValidationAdmissionController,
        webhook.NewConfigValidationController,
        webhook.NewConversionController,
    )
}
```

### Webhook Types

From `config/500-webhooks.yaml`:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: validation.webhook.pipeline.tekton.dev
webhooks:
  - admissionReviewVersions: ["v1"]
    clientConfig:
      service:
        name: tekton-pipelines-webhook
        namespace: tekton-pipelines
    failurePolicy: Fail
    name: validation.webhook.pipeline.tekton.dev
    sideEffects: None

---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: webhook.pipeline.tekton.dev
webhooks:
  - admissionReviewVersions: ["v1"]
    clientConfig:
      service:
        name: tekton-pipelines-webhook
        namespace: tekton-pipelines
    failurePolicy: Fail
    name: webhook.pipeline.tekton.dev
    sideEffects: None

---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: config.webhook.pipeline.tekton.dev
webhooks:
  - admissionReviewVersions: ["v1"]
    clientConfig:
      service:
        name: tekton-pipelines-webhook
        namespace: tekton-pipelines
    failurePolicy: Fail
    objectSelector:
      matchLabels:
        app.kubernetes.io/part-of: tekton-pipelines
    name: config.webhook.pipeline.tekton.dev
    sideEffects: None
```

### Recommendation for AgentRun

**Option 1: No Webhooks (Recommended for MVP)**
- Do validation asynchronously in reconciler
- Set status.conditions with validation errors
- Simpler deployment, fewer moving parts

**Option 2: Add Webhooks Later**
- If synchronous validation becomes important
- Follow Tekton's webhook patterns
- Use Knative's webhook framework
- Implement defaulting and validation separately

---

## 5. RBAC Patterns

From `config/200-clusterrole.yaml`:

```yaml
# Controller needs broad read access and limited write
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tekton-pipelines-controller-cluster-access
  labels:
    app.kubernetes.io/component: controller
    app.kubernetes.io/instance: default
    app.kubernetes.io/part-of: tekton-pipelines
rules:
  # Tekton resources - full CRUD
  - apiGroups: ["tekton.dev"]
    resources: ["tasks", "taskruns", "pipelines", "pipelineruns", "customruns"]
    verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
  # TaskRuns/PipelineRuns status subresource
  - apiGroups: ["tekton.dev"]
    resources: ["taskruns/status", "pipelineruns/status", "customruns/status"]
    verbs: ["get", "update", "patch"]
  # Resolution APIs
  - apiGroups: ["resolution.tekton.dev"]
    resources: ["resolutionrequests"]
    verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
  # Pods - for TaskRun execution
  - apiGroups: [""]
    resources: ["pods", "pods/log"]
    verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
  # ConfigMaps - for configuration
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch"]
  # PVCs - for workspaces
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
  # Events - for Kubernetes events
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "update", "patch"]
  # Namespaces - for namespace validation
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list"]

---
# Controller tenant access (namespace-scoped)
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tekton-pipelines-controller-tenant-access
  labels:
    app.kubernetes.io/component: controller
    app.kubernetes.io/instance: default
    app.kubernetes.io/part-of: tekton-pipelines
rules:
  # Pods
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
  # ConfigMaps (read-only)
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch"]
  # Secrets (read-only) - minimal access
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  # ServiceAccounts (read-only)
  - apiGroups: [""]
    resources: ["serviceaccounts"]
    verbs: ["get", "list"]
  # Events
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "update", "patch"]
  # PVCs
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
  # LimitRanges (read-only)
  - apiGroups: [""]
    resources: ["limitranges"]
    verbs: ["get", "list"]

---
# Webhook cluster access
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tekton-pipelines-webhook-cluster-access
  labels:
    app.kubernetes.io/component: webhook
    app.kubernetes.io/instance: default
    app.kubernetes.io/part-of: tekton-pipelines
rules:
  # CRD management
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["get", "list", "update", "patch", "watch"]
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions/status"]
    verbs: ["get", "update", "patch"]
  # Webhook configuration
  - apiGroups: ["admissionregistration.k8s.io"]
    resources: ["mutatingwebhookconfigurations", "validatingwebhookconfigurations"]
    verbs: ["list", "watch"]
  - apiGroups: ["admissionregistration.k8s.io"]
    resources: ["mutatingwebhookconfigurations"]
    resourceNames: ["webhook.pipeline.tekton.dev"]
    verbs: ["get", "update", "delete"]
  - apiGroups: ["admissionregistration.k8s.io"]
    resources: ["validatingwebhookconfigurations"]
    resourceNames: ["validation.webhook.pipeline.tekton.dev", "config.webhook.pipeline.tekton.dev"]
    verbs: ["get", "update", "delete"]
```

### Key RBAC Principles:

1. **Separation of concerns**
   - Controller cluster access (broad read, scoped write)
   - Controller tenant access (namespace operations)
   - Webhook cluster access (CRD and webhook management)

2. **Principle of least privilege**
   - Read-only where possible (configmaps, secrets, serviceaccounts)
   - Write only for owned resources
   - Status subresource permissions separate

3. **Explicit resource names**
   - Webhook configs reference specific resource names
   - Prevents overly broad permissions

### Recommendations for AgentRun

```yaml
# Controller needs to manage AgentRuns and create CustomRuns
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: agentrun-controller-cluster-access
rules:
  # AgentRun CRD
  - apiGroups: ["agent.tekton.dev"]
    resources: ["agentconfigs", "agentruns"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["agent.tekton.dev"]
    resources: ["agentruns/status"]
    verbs: ["get", "update", "patch"]

  # Create CustomRuns for agent execution
  - apiGroups: ["tekton.dev"]
    resources: ["customruns"]
    verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
  - apiGroups: ["tekton.dev"]
    resources: ["customruns/status"]
    verbs: ["get", "update", "patch"]

  # Create Pods for agent execution
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "create", "update", "delete", "watch"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get", "list"]

  # Read PVCs for config/data volumes
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list"]

  # Read ServiceAccounts for agent execution
  - apiGroups: [""]
    resources: ["serviceaccounts"]
    verbs: ["get", "list"]

  # Read Secrets for LLM API keys
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]

  # Create RBAC for agent pods
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["roles", "rolebindings"]
    verbs: ["get", "list", "create", "update", "delete"]

  # Create NetworkPolicies
  - apiGroups: ["networking.k8s.io"]
    resources: ["networkpolicies"]
    verbs: ["get", "list", "create", "update", "delete"]

  # Events
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "update", "patch"]

  # ConfigMaps for feature flags
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch"]
```

---

## 6. Testing Patterns

### Test Structure

```
test/
├── conformance/           # API spec conformance tests
├── custom-task-ctrls/    # Test custom task controllers
├── testdata/             # Test fixtures (YAML manifests)
└── upgrade/              # Upgrade scenario tests

pkg/reconciler/{resource}/
├── controller.go         # Controller setup
├── {resource}.go         # Reconciliation logic
└── {resource}_test.go    # Unit tests
```

### Unit Test Pattern

From `test/controller.go`:

```go
// Data represents the desired state of the system for testing
type Data struct {
    PipelineRuns []*v1beta1.PipelineRun
    Pipelines    []*v1beta1.Pipeline
    TaskRuns     []*v1beta1.TaskRun
    Tasks        []*v1beta1.Task
    Pods         []*corev1.Pod
    Namespaces   []*corev1.Namespace
    ConfigMaps   []*corev1.ConfigMap
}

// Clients holds fake clients
type Clients struct {
    Pipeline   *fakepipelineclient.Clientset
    Kube       *fakekubeclient.Clientset
    CloudEvent *cloudevent.FakeClient
}

// Assets holds controller test assets
type Assets struct {
    Logger     *zap.SugaredLogger
    Controller *controller.Impl
    Clients    Clients
    Informers  Informers
    Ctx        context.Context
}

// SeedTestData populates fake clients with test data
func SeedTestData(t *testing.T, ctx context.Context, d Data) (Clients, Informers) {
    c := Clients{
        Pipeline: fakepipelineclient.NewSimpleClientset(),
        Kube:     fakekubeclient.NewSimpleClientset(),
    }

    // Add pipeline resources
    for _, pr := range d.PipelineRuns {
        if _, err := c.Pipeline.TektonV1beta1().PipelineRuns(pr.Namespace).Create(ctx, pr, metav1.CreateOptions{}); err != nil {
            t.Fatal(err)
        }
    }

    // Add kube resources
    for _, pod := range d.Pods {
        if _, err := c.Kube.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
            t.Fatal(err)
        }
    }

    // ... (similar for other resources)

    return c, informers
}
```

### Table-Driven Test Example

```go
func TestReconcile(t *testing.T) {
    testCases := []struct {
        name            string
        pipelineRun     *v1beta1.PipelineRun
        pipeline        *v1beta1.Pipeline
        tasks           []*v1beta1.Task
        expectedStatus  corev1.ConditionStatus
        expectedReason  string
    }{
        {
            name: "successful pipeline execution",
            pipelineRun: &v1beta1.PipelineRun{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-pipeline-run",
                    Namespace: "default",
                },
                Spec: v1beta1.PipelineRunSpec{
                    PipelineRef: &v1beta1.PipelineRef{Name: "test-pipeline"},
                },
            },
            pipeline: &v1beta1.Pipeline{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-pipeline",
                    Namespace: "default",
                },
                Spec: v1beta1.PipelineSpec{
                    Tasks: []v1beta1.PipelineTask{
                        {Name: "task1", TaskRef: &v1beta1.TaskRef{Name: "test-task"}},
                    },
                },
            },
            tasks: []*v1beta1.Task{
                {
                    ObjectMeta: metav1.ObjectMeta{
                        Name:      "test-task",
                        Namespace: "default",
                    },
                },
            },
            expectedStatus: corev1.ConditionTrue,
            expectedReason: "Succeeded",
        },
        // ... more test cases
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup
            d := Data{
                PipelineRuns: []*v1beta1.PipelineRun{tc.pipelineRun},
                Pipelines:    []*v1beta1.Pipeline{tc.pipeline},
                Tasks:        tc.tasks,
            }
            clients, informers := SeedTestData(t, ctx, d)

            // Reconcile
            reconciler := &Reconciler{
                PipelineClientSet: clients.Pipeline,
                KubeClientSet:     clients.Kube,
            }
            err := reconciler.ReconcileKind(ctx, tc.pipelineRun)

            // Assert
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            // Check status
            reconciledPR, _ := clients.Pipeline.TektonV1beta1().PipelineRuns("default").Get(ctx, "test-pipeline-run", metav1.GetOptions{})
            condition := reconciledPR.Status.GetCondition(apis.ConditionSucceeded)

            if condition.Status != tc.expectedStatus {
                t.Errorf("expected status %v, got %v", tc.expectedStatus, condition.Status)
            }
            if condition.Reason != tc.expectedReason {
                t.Errorf("expected reason %v, got %v", tc.expectedReason, condition.Reason)
            }
        })
    }
}
```

### E2E Test Pattern

```go
// +build e2e

func TestPipelineRun(t *testing.T) {
    // Create random namespace
    namespace := names.SimpleNameGenerator.GenerateName("test-")

    // Cleanup
    defer func() {
        if err := c.KubeClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
            t.Errorf("failed to delete namespace: %v", err)
        }
    }()

    // Create PipelineRun
    pr := &v1beta1.PipelineRun{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-pipeline-run",
            Namespace: namespace,
        },
        Spec: v1beta1.PipelineRunSpec{
            PipelineRef: &v1beta1.PipelineRef{Name: "test-pipeline"},
        },
    }

    if _, err := c.PipelineClient.TektonV1beta1().PipelineRuns(namespace).Create(ctx, pr, metav1.CreateOptions{}); err != nil {
        t.Fatalf("failed to create PipelineRun: %v", err)
    }

    // Wait for completion
    if err := WaitForPipelineRunState(ctx, c, "test-pipeline-run", namespace, 5*time.Minute, func(pr *v1beta1.PipelineRun) (bool, error) {
        return pr.IsDone(), nil
    }); err != nil {
        t.Fatalf("PipelineRun did not complete: %v", err)
    }

    // Assert success
    pr, _ = c.PipelineClient.TektonV1beta1().PipelineRuns(namespace).Get(ctx, "test-pipeline-run", metav1.GetOptions{})
    if !pr.IsSuccessful() {
        t.Errorf("PipelineRun failed: %v", pr.Status.Conditions)
    }
}
```

### Recommendations for AgentRun

1. **Follow table-driven test pattern**
   - Clear test case structure
   - Comprehensive coverage
   - Easy to add new cases

2. **Use fake clients for unit tests**
   - Fast execution
   - No cluster required
   - Predictable behavior

3. **Implement E2E tests separately**
   - Use build tags: `// +build e2e`
   - Test against real cluster
   - Cover full workflows

4. **Test utilities to create**
   - `SeedTestData()` - populate fake clients
   - `WaitForAgentRunState()` - polling helper
   - Test fixtures in `test/testdata/`

---

## 7. Code Generation

### Code Generation Script

From `hack/update-codegen.sh`:

```bash
#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# Generate deepcopy, client, informer, and lister
echo "Generating client code..."
bash vendor/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/tektoncd/pipeline/pkg/client github.com/tektoncd/pipeline/pkg/apis \
  "pipeline:v1alpha1,v1beta1,v1 resolution:v1alpha1,v1beta1" \
  --go-header-file hack/boilerplate.go.txt

# Generate Knative injection
echo "Generating injection code..."
bash vendor/knative.dev/pkg/hack/generate-knative.sh "injection" \
  github.com/tektoncd/pipeline/pkg/client github.com/tektoncd/pipeline/pkg/apis \
  "pipeline:v1alpha1,v1beta1,v1 resolution:v1alpha1,v1beta1" \
  --go-header-file hack/boilerplate.go.txt

# Generate OpenAPI spec
echo "Generating OpenAPI spec..."
go run k8s.io/kube-openapi/cmd/openapi-gen \
  --input-dirs github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1 \
  --output-package github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1 \
  --output-file-base zz_generated.openapi \
  --go-header-file hack/boilerplate.go.txt

# Update CRD with OpenAPI schema
echo "Updating CRD schemas..."
controller-gen crd:crdVersions=v1 paths=./pkg/apis/pipeline/v1beta1/... output:crd:dir=config/300-crds
```

### What Gets Generated

```
pkg/client/
├── clientset/
│   └── versioned/
│       ├── typed/
│       │   └── pipeline/
│       │       └── v1alpha1/
│       │           ├── pipeline_client.go
│       │           ├── pipeline.go
│       │           ├── pipelinerun.go
│       │           └── ...
│       └── clientset.go
├── informers/
│   └── externalversions/
│       ├── pipeline/
│       │   └── v1alpha1/
│       │       ├── interface.go
│       │       ├── pipeline.go
│       │       ├── pipelinerun.go
│       │       └── ...
│       └── factory.go
├── listers/
│   └── pipeline/
│       └── v1alpha1/
│           ├── pipeline.go
│           └── pipelinerun.go
└── injection/
    ├── client/
    ├── informers/
    └── reconciler/
```

### Kubebuilder Markers for Code Generation

```go
// Package-level markers (doc.go)
// +k8s:deepcopy-gen=package
// +groupName=tekton.dev

// Type-level markers
// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Field-level markers
// +optional
// +listType=atomic
// +listType=map
// +listMapKey=name
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=64
// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
```

### Recommendations for AgentRun

1. **Create `hack/update-codegen.sh`**
   ```bash
   #!/usr/bin/env bash

   set -o errexit
   set -o nounset
   set -o pipefail

   REPO_ROOT=$(git rev-parse --show-toplevel)
   cd "${REPO_ROOT}"

   # Generate deepcopy, client, informer, lister
   bash vendor/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
     github.com/waveywaves/agentrun-controller/pkg/client \
     github.com/waveywaves/agentrun-controller/pkg/apis \
     "agent:v1alpha1" \
     --go-header-file hack/boilerplate.go.txt

   # Generate Knative injection
   bash vendor/knative.dev/pkg/hack/generate-knative.sh "injection" \
     github.com/waveywaves/agentrun-controller/pkg/client \
     github.com/waveywaves/agentrun-controller/pkg/apis \
     "agent:v1alpha1" \
     --go-header-file hack/boilerplate.go.txt
   ```

2. **Add to Makefile**
   ```makefile
   .PHONY: generated
   generated:
       ./hack/update-codegen.sh
   ```

3. **Run after API changes**
   ```bash
   make generated
   ```

4. **Add generated files to `.gitignore`? NO!**
   - Tekton commits generated code
   - Makes reviews easier
   - Avoids generation step in CI

---

## 8. Configuration Management

### Feature Flags Pattern

From `pkg/apis/config/feature_flags.go`:

```go
const (
    // FeatureFlagKey is the ConfigMap key for feature flags
    FeatureFlagKey = "feature-flags"

    // EnableAPIFields enables alpha/beta API fields
    enableAPIFieldsKey = "enable-api-fields"
    defaultEnableAPIFields = "stable"
)

type FeatureFlags struct {
    EnableAPIFields string
    EnableCELInWhenExpression bool
    // ... more flags
}

func NewFeatureFlagsFromMap(data map[string]string) (*FeatureFlags, error) {
    ff := &FeatureFlags{}

    if v, ok := data[enableAPIFieldsKey]; ok {
        ff.EnableAPIFields = v
    } else {
        ff.EnableAPIFields = defaultEnableAPIFields
    }

    // Parse bool flags
    if v, ok := data["enable-cel-in-when-expression"]; ok {
        boolValue, err := strconv.ParseBool(v)
        if err != nil {
            return nil, fmt.Errorf("failed to parse %s: %w", "enable-cel-in-when-expression", err)
        }
        ff.EnableCELInWhenExpression = boolValue
    }

    return ff, nil
}
```

### Config Store Pattern

From `pkg/apis/config/store.go`:

```go
type Store struct {
    *configmap.UntypedStore
}

func NewStore(logger *zap.SugaredLogger, onAfterStore ...func(name string, value interface{})) *Store {
    return &Store{
        UntypedStore: configmap.NewUntypedStore(
            "config",
            logger,
            configmap.Constructors{
                FeatureFlagKey: NewFeatureFlagsFromConfigMap,
                DefaultsConfigName: NewDefaultsFromConfigMap,
                MetricsConfigName: NewMetricsFromConfigMap,
            },
            onAfterStore...,
        ),
    }
}

func (s *Store) ToContext(ctx context.Context) context.Context {
    return s.UntypedStore.ToContext(ctx)
}

func FromContext(ctx context.Context) *Config {
    return &Config{
        FeatureFlags: featureFlagsFromContext(ctx),
        Defaults:     defaultsFromContext(ctx),
        Metrics:      metricsFromContext(ctx),
    }
}

func FromContextOrDefaults(ctx context.Context) *Config {
    cfg := FromContext(ctx)
    if cfg.FeatureFlags == nil {
        cfg.FeatureFlags = DefaultFeatureFlags()
    }
    // ... set other defaults
    return cfg
}
```

### Loading in Controller

```go
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
    // Create config store
    configStore := config.NewStore(logger.Named("config-store"))

    // Watch ConfigMaps
    configStore.WatchConfigs(cmw)

    // Pass to controller
    impl := reconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
        return controller.Options{
            ConfigStore: configStore,
        }
    })

    return impl
}
```

### Using in Reconciler

```go
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) error {
    cfg := config.FromContextOrDefaults(ctx)

    if cfg.FeatureFlags.EnableAPIFields == "alpha" {
        // Use alpha features
    }

    return nil
}
```

### Recommendations for AgentRun

1. **Create config package**
   ```go
   // pkg/apis/config/config.go
   type Config struct {
       FeatureFlags *FeatureFlags
       Defaults     *Defaults
   }

   type FeatureFlags struct {
       EnableWebhooks bool
       // ... other flags
   }

   type Defaults struct {
       DefaultServiceAccount string
       DefaultTimeout        time.Duration
   }
   ```

2. **Use ConfigMap for configuration**
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: agentrun-config
     namespace: agentrun-system
   data:
     feature-flags: |
       enable-webhooks: false
     defaults: |
       default-service-account: "default"
       default-timeout: "8m"
   ```

3. **Watch ConfigMap in controller**
   ```go
   configStore := config.NewStore(logger)
   configStore.WatchConfigs(cmw)
   ```

---

## 9. Observability

### Structured Logging

Tekton uses Knative's logging package (zap-based):

```go
import (
    "knative.dev/pkg/logging"
)

func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) error {
    logger := logging.FromContext(ctx)

    logger.Infof("Reconciling PipelineRun %s/%s", pr.Namespace, pr.Name)
    logger.Debugf("PipelineRun spec: %+v", pr.Spec)

    if err := r.reconcile(ctx, pr); err != nil {
        logger.Errorf("Reconcile failed: %v", err)
        return err
    }

    logger.Info("Reconciliation completed")
    return nil
}
```

### Metrics

Tekton exposes Prometheus metrics:

```go
import (
    "knative.dev/pkg/metrics"
)

var (
    pipelineRunCount = stats.Int64(
        "pipelinerun_count",
        "Number of pipelineruns",
        stats.UnitDimensionless,
    )

    pipelineRunDuration = stats.Float64(
        "pipelinerun_duration_seconds",
        "PipelineRun execution duration",
        stats.UnitSeconds,
    )
)

func recordMetrics(ctx context.Context, pr *v1beta1.PipelineRun) {
    stats.Record(ctx, pipelineRunCount.M(1))

    if pr.Status.CompletionTime != nil {
        duration := pr.Status.CompletionTime.Sub(pr.Status.StartTime.Time).Seconds()
        stats.Record(ctx, pipelineRunDuration.M(duration))
    }
}
```

### Tracing

Tekton integrates OpenTelemetry:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) error {
    tracer := otel.Tracer("tekton-pipelines")

    ctx, span := tracer.Start(ctx, "ReconcilePipelineRun",
        trace.WithAttributes(
            attribute.String("namespace", pr.Namespace),
            attribute.String("name", pr.Name),
        ),
    )
    defer span.End()

    // Reconcile logic
    if err := r.reconcile(ctx, pr); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    span.SetStatus(codes.Ok, "Reconciliation completed")
    return nil
}
```

### Recommendations for AgentRun

1. **Use structured logging**
   - Include run_id, agent, phase, iteration in all logs
   - Log tool calls with latency, tokens, decision_hash

2. **Implement metrics**
   - agentrun_total (counter)
   - agentrun_duration_seconds (histogram)
   - agentrun_iterations (histogram)
   - agentrun_tool_calls_total (counter by tool name)
   - agentrun_llm_tokens_total (counter by direction: in/out)

3. **Add tracing**
   - Span per reconciliation
   - Child spans for: plan, tool calls, reflect
   - Export to in-cluster collector

---

## 10. Security Patterns

### SecurityContext for Pods

From Tekton's pod creation:

```go
securityContext := &corev1.PodSecurityContext{
    RunAsNonRoot: &[]bool{true}[0],
    RunAsUser:    &[]int64{65532}[0],  // nonroot user
    FSGroup:      &[]int64{65532}[0],
}

if cfg.FeatureFlags.SetSecurityContext {
    securityContext.SeccompProfile = &corev1.SeccompProfile{
        Type: corev1.SeccompProfileTypeRuntimeDefault,
    }
}

pod := &corev1.Pod{
    Spec: corev1.PodSpec{
        SecurityContext: securityContext,
        Containers: []corev1.Container{
            {
                SecurityContext: &corev1.SecurityContext{
                    AllowPrivilegeEscalation: &[]bool{false}[0],
                    Capabilities: &corev1.Capabilities{
                        Drop: []corev1.Capability{"ALL"},
                    },
                    RunAsNonRoot: &[]bool{true}[0],
                    SeccompProfile: &corev1.SeccompProfile{
                        Type: corev1.SeccompProfileTypeRuntimeDefault,
                    },
                },
            },
        },
    },
}
```

### NetworkPolicy

Tekton doesn't create NetworkPolicies by default, but OpenShift integration does:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tekton-pipelines-controller
spec:
  podSelector:
    matchLabels:
      app: tekton-pipelines-controller
  policyTypes:
    - Ingress
    - Egress
  egress:
    # Kubernetes API
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
    # DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53
```

### Recommendations for AgentRun

1. **Pod SecurityContext**
   ```go
   securityContext := &corev1.PodSecurityContext{
       RunAsNonRoot: ptr.Bool(true),
       RunAsUser:    ptr.Int64(65532),
       FSGroup:      ptr.Int64(65532),
       SeccompProfile: &corev1.SeccompProfile{
           Type: corev1.SeccompProfileTypeRuntimeDefault,
       },
   }
   ```

2. **Container SecurityContext**
   ```go
   securityContext := &corev1.SecurityContext{
       AllowPrivilegeEscalation: ptr.Bool(false),
       Capabilities: &corev1.Capabilities{
           Drop: []corev1.Capability{"ALL"},
       },
       RunAsNonRoot: ptr.Bool(true),
       ReadOnlyRootFilesystem: ptr.Bool(true),
   }
   ```

3. **NetworkPolicy per AgentRun**
   - Create dynamically in reconciler
   - Default deny ingress
   - Whitelist egress: K8s API, LLM endpoints
   - Use label selectors to target agent pods

4. **RBAC per AgentRun**
   - Generate Role with limited verbs (get, list, watch)
   - Scoped to safe resources (no secrets)
   - Bind to ServiceAccount specified in AgentConfig

---

## 11. Pod Creation Patterns

### Builder Pattern

From `pkg/pod/pod.go`:

```go
type Builder struct {
    Images          map[string]string
    KubeClient      kubernetes.Interface
    EntrypointCache EntrypointCache
}

func (b *Builder) Build(ctx context.Context, taskRun *v1beta1.TaskRun, task *v1beta1.Task) (*corev1.Pod, error) {
    pod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      taskRun.Name,
            Namespace: taskRun.Namespace,
            Labels: map[string]string{
                pipeline.TaskRunLabelKey: taskRun.Name,
            },
            OwnerReferences: []metav1.OwnerReference{
                *metav1.NewControllerRef(taskRun, v1beta1.SchemeGroupVersion.WithKind("TaskRun")),
            },
        },
        Spec: corev1.PodSpec{
            ServiceAccountName: taskRun.Spec.ServiceAccountName,
            RestartPolicy:      corev1.RestartPolicyNever,
        },
    }

    // Apply security context
    applySecurityContext(pod, task)

    // Add init containers
    pod.Spec.InitContainers = b.buildInitContainers(task)

    // Add step containers
    pod.Spec.Containers = b.buildStepContainers(task)

    // Add volumes
    pod.Spec.Volumes = b.buildVolumes(task)

    return pod, nil
}
```

### Owner References

```go
ownerRef := metav1.OwnerReference{
    APIVersion:         v1alpha1.SchemeGroupVersion.String(),
    Kind:               "AgentRun",
    Name:               agentRun.Name,
    UID:                agentRun.UID,
    Controller:         ptr.Bool(true),
    BlockOwnerDeletion: ptr.Bool(true),
}

pod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerRef}
```

This enables automatic garbage collection - when AgentRun is deleted, owned pods are deleted too.

### Recommendations for AgentRun

1. **Create pod builder utility**
   ```go
   // pkg/pod/builder.go
   type Builder struct {
       Images map[string]string
   }

   func (b *Builder) Build(ctx context.Context, ar *v1alpha1.AgentRun, config *v1alpha1.AgentConfig) (*corev1.Pod, error) {
       // Build pod spec
   }
   ```

2. **Use owner references**
   - Set AgentRun as controller
   - Enables automatic cleanup

3. **Label consistently**
   ```go
   labels := map[string]string{
       "agent.tekton.dev/agentrun": ar.Name,
       "agent.tekton.dev/config":   config.Name,
   }
   ```

---

## 12. Example Custom Task Implementations

### 1. tekton-task-group (openshift-pipelines)

**Repository**: https://github.com/openshift-pipelines/tekton-task-group

**Structure**:
```
tekton-task-group/
├── cmd/controller/
├── pkg/
│   ├── apis/taskgroup/v1alpha1/
│   │   ├── taskgroup_types.go
│   │   ├── taskgroup_validation.go
│   │   └── taskgroup_defaults.go
│   └── reconciler/taskgroup/
│       ├── controller.go
│       └── taskgroup.go
├── config/
└── examples/
```

**Key Insights**:
- Simple CustomRun controller that groups tasks
- Creates multiple TaskRuns from a single CustomRun
- Aggregates status from child TaskRuns
- Uses owner references for lifecycle management

### 2. tekton-custom-task (KubeRocketCI)

**Repository**: https://github.com/KubeRocketCI/tekton-custom-task

**Structure**:
```
tekton-custom-task/
├── api/v1alpha1/
│   ├── approvaltask_types.go
│   └── groupversion_info.go
├── cmd/
├── internal/controller/
│   ├── customrun_controller.go
│   └── customrun_controller_test.go
├── config/
├── deploy-templates/  # Helm charts
└── docs/
```

**Key Insights**:
- Implements approval workflow as CustomRun
- Uses kubebuilder for scaffolding
- Includes Helm chart for deployment
- OLM bundle for OpenShift

### 3. Tekton Pipeline Internal Custom Tasks

**Location**: `test/custom-task-ctrls/` in tektoncd/pipeline

**Purpose**: Test controllers for validation

**Patterns**:
- Minimal controller implementation
- Focus on status updates
- Used in conformance tests

---

## 13. Gaps in Current Plan (claude.md)

### Comparing claude.md to Tekton Conventions

#### 1. Directory Structure

**Gap**: claude.md doesn't specify detailed directory structure.

**Recommendation**: Follow Tekton's layout (see Section 1).

#### 2. API Versioning

**Current**: Plan mentions "v1alpha1" but not versioning strategy.

**Gap**: No plan for v1beta1 or v1 graduation.

**Recommendation**:
- Start with v1alpha1
- Define graduation criteria (API stability, adoption)
- Plan conversion webhooks for future versions

#### 3. Code Generation

**Current**: Phase 0 mentions "Go module scaffolding" but no code generation specifics.

**Gap**: No mention of:
- `hack/update-codegen.sh`
- Generated clients, informers, listers
- Knative injection
- CRD generation from Go types

**Recommendation**: Add to Phase 0:
- Set up code generation scripts
- Add Makefile targets
- Document generation workflow

#### 4. Testing Strategy

**Current**: "Development Phases" mention testing but no structure.

**Gap**:
- No unit test structure
- No E2E test plan
- No test utilities

**Recommendation**: Add to Phase 1:
- Create `test/` directory with fixtures
- Implement fake client utilities
- Add `*_test.go` files for all reconcilers
- Plan E2E tests for Phase 3

#### 5. Configuration Management

**Current**: References "feature flags" but no implementation.

**Gap**:
- No ConfigMap structure
- No config store pattern
- No context-based config loading

**Recommendation**: Add to Phase 1:
- Create `pkg/apis/config/` package
- Implement ConfigStore
- Watch ConfigMaps in controller

#### 6. Webhook Strategy

**Current**: Mentions webhooks but unclear if needed for MVP.

**Gap**: No decision on:
- Synchronous vs asynchronous validation
- Defaulting strategy
- Webhook deployment complexity

**Recommendation**:
- MVP: Skip webhooks, validate asynchronously in reconciler
- Phase 4: Add webhooks if needed

#### 7. RBAC Details

**Current**: Mentions RBAC generation but no specifics.

**Gap**:
- No ClusterRole definition for controller
- No per-AgentRun Role template
- No ServiceAccount strategy

**Recommendation**: Add to Phase 0:
- Define controller ClusterRole (see Section 5)
- Create RBAC generator for agent pods
- Document permission boundaries

#### 8. Status Subresource

**Current**: Status phases defined but no subresource specification.

**Gap**: CRD should use status subresource for proper RBAC.

**Recommendation**: Add to CRD:
```yaml
subresources:
  status: {}
```

#### 9. Observability Details

**Current**: Phase 3 mentions "structured logging" and "OTel tracing" but no specifics.

**Gap**:
- No metric definitions
- No trace span structure
- No logging format

**Recommendation**: Add to Phase 3:
- Define metrics (see Section 9)
- Implement trace spans per phase
- Use structured logging with consistent fields

#### 10. Pod Creation Patterns

**Current**: Mentions "Agent pod starts" but no builder pattern.

**Gap**:
- No pod builder utility
- No security context details
- No owner reference strategy

**Recommendation**: Add to Phase 1:
- Create `pkg/pod/builder.go`
- Implement security contexts (see Section 10)
- Use owner references for garbage collection

#### 11. CustomRun vs Direct Pod

**Current**: Plan uses CustomRun but also mentions "Agent pod".

**Gap**: Unclear relationship between CustomRun and Pod.

**Options**:

**Option A**: AgentRun → CustomRun → Pod (current plan)
- AgentRun controller creates CustomRun
- CustomRun controller creates Pod
- Two reconcilers

**Option B**: AgentRun → Pod (simpler)
- AgentRun controller creates Pod directly
- One reconciler
- No Tekton CustomRun dependency

**Recommendation**:
- **Use Option B for MVP** (simpler, fewer moving parts)
- CustomRun is useful if:
  - Integrating with Tekton Pipelines
  - Reusing Tekton's hook infrastructure
  - Need Tekton Results integration

If not using Tekton Pipelines features, direct Pod creation is cleaner.

#### 12. Hook Implementation

**Current**: Mentions "preHooks" and "postHooks" as Tekton Tasks.

**Gap**: How are hooks orchestrated?

**Options**:

**Option A**: Use Tekton Pipeline (requires CustomRun)
```yaml
Pipeline:
  tasks:
    - name: prehook
      taskRef: llm-security-check
    - name: agent
      runAfter: [prehook]
      customRunRef: agentrun
    - name: posthook
      runAfter: [agent]
      taskRef: audit-bundle
```

**Option B**: Orchestrate in AgentRun controller
- Controller runs hooks as separate Pods
- Phases: PreHooks → Acting → PostHooks
- More control, no Tekton dependency

**Recommendation**:
- **Use Option B for MVP**
- Simpler implementation
- Can add Tekton integration later

#### 13. LLM Provider Sidecar

**Current**: Mentions "provider sidecar" for API keys.

**Gap**: No implementation details.

**Questions**:
- Why sidecar vs main container?
- How do containers communicate?
- What does sidecar do?

**Recommendation**:
- **Simplify for MVP**: Use main container with Secret volume mount
- Sidecar adds complexity without clear benefit
- If needed, add in Phase 2

#### 14. OPA Integration

**Current**: Mentions OPA policy engine.

**Gap**: How is OPA deployed and integrated?

**Options**:

**Option A**: OPA sidecar in agent pod
- OPA container alongside agent
- Agent queries OPA via localhost

**Option B**: Centralized OPA server
- OPA deployed separately
- Agent queries via network

**Option C**: OPA library (rego in Go)
- Embed OPA in agent binary
- No extra container

**Recommendation**:
- **Use Option C for MVP** (simplest)
- No network calls, no sidecar
- Can switch to sidecar later if policies need hot-reload

#### 15. PVC Structure

**Current**: Mentions config PVC and data PVC.

**Gap**: Who creates PVCs? Lifecycle?

**Recommendation**:
- Config PVC: Pre-created by user (contains prompts, schemas)
- Data PVC: Created by controller per AgentRun (ephemeral)
- Add PVC creation to reconciler Phase 1

#### 16. Iteration Limit

**Current**: "Max 3 iterations" hardcoded.

**Gap**: Should be configurable.

**Recommendation**: Add to AgentConfig:
```go
type AgentConfigSpec struct {
    MaxIterations int32 `json:"maxIterations,omitempty"`
}
```

Default: 3, max: 10

#### 17. Result Handling

**Current**: No mention of how results are returned.

**Gap**: Where do agent findings go?

**Recommendation**: Add to AgentRunStatus:
```go
type AgentRunStatus struct {
    // ... conditions, phases

    Results []AgentResult `json:"results,omitempty"`
}

type AgentResult struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}
```

Store summary in status, full audit in PVC.

#### 18. Error Handling Strategy

**Current**: No error handling patterns.

**Gap**: How to handle:
- LLM API failures (transient)
- Policy violations (permanent)
- Timeout (terminal)
- OOM (terminal)

**Recommendation**: Use Tekton's pattern:
- Transient errors: Return error from ReconcileKind (requeue)
- Permanent errors: Set status Failed, return nil
- Use `controller.NewPermanentError()` for permanent failures

#### 19. Event Emission

**Current**: No mention of Kubernetes events.

**Gap**: How to notify users of state changes?

**Recommendation**: Add to Phase 1:
- Emit events on phase transitions
- Use `recorder.Event()` from controller
- Example: "AgentRun started", "Planning completed", "Failed: timeout"

#### 20. Documentation Structure

**Current**: Phase 4 mentions "Documentation" but no structure.

**Gap**: What docs are needed?

**Recommendation**: Create docs/:
- `installation.md` - How to install controller
- `agentconfig.md` - AgentConfig CRD reference
- `agentrun.md` - AgentRun CRD reference
- `security.md` - Security model
- `tools.md` - Available tools
- `examples/` - Example manifests

---

## 14. Specific Recommendations for agentrun-controller

### Phase 0: Foundation

**Add to phase 0**:

1. **Directory structure** (see Section 1)
   - Create `pkg/apis/agent/v1alpha1/`
   - Create `pkg/reconciler/agentrun/`
   - Create `config/`, `test/`, `hack/`

2. **Code generation setup**
   - Add `hack/update-codegen.sh`
   - Add `hack/update-deps.sh`
   - Add Makefile targets
   - Run generation to create clients

3. **API types**
   - `agentconfig_types.go` with full spec
   - `agentrun_types.go` with full spec
   - `*_validation.go` for both
   - `*_defaults.go` for both
   - Add kubebuilder markers

4. **RBAC definitions**
   - Controller ClusterRole
   - Agent Role template (generated per AgentRun)
   - ServiceAccount

5. **CRD manifests**
   - Generate from Go types
   - Include status subresource
   - Add to `config/300-*.yaml`

### Phase 1: Core Loop

**Refine phase 1**:

1. **Simplify architecture**
   - Remove CustomRun layer (use direct Pod creation)
   - Remove sidecar pattern (use main container)
   - Use OPA library (not sidecar)

2. **Controller setup**
   - Implement `pkg/reconciler/agentrun/controller.go`
   - Use Knative injection
   - Watch AgentRuns and Pods

3. **Reconciler implementation**
   - Implement `ReconcileKind`
   - State machine: Pending → PreHooks → Acting → Reflecting → PostHooks
   - Pod creation with builder pattern
   - Status updates

4. **Pod builder**
   - Create `pkg/pod/builder.go`
   - Apply security contexts
   - Owner references
   - Volume mounts (config PVC, data PVC)

5. **Agent runtime**
   - Plan-act-reflect loop in Go
   - Claude provider integration
   - k8s_get_resources tool
   - k8s_get_logs tool
   - OPA policy enforcement (embedded)

6. **Testing**
   - Create `test/controller.go` with fake client helpers
   - Unit tests for reconciler
   - Table-driven test pattern

### Phase 2: Security Hardening

**Refine phase 2**:

1. **PreHook implementation**
   - Controller creates PreHook Pods (not Tekton Tasks)
   - llm-security-check as simple Go binary
   - Runs before agent pod

2. **PostHook implementation**
   - Controller creates PostHook Pods
   - audit-bundle as simple Go binary
   - Collects logs from agent pod

3. **NetworkPolicy generation**
   - Create in reconciler
   - Default deny + whitelist pattern

4. **RBAC generation**
   - Generate Role per AgentRun
   - Bind to ServiceAccount from AgentConfig

5. **Security contexts**
   - Non-root, read-only root filesystem
   - Drop all capabilities
   - Seccomp profile

### Phase 3: Observability

**Refine phase 3**:

1. **Structured logging**
   - Use Knative's logger
   - Consistent fields: run_id, phase, iteration

2. **Metrics**
   - Define Prometheus metrics
   - Record in reconciler

3. **Tracing**
   - OTel spans per phase
   - Export to collector

4. **Audit bundle**
   - Write to PVC
   - Include tool calls, decisions, policy verdicts

### Phase 4: Packaging

**Refine phase 4**:

1. **Helm chart**
   - Chart structure
   - Controller deployment
   - RBAC
   - ConfigMap for defaults
   - CRDs

2. **Documentation**
   - Installation guide
   - API reference
   - Security model
   - Examples

3. **Release process**
   - Container image build
   - GitHub releases
   - Helm chart versioning

### Phase 5: Extended Tools

**No changes needed** - looks good!

---

## 15. Tekton Dependencies

### Do We Need Tekton Pipelines?

**Current plan**: Uses CustomRun (Tekton API)

**Analysis**:

**Pros of using Tekton**:
- Integrates with existing Tekton installations
- Can reference AgentRun from Pipeline
- Reuse Tekton's hook mechanisms (preHooks/postHooks as Tasks)
- Tekton Results integration for audit
- Shared RBAC, webhook, and controller patterns

**Cons of using Tekton**:
- Requires Tekton Pipelines installed
- More complex (CustomRun → Pod)
- Extra dependency to maintain

**Recommendation**:

**For MVP**: Don't depend on Tekton
- Create standalone controller
- AgentRun → Pod directly
- Simpler installation
- Fewer moving parts

**For future**: Add Tekton integration
- Implement CustomRun controller that wraps AgentRun
- Users can choose: standalone or Tekton-integrated
- Best of both worlds

**Implementation path**:

```
MVP:
  AgentRun (standalone CRD)
    ↓
  Controller creates Pod directly
    ↓
  Agent runs in Pod

Future:
  PipelineRun
    ↓
  CustomRun (kind: AgentRun)
    ↓
  AgentRun CRD
    ↓
  Pod
```

This allows:
1. Standalone usage (simpler)
2. Tekton integration (more powerful)

---

## 16. Key Takeaways

### What to Adopt from Tekton

1. **Directory structure** - Follow exactly
2. **File naming** - `{resource}_types.go`, `*_validation.go`, `*_defaults.go`
3. **Code generation** - Use k8s.io/code-generator and Knative injection
4. **Controller patterns** - ReconcileKind, state checks, status updates
5. **Testing patterns** - Table-driven, fake clients, test utilities
6. **RBAC structure** - Separate cluster/tenant roles
7. **Security contexts** - Non-root, capabilities drop, seccomp
8. **Owner references** - For automatic garbage collection
9. **Configuration management** - ConfigStore, context-based loading
10. **Observability** - Structured logging, metrics, tracing

### What to Simplify for MVP

1. **No CustomRun dependency** - Direct Pod creation
2. **No webhooks** - Asynchronous validation
3. **No sidecar pattern** - Main container only
4. **Embedded OPA** - No OPA sidecar
5. **Simple hooks** - Pods, not Tekton Tasks
6. **Minimal metrics** - Add more in Phase 3

### What to Add to Current Plan

1. **Code generation setup** - Phase 0
2. **Test structure** - Phase 1
3. **Config management** - Phase 1
4. **Pod builder utility** - Phase 1
5. **Event emission** - Phase 1
6. **Error handling strategy** - Phase 1
7. **Result handling** - Phase 2
8. **Documentation structure** - Phase 4

---

## Conclusion

The tektoncd/pipeline repository demonstrates mature patterns for building Kubernetes controllers, especially for CustomRun implementations. The key insights are:

1. **Follow conventions strictly** - Directory structure, file naming, markers
2. **Use code generation** - Reduces boilerplate, ensures consistency
3. **Test thoroughly** - Table-driven unit tests, E2E tests with real cluster
4. **Security by default** - Non-root, capabilities drop, NetworkPolicy
5. **Observability first-class** - Structured logs, metrics, traces
6. **Configuration via ConfigMap** - Feature flags, defaults

For agentrun-controller, adopting these patterns will result in a production-ready, Tekton-compatible controller. The main simplification for MVP is removing the Tekton dependency (CustomRun) and implementing a standalone controller, with the option to add Tekton integration later.

This approach balances:
- **Simplicity** for MVP (fewer dependencies)
- **Quality** (following proven patterns)
- **Extensibility** (can add Tekton integration later)

---

## References

- Tekton Pipeline Repository: https://github.com/tektoncd/pipeline
- Tekton CustomRuns Documentation: https://tekton.dev/docs/pipelines/customruns/
- Tekton Custom Tasks TEP: https://github.com/tektoncd/community/blob/main/teps/0002-custom-tasks.md
- openshift-pipelines/tekton-task-group: https://github.com/openshift-pipelines/tekton-task-group
- KubeRocketCI/tekton-custom-task: https://github.com/KubeRocketCI/tekton-custom-task
- Kubernetes Code Generator: https://github.com/kubernetes/code-generator
- Knative Injection: https://github.com/knative/pkg/tree/main/injection
