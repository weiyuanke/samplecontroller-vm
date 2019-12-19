package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/labels"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	"k8s.io/klog"
	samplev1alpha1 "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"
	clientset "k8s.io/sample-controller/pkg/generated/clientset/versioned"
	samplescheme "k8s.io/sample-controller/pkg/generated/clientset/versioned/scheme"
	informers "k8s.io/sample-controller/pkg/generated/informers/externalversions/samplecontroller/v1alpha1"
	listers "k8s.io/sample-controller/pkg/generated/listers/samplecontroller/v1alpha1"
)

const controllerAgentName = "sample-controller"
const VMFinalizerName = "vmFinalizer"

type Controller struct {
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset clientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	vmsLister         listers.VMLister
	vmsSynced         cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder record.EventRecorder

	// vm mock manager
	vmManager *MockManager
}

func NewController(
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	vmInformer informers.VMInformer) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	// set Custom Metrics provider
	customMetricsProvider := &workqueueCustomMetricsProvider{}
	customMetricsProvider.Register(prometheus.DefaultRegisterer)

	controller := &Controller{
		kubeclientset:     kubeclientset,
		sampleclientset:   sampleclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		vmsLister:         vmInformer.Lister(),
		vmsSynced:         vmInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "vms"),
		recorder:          recorder,
		vmManager:         NewMockManager(),
	}

	// Set up an event handler for when VM resources change
	vmInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.onVMAdd,
		UpdateFunc: controller.onVMUpdate,
	})

	return controller
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting VM controller")

	// Wait for the caches to be synced before starting workers
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.vmsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	//periodically print currentState
	go wait.Until(c.printCurrentVmState, time.Second * 5, stopCh)

	//periodically update vm cpu utilization
	go wait.Until(c.updateVmCpuUtilization, time.Second * 5, stopCh)

	klog.Info("Starting workers")
	// Launch two workers to process VM resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started 2 workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the work queue and
// attempt to process it, by calling the syncVmHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		// Run the syncVmHandler, passing it the namespace/name string of the Vm resource to be synced.
		if err := c.syncVmHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncVmHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the VM resource
// with the current status of the resource.
func (c *Controller) syncVmHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the VM resource with this namespace/name
	vm, err := c.vmsLister.VMs(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("VM '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	if vm.Spec.VMName == "" {
		utilruntime.HandleError(fmt.Errorf("%s: vmname name must be specified", key))
		return nil
	}

	switch vm.Status.OpStatus {
	case samplev1alpha1.NewState:
		err = c.createVmAndUpdateCrStatus(vm)
	case samplev1alpha1.FailedState, samplev1alpha1.UnknownState:
		klog.Infof("VM %s in %s, try to create again", key, vm.Status.OpStatus)
		err = c.createVmAndUpdateCrStatus(vm)
	case samplev1alpha1.RunningState:
		if vm.Status.VMId == "" {
			err = c.createVmAndUpdateCrStatus(vm)
		} else {
			vmResponse := c.vmManager.GetVm(vm.Status.VMId)
			if !vmResponse.success {
				err = c.createVmAndUpdateCrStatus(vm)
			}
		}
	}

	return err
}

func (c *Controller) onVMAdd(obj interface{}) {
	vm := obj.(*samplev1alpha1.VM)

	spaceKey, err := cache.MetaNamespaceKeyFunc(vm)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	//add finalizer to do pre delete actions
	deepCopy := vm.DeepCopy()
	deepCopy.Finalizers = []string{VMFinalizerName}
	_, updateErr := c.sampleclientset.SamplecontrollerV1alpha1().VMs(deepCopy.Namespace).Update(deepCopy)
	if updateErr != nil {
		utilruntime.HandleError(updateErr)
		return
	}

	klog.Infof("VM '%s' was added, enqueue it for submission.", spaceKey)
	c.workqueue.Add(spaceKey)
}

