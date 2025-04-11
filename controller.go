package main

import (
	"context"
	"fmt"
	"golang.org/x/time/rate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"

	customv1 "github.com/refat75/apiServerController/pkg/apis/appscode.refat.dev/v1"
	clientset "github.com/refat75/apiServerController/pkg/generated/clientset/versioned"
	samplescheme "github.com/refat75/apiServerController/pkg/generated/clientset/versioned/scheme"
	informers "github.com/refat75/apiServerController/pkg/generated/informers/externalversions/appscode.refat.dev/v1"
	listers "github.com/refat75/apiServerController/pkg/generated/listers/appscode.refat.dev/v1"
)

const controllerAgentName = "apiServerController"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Apiserver is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Apiserver fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Apiserver"
	// MessageResourceSynced is the message used for an Event fired when a Apiserver
	// is synced successfully
	MessageResourceSynced = "Foo synced successfully"
	// FieldManager distinguishes this controller from other things writing to API objects
	FieldManager = controllerAgentName
)

// Controller is the controller implementation for Apiserver resources
type Controller struct {
	// kubeclientset is the standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset clientset.Interface

	deploymentLister appslisters.DeploymentLister
	deploymentSynced cache.InformerSynced
	apiServerLister  listers.ApiserverLister
	apiServerSynced  cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.TypedRateLimitingInterface[cache.ObjectName]

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	apiserverInformer informers.ApiserverInformer) *Controller {
	logger := klog.FromContext(ctx)

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	logger.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	ratelimiter := workqueue.NewTypedMaxOfRateLimiter(
		workqueue.NewTypedItemExponentialFailureRateLimiter[cache.ObjectName](5*time.Microsecond, 1000*time.Second),
		&workqueue.TypedBucketRateLimiter[cache.ObjectName]{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)

	controller := &Controller{
		kubeclientset:    kubeclientset,
		sampleclientset:  sampleclientset,
		deploymentLister: deploymentInformer.Lister(),
		deploymentSynced: deploymentInformer.Informer().HasSynced,
		apiServerLister:  apiserverInformer.Lister(),
		apiServerSynced:  apiserverInformer.Informer().HasSynced,
		workqueue:        workqueue.NewTypedRateLimitingQueue(ratelimiter),
		recorder:         recorder,
	}

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Apiserver resources change
	apiserverInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueApiServer,
		UpdateFunc: func(old, new interface{}) {
			//controller.enqueueApiServer(new)
			controller.enqueueApiServer(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	// Start the informer factories to begin populating the informer caches
	logger.Info("Starting API Server Controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentSynced, c.apiServerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runworker, time.Second)
	}

	logger.Info("Started Workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")
	return nil
}

func (c *Controller) runworker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {

	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	objRef, shutdown := c.workqueue.Get()
	logger := klog.FromContext(ctx)
	logger.Info("Processing work item")

	if shutdown {
		return false
	}

	// We call Done at the end of this func so the workqueue knows we have
	// finished processing this item. We also must remember to call Forget
	// if we do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the workqueue and attempted again after a back-off
	// period.
	defer c.workqueue.Done(objRef)

	err := c.syncHandler(ctx, objRef)
	if err == nil {
		// If no error occurs then we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(objRef)
		logger.Info("Successfully synced", "objectName", objRef)
		return true
	}
	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring.
	utilruntime.HandleErrorWithContext(ctx, err, "Error syncing; requeuing for later retry", "objectReference", objRef)
	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	c.workqueue.AddRateLimited(objRef)
	return true
}

func (c *Controller) syncHandler(ctx context.Context, objectRef cache.ObjectName) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", objectRef)

	//Get the Apiserver resource with this namespace/name
	apiServer, err := c.apiServerLister.Apiservers(objectRef.Namespace).Get(objectRef.Name)
	if err != nil {
		// The apiServer resource may no longer exist, in which case we assume the reource is deleted
		if errors.IsNotFound(err) {
			utilruntime.HandleErrorWithContext(ctx, err, "Failed looking up Apiserver", "objectRef", objectRef)
			// Set deployment owner reference as Apiserver
			// If custom resource is deleted, the deployment will be deleted automatically
			return nil
		}
		return err
	}

	//Get the deployment
	deployment, err := c.deploymentLister.Deployments(apiServer.Namespace).Get(apiServer.Name)

	//If the resource doesn't exist we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(apiServer.Namespace).Create(ctx, newDeployment(apiServer), metav1.CreateOptions{FieldManager: FieldManager})
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		utilruntime.HandleErrorWithContext(ctx, err, "failed to get deployment")
		return err
	}

	//If the current state is not equal to desired state we should update the deployment
	deploymentImage := deployment.Spec.Template.Spec.Containers[0].Image
	deploymentPort := deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort
	if *apiServer.Spec.Replicas != *deployment.Spec.Replicas || apiServer.Spec.Image != deploymentImage || apiServer.Spec.Port != deploymentPort {
		logger.V(4).Info("Updating deployment state to the desired sate")
		deployment, err = c.kubeclientset.AppsV1().Deployments(apiServer.Namespace).Update(ctx, newDeployment(apiServer), metav1.UpdateOptions{FieldManager: FieldManager})
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	logger.V(4).Info("Found Apiserver resource", "objectRef", objectRef)
	c.recorder.Event(apiServer, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) enqueueApiServer(obj interface{}) {
	if objectRef, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		c.workqueue.Add(objectRef)
	}
}

func newDeployment(apiServer *customv1.Apiserver) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "apiServer",
		"controller": apiServer.Name,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServer.Name,
			Namespace: apiServer.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(apiServer, customv1.SchemeGroupVersion.WithKind("Apiserver")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: apiServer.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  string(apiServer.UID),
							Image: apiServer.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: apiServer.Spec.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}
