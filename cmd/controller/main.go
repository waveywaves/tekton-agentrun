package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/waveywaves/agentrun-controller/pkg/apis/agent/v1alpha1"
	"github.com/waveywaves/agentrun-controller/pkg/reconciler/agentrun"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL  string
	kubeconfig string
	image      string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (optional, defaults to in-cluster config)")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (optional)")
	flag.StringVar(&image, "agent-image", "ko://github.com/waveywaves/agentrun-controller/cmd/agent", "Agent runtime image")
	flag.Parse()

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal, stopping controller...")
		cancel()
	}()

	// Build Kubernetes config
	cfg, err := buildConfig(kubeconfig, masterURL)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	// Create Kubernetes client
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes client: %v", err)
	}

	// Create dynamic client for AgentRun CRDs
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building dynamic client: %v", err)
	}

	// Register our CRD types with the scheme
	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalf("Error adding types to scheme: %v", err)
	}

	// GVR for AgentRun
	agentRunGVR := schema.GroupVersionResource{
		Group:    "agent.tekton.dev",
		Version:  "v1alpha1",
		Resource: "agentruns",
	}

	// GVR for AgentConfig
	agentConfigGVR := schema.GroupVersionResource{
		Group:    "agent.tekton.dev",
		Version:  "v1alpha1",
		Resource: "agentconfigs",
	}

	// Create reconciler
	reconciler := &agentrun.Reconciler{
		KubeClient: kubeClient,
		Image:      image,
	}
	log.Printf("Reconciler initialized with image: %s", reconciler.Image)

	// Create informer factory
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)

	// Create pod informer
	podInformer := informerFactory.Core().V1().Pods()

	// Set up pod event handlers (to update AgentRun status when pods complete)
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// This is a simplified handler - in production, check labels and enqueue the owning AgentRun
			log.Println("Pod updated")
		},
	})

	// Start informers
	informerFactory.Start(ctx.Done())

	// Wait for cache sync
	log.Println("Waiting for informer caches to sync...")
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.Informer().HasSynced) {
		log.Fatal("Failed to sync informer caches")
	}

	log.Printf("AgentRun Controller started (agent image: %s)", image)
	log.Println("Watching for AgentRun resources...")

	// Run reconciliation loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, shutting down")
			return
		case <-ticker.C:
			// List all AgentRuns
			agentRuns, err := dynamicClient.Resource(agentRunGVR).Namespace("").List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Printf("Error listing AgentRuns: %v", err)
				continue
			}

			// Reconcile each AgentRun
			for _, item := range agentRuns.Items {
				if err := reconcileAgentRun(ctx, reconciler, dynamicClient, agentRunGVR, agentConfigGVR, &item); err != nil {
					log.Printf("Error reconciling AgentRun %s/%s: %v", item.GetNamespace(), item.GetName(), err)
				}
			}
		}
	}
}

func reconcileAgentRun(ctx context.Context, reconciler *agentrun.Reconciler, dynamicClient dynamic.Interface, agentRunGVR, agentConfigGVR schema.GroupVersionResource, unstr *unstructured.Unstructured) error {
	// Convert unstructured to AgentRun
	var ar v1alpha1.AgentRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.Object, &ar); err != nil {
		return err
	}

	// Skip if already done
	if ar.IsDone() {
		return nil
	}

	// Ensure phase is set
	if ar.Status.Phase == "" {
		ar.Status.Phase = v1alpha1.AgentRunPhasePending
	}

	// Get AgentConfig
	agentConfigUnstr, err := dynamicClient.Resource(agentConfigGVR).Namespace(ar.Namespace).Get(ctx, ar.Spec.ConfigRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get AgentConfig %s/%s: %v", ar.Namespace, ar.Spec.ConfigRef.Name, err)
		return err
	}

	var agentConfig v1alpha1.AgentConfig
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(agentConfigUnstr.Object, &agentConfig); err != nil {
		return err
	}

	// Store in reconciler's map (temporary solution)
	if reconciler.AgentConfigs == nil {
		reconciler.AgentConfigs = make(map[string]*v1alpha1.AgentConfig)
	}
	reconciler.AgentConfigs[agentConfig.Name] = &agentConfig

	// Reconcile
	if err := reconciler.Reconcile(ctx, &ar); err != nil {
		log.Printf("Reconciliation error for %s/%s: %v", ar.Namespace, ar.Name, err)
		return err
	}

	// Convert back to unstructured and update status
	arUnstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ar)
	if err != nil {
		return err
	}

	// Update status subresource
	unstr.Object["status"] = arUnstr["status"]
	_, err = dynamicClient.Resource(agentRunGVR).Namespace(ar.Namespace).UpdateStatus(ctx, unstr, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update status for %s/%s: %v", ar.Namespace, ar.Name, err)
		return err
	}

	log.Printf("Reconciled AgentRun %s/%s: phase=%s iterations=%d", ar.Namespace, ar.Name, ar.Status.Phase, ar.Status.Iterations)
	return nil
}

func buildConfig(kubeconfig, masterURL string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	}
	return rest.InClusterConfig()
}

func init() {
	// Add our types to the default Kubernetes scheme
	v1alpha1.AddToScheme(scheme.Scheme)
}