func (c *Controller) onVMUpdate(oldObj, newObj interface{}) {
	oldVm := oldObj.(*samplev1alpha1.VM)
	newVM := newObj.(*samplev1alpha1.VM)

	spaceKey, err := cache.MetaNamespaceKeyFunc(newVM)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	//pre delete actions
	if !newVM.ObjectMeta.DeletionTimestamp.IsZero() {
		vmRes := c.vmManager.DeleteVm(newVM.Status.VMId)
		if vmRes.success {
			deepCopy := newVM.DeepCopy()
			deepCopy.Finalizers = []string{}
			_, _ = c.sampleclientset.SamplecontrollerV1alpha1().VMs(newVM.Namespace).Update(deepCopy)
		}
		return
	}

	if oldVm.ResourceVersion == newVM.ResourceVersion {
		return
	}

	if oldVm.Name == newVM.Name && oldVm.Spec.VMName == newVM.Spec.VMName {
		return
	}

	klog.Infof("VM '%s' was updated, enqueue it for update.", spaceKey)
	c.workqueue.Add(spaceKey)
}

func (c *Controller) createVmAndUpdateCrStatus(vm *samplev1alpha1.VM) error {
	//vm name check
	if !c.vmManager.CheckNameIsOk(vm.Spec.VMName) {
		vmStatus := &samplev1alpha1.VMStatus{
			OpStatus: samplev1alpha1.FailedState,
			Msg: "checkName failed",
			CPUUtilization: "",
			VMId: "",
		}
		_ = c.updateVmStatus(vm, vmStatus)
		return fmt.Errorf("checkName failed: " + vm.Spec.VMName)
	}

	vmResponse := c.vmManager.CreateVm(vm)
	if vmResponse.success {
		vmStatus := &samplev1alpha1.VMStatus{
			OpStatus: samplev1alpha1.RunningState,
			VMId: vmResponse.vmEntity.id,
			CPUUtilization: vmResponse.vmEntity.cpuUtilization,
			Msg: "CreateVm success",
		}
		err := c.updateVmStatus(vm, vmStatus)
		return err
	} else {
		vmStatus := &samplev1alpha1.VMStatus{
			OpStatus: samplev1alpha1.FailedState,
			Msg: vmResponse.msg,
			VMId: "",
			CPUUtilization: "",
		}
		_ = c.updateVmStatus(vm, vmStatus)
		return fmt.Errorf("error msg: %s", vmResponse.msg)
	}
}

// update Status of Vm Resource field
func (c *Controller) updateVmStatus(vm *samplev1alpha1.VM, vmStatus *samplev1alpha1.VMStatus) error {
	vmCopy := vm.DeepCopy()
	vmCopy.Status = *vmStatus
	_, err := c.sampleclientset.SamplecontrollerV1alpha1().VMs(vm.Namespace).UpdateStatus(vmCopy)
	return err
}

func (c *Controller) printCurrentVmState() {
	klog.Info("")
	klog.Info("----------------------VMList-----------------------------")
	c.vmManager.PrintVmListResponse(c.vmManager.GetVmList())
	klog.Info("")
	klog.Info("")
}

// periodically update the VM cpu utilization and update the Status field
func (c *Controller) updateVmCpuUtilization() {
	var ret []*samplev1alpha1.VM
	var err error

	ret, err = c.vmsLister.List(labels.Everything())
	if err != nil {
		return
	}

	for _, val := range ret {
		if val.Status.VMId == "" {
			continue
		}

		vmResponse := c.vmManager.GetVm(val.Status.VMId)
		if !vmResponse.success {
			continue
		}

		vmStatus := &samplev1alpha1.VMStatus{
			VMId:           val.Status.VMId,
			CPUUtilization: vmResponse.vmEntity.cpuUtilization,
			OpStatus:       val.Status.OpStatus,
			Msg:            val.Status.Msg,
		}
		_ = c.updateVmStatus(val, vmStatus)
	}
}
