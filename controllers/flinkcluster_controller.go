/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 10/1/21, 10:31 AM  @author hu.xiaolin3
 */

package controllers

import (
	"context"
	"time"

	flinkclustersv1 "flink-operator/api/v1"
	"flink-operator/controllers/flinkclient"
	"flink-operator/controllers/history"
	"flink-operator/controllers/model"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = flinkclustersv1.GroupVersion.WithKind("FlinkCluster")

// FlinkClusterReconciler reconciles a FlinkCluster object
type FlinkClusterReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Mgr            ctrl.Manager
	WatchNamespace string
}

// +kubebuilder:rbac:groups=daas.tenxcloud.com,resources=flinkclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=daas.tenxcloud.com,resources=flinkclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=daas.tenxcloud.com,resources=flinkclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FlinkCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *FlinkClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = log.FromContext(ctx).WithValues("cluster", req.NamespacedName)

	var handler = FlinkClusterHandler{
		k8sClient:      r.Client,
		watchNamespace: r.WatchNamespace,
		flinkClient: flinkclient.FlinkClient{
			Log:        log,
			HTTPClient: flinkclient.HTTPClient{Log: log},
		},
		request:  req,
		context:  context.Background(),
		log:      log,
		recorder: r.Mgr.GetEventRecorderFor("FlinkOperator"),
		observed: ObservedClusterState{},
	}
	return handler.reconcile(req)
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlinkClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Mgr = mgr
	return ctrl.NewControllerManagedBy(mgr).
		For(&flinkclustersv1.FlinkCluster{}).
		Complete(r)
}

// FlinkClusterHandler holds the context and state for a
// reconcile request.
type FlinkClusterHandler struct {
	watchNamespace string
	k8sClient      client.Client
	flinkClient    flinkclient.FlinkClient
	request        ctrl.Request
	context        context.Context
	log            logr.Logger
	recorder       record.EventRecorder
	observed       ObservedClusterState
	desired        model.DesiredClusterState
}

func (handler *FlinkClusterHandler) reconcile(
	request ctrl.Request) (ctrl.Result, error) {
	var k8sClient = handler.k8sClient
	var flinkClient = handler.flinkClient
	var log = handler.log
	var context = handler.context
	var observed = &handler.observed
	var desired = &handler.desired
	var statusChanged bool
	var err error

	// History interface
	var history = history.NewHistory(k8sClient, context)

	log.Info("============================================================")
	if handler.watchNamespace != "" &&
		handler.watchNamespace != request.Namespace {
		log.Info(
			"Ignore the custom resource.", "watchNamespace", handler.watchNamespace)
		return ctrl.Result{}, nil
	}

	log.Info("---------- 1. Observe the current state ----------")

	var observer = ClusterStateObserver{
		k8sClient:   k8sClient,
		flinkClient: flinkClient,
		request:     request,
		context:     context,
		log:         log,
		history:     history,
	}
	err = observer.observe(observed)
	if err != nil {
		log.Error(err, "Failed to observe the current state")
		return ctrl.Result{}, err
	}

	observed.cluster.Default()

	// Sync history and observe revision status
	err = observer.syncRevisionStatus(observed)
	if err != nil {
		log.Error(err, "Failed to sync flinkCluster history")
		return ctrl.Result{}, err
	}

	log.Info("---------- 2. Update cluster status ----------")

	var updater = ClusterStatusUpdater{
		k8sClient: k8sClient,
		context:   context,
		log:       log,
		recorder:  handler.recorder,
		observed:  handler.observed,
	}
	statusChanged, err = updater.updateStatusIfChanged()
	if err != nil {
		log.Error(err, "Failed to update cluster status")
		return ctrl.Result{}, err
	}
	if statusChanged {
		log.Info(
			"Wait status to be stable before taking further actions.",
			"requeueAfter",
			5)
		return ctrl.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		}, nil
	}

	log.Info("---------- 3. Compute the desired state ----------")

	*desired = getDesiredClusterState(observed)
	if desired.ConfigMap != nil {
		log.Info("Desired state", "ConfigMap", *desired.ConfigMap)
	} else {
		log.Info("Desired state", "ConfigMap", "nil")
	}
	if desired.JmStatefulSet != nil {
		log.Info("Desired state", "JobManager StatefulSet", *desired.JmStatefulSet)
	} else {
		log.Info("Desired state", "JobManager StatefulSet", "nil")
	}
	if desired.JmService != nil {
		log.Info("Desired state", "JobManager service", *desired.JmService)
	} else {
		log.Info("Desired state", "JobManager service", "nil")
	}
	if desired.JmIngress != nil {
		log.Info("Desired state", "JobManager ingress", *desired.JmIngress)
	} else {
		log.Info("Desired state", "JobManager ingress", "nil")
	}
	if desired.TmStatefulSet != nil {
		log.Info("Desired state", "TaskManager StatefulSet", *desired.TmStatefulSet)
	} else {
		log.Info("Desired state", "TaskManager StatefulSet", "nil")
	}
	if desired.Job != nil {
		log.Info("Desired state", "Job", *desired.Job)
	} else {
		log.Info("Desired state", "Job", "nil")
	}

	log.Info("---------- 4. Take actions ----------")

	var reconciler = ClusterReconciler{
		k8sClient:   k8sClient,
		flinkClient: flinkClient,
		context:     context,
		log:         log,
		observed:    handler.observed,
		desired:     handler.desired,
		recorder:    handler.recorder,
	}

	result, err := reconciler.reconcile()
	if err != nil {
		log.Error(err, "Failed to reconcile")
	}
	if result.RequeueAfter > 0 {
		log.Info("Requeue reconcile request", "after", result.RequeueAfter)
	}

	return result, err
}
