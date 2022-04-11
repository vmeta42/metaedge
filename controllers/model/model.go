/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 9/30/21, 3:25 PM  @author hu.xiaolin3
 */

package model

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

// DesiredClusterState holds desired state of a cluster.
type DesiredClusterState struct {
	JmStatefulSet *appsv1.StatefulSet
	JmService     *corev1.Service
	JmIngress     *extensionsv1beta1.Ingress
	TmStatefulSet *appsv1.StatefulSet
	ConfigMap     *corev1.ConfigMap
	Job           *batchv1.Job
}
