/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 9/30/21, 3:23 PM  @author hu.xiaolin3
 */

// +kubebuilder:docs-gen:collapse=Apache License

package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:docs-gen:collapse=Go imports

var log = logf.Log.WithName("webhook")

// SetupWebhookWithManager adds webhook for FlinkCluster.
func (cluster *FlinkCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(cluster).
		Complete()
}

/*
Kubebuilder markers to generate webhook manifests.
This marker is responsible for generating a mutating webhook manifest.
The meaning of each marker can be found [here](/reference/markers/webhook.md).
*/

// +kubebuilder:webhook:path=/mutate-flinkoperator-k8s-io-v1-flinkcluster,mutating=true,failurePolicy=fail,groups=daas.tenxcloud.com,resources=flinkclusters,verbs=create;update,versions=v1,name=flinkcluster.daas.tenxcloud.com

/*
We use the `webhook.Defaulter` interface to set defaults to our CRD.
A webhook will automatically be served that calls this defaulting.
The `Default` method is expected to mutate the receiver, setting the defaults.
*/

var _ webhook.Defaulter = &FlinkCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the
// type.
func (cluster *FlinkCluster) Default() {
	if cluster == nil {
		return
	}
	log.Info("default", "name", cluster.Name, "original", *cluster)
	_SetDefault(cluster)
	log.Info("default", "name", cluster.Name, "augmented", *cluster)
}

/*
This marker is responsible for generating a validating webhook manifest.
*/

// +kubebuilder:webhook:path=/validate-flinkoperator-k8s-io-v1-flinkcluster,mutating=false,failurePolicy=fail,groups=daas.tenxcloud.com,resources=flinkclusters,verbs=create;update,versions=v1,name=vflinkcluster.daas.tenxcloud.com

var _ webhook.Validator = &FlinkCluster{}
var validator = Validator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered
// for the type.
func (cluster *FlinkCluster) ValidateCreate() error {
	log.Info("Validate create", "name", cluster.Name)
	return validator.ValidateCreate(cluster)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered
// for the type.
func (cluster *FlinkCluster) ValidateUpdate(old runtime.Object) error {
	log.Info("Validate update", "name", cluster.Name)
	var oldCluster = old.(*FlinkCluster)
	return validator.ValidateUpdate(oldCluster, cluster)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered
// for the type.
func (cluster *FlinkCluster) ValidateDelete() error {
	log.Info("validate delete", "name", cluster.Name)

	// TODO
	return nil
}

// +kubebuilder:docs-gen:collapse=Validate object name
