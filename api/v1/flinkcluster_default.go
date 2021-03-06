/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 9/30/21, 3:21 PM  @author hu.xiaolin3
 */

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Sets default values for unspecified FlinkCluster properties.
func _SetDefault(cluster *FlinkCluster) {
	if cluster.Spec.RecreateOnUpdate == nil {
		cluster.Spec.RecreateOnUpdate = new(bool)
		*cluster.Spec.RecreateOnUpdate = true
	}
	_SetImageDefault(&cluster.Spec.Image)
	_SetJobManagerDefault(&cluster.Spec.JobManager)
	_SetTaskManagerDefault(&cluster.Spec.TaskManager)
	_SetJobDefault(cluster.Spec.Job)
	_SetHadoopConfigDefault(cluster.Spec.HadoopConfig)
}

func _SetImageDefault(imageSpec *ImageSpec) {
	if len(imageSpec.PullPolicy) == 0 {
		imageSpec.PullPolicy = corev1.PullAlways
	}
}

func _SetJobManagerDefault(jmSpec *JobManagerSpec) {
	if jmSpec.Replicas == nil {
		jmSpec.Replicas = new(int32)
		*jmSpec.Replicas = 1
	}
	if len(jmSpec.AccessScope) == 0 {
		jmSpec.AccessScope = AccessScopeCluster
	}
	if jmSpec.Ingress != nil {
		if jmSpec.Ingress.UseTLS == nil {
			jmSpec.Ingress.UseTLS = new(bool)
			*jmSpec.Ingress.UseTLS = false
		}
	}
	if jmSpec.Ports.RPC == nil {
		jmSpec.Ports.RPC = new(int32)
		*jmSpec.Ports.RPC = 6123
	}
	if jmSpec.Ports.Blob == nil {
		jmSpec.Ports.Blob = new(int32)
		*jmSpec.Ports.Blob = 6124
	}
	if jmSpec.Ports.Query == nil {
		jmSpec.Ports.Query = new(int32)
		*jmSpec.Ports.Query = 6125
	}
	if jmSpec.Ports.UI == nil {
		jmSpec.Ports.UI = new(int32)
		*jmSpec.Ports.UI = 8081
	}
	if jmSpec.MemoryOffHeapMin.Format == "" {
		jmSpec.MemoryOffHeapMin = *resource.NewScaledQuantity(600, 6) // 600MB
	}
	if jmSpec.MemoryOffHeapRatio == nil {
		jmSpec.MemoryOffHeapRatio = new(int32)
		*jmSpec.MemoryOffHeapRatio = 25
	}
}

func _SetTaskManagerDefault(tmSpec *TaskManagerSpec) {
	if tmSpec.Ports.Data == nil {
		tmSpec.Ports.Data = new(int32)
		*tmSpec.Ports.Data = 6121
	}
	if tmSpec.Ports.RPC == nil {
		tmSpec.Ports.RPC = new(int32)
		*tmSpec.Ports.RPC = 6122
	}
	if tmSpec.Ports.Query == nil {
		tmSpec.Ports.Query = new(int32)
		*tmSpec.Ports.Query = 6125
	}
	if tmSpec.MemoryOffHeapMin.Format == "" {
		tmSpec.MemoryOffHeapMin = *resource.NewScaledQuantity(600, 6) // 600MB
	}
	if tmSpec.MemoryOffHeapRatio == nil {
		tmSpec.MemoryOffHeapRatio = new(int32)
		*tmSpec.MemoryOffHeapRatio = 25
	}
}

func _SetJobDefault(jobSpec *JobSpec) {
	if jobSpec == nil {
		return
	}
	if jobSpec.AllowNonRestoredState == nil {
		jobSpec.AllowNonRestoredState = new(bool)
		*jobSpec.AllowNonRestoredState = false
	}
	if jobSpec.Parallelism == nil {
		jobSpec.Parallelism = new(int32)
		*jobSpec.Parallelism = 1
	}
	if jobSpec.NoLoggingToStdout == nil {
		jobSpec.NoLoggingToStdout = new(bool)
		*jobSpec.NoLoggingToStdout = false
	}
	if jobSpec.RestartPolicy == nil {
		jobSpec.RestartPolicy = new(JobRestartPolicy)
		*jobSpec.RestartPolicy = JobRestartPolicyNever
	}
	if jobSpec.CleanupPolicy == nil {
		jobSpec.CleanupPolicy = &CleanupPolicy{
			AfterJobSucceeds:  CleanupActionDeleteCluster,
			AfterJobFails:     CleanupActionKeepCluster,
			AfterJobCancelled: CleanupActionDeleteCluster,
		}
	}
}

func _SetHadoopConfigDefault(hadoopConfig *HadoopConfig) {
	if hadoopConfig == nil {
		return
	}
	if len(hadoopConfig.MountPath) == 0 {
		hadoopConfig.MountPath = "/etc/hadoop/conf"
	}
}
