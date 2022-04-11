/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 10/1/21, 10:31 AM  @author hu.xiaolin3
 */

package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	apiv1 "flink-operator/api/v1"
	"flink-operator/controllers/history"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ControlSavepointTriggerID = "SavepointTriggerID"
	ControlJobID              = "jobID"
	ControlRetries            = "retries"
	ControlMaxRetries         = "3"

	SavepointTimeoutSec = 900 // 15 mins

	RevisionNameLabel = "daas.tenxcloud.com/revision-name"

	// TODO: need to be user configurable
	SavepointAgeForJobUpdateSec      = 300
	SavepointRequestRetryIntervalSec = 10
)

type UpdateState string

const (
	UpdateStatePreparing  UpdateState = "Preparing"
	UpdateStateInProgress UpdateState = "InProgress"
	UpdateStateFinished   UpdateState = "Finished"
)

type objectForPatch struct {
	Metadata objectMetaForPatch `json:"metadata"`
}

// objectMetaForPatch define object meta struct for patch operation
type objectMetaForPatch struct {
	Annotations map[string]interface{} `json:"annotations"`
}

func getFlinkAPIBaseURL(cluster *apiv1.FlinkCluster) string {
	clusterDomain := os.Getenv("CLUSTER_DOMAIN")
	if clusterDomain == "" {
		clusterDomain = "cluster.local"
	}

	return fmt.Sprintf(
		"http://%s.%s.svc.%s:%d",
		getJobManagerServiceName(cluster.ObjectMeta.Name),
		cluster.ObjectMeta.Namespace,
		clusterDomain,
		*cluster.Spec.JobManager.Ports.UI)
}

// Gets JobManager ingress name
func getConfigMapName(clusterName string) string {
	return clusterName + "-configmap"
}

// Gets JobManager StatefulSet name
func getJobManagerStatefulSetName(clusterName string) string {
	return clusterName + "-jobmanager"
}

// Gets JobManager service name
func getJobManagerServiceName(clusterName string) string {
	return clusterName + "-jobmanager"
}

// Gets JobManager ingress name
func getJobManagerIngressName(clusterName string) string {
	return clusterName + "-jobmanager"
}

// Gets TaskManager name
func getTaskManagerStatefulSetName(clusterName string) string {
	return clusterName + "-taskmanager"
}

// Gets Job name
func getJobName(clusterName string) string {
	return clusterName + "-job-submitter"
}

// TimeConverter converts between time.Time and string.
type TimeConverter struct{}

// FromString converts string to time.Time.
func (tc *TimeConverter) FromString(timeStr string) time.Time {
	timestamp, err := time.Parse(
		time.RFC3339, timeStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse time string: %s", timeStr))
	}
	return timestamp
}

// ToString converts time.Time to string.
func (tc *TimeConverter) ToString(timestamp time.Time) string {
	return timestamp.Format(time.RFC3339)
}

// setTimestamp sets the current timestamp to the target.
func setTimestamp(target *string) {
	var tc = &TimeConverter{}
	var now = time.Now()
	*target = tc.ToString(now)
}

// Checks whether it is possible to take savepoint.
func canTakeSavepoint(cluster apiv1.FlinkCluster) bool {
	var jobSpec = cluster.Spec.Job
	var savepointStatus = cluster.Status.Savepoint
	var jobStatus = cluster.Status.Components.Job
	return jobSpec != nil && jobSpec.SavepointsDir != nil &&
		!isJobStopped(jobStatus) &&
		(savepointStatus == nil || savepointStatus.State != apiv1.SavepointStateInProgress)
}

// shouldRestartJob returns true if the controller should restart failed or lost job.
func shouldRestartJob(
	restartPolicy *apiv1.JobRestartPolicy,
	jobStatus *apiv1.JobStatus) bool {
	return restartPolicy != nil &&
		*restartPolicy == apiv1.JobRestartPolicyFromSavepointOnFailure &&
		jobStatus != nil &&
		(jobStatus.State == apiv1.JobStateFailed || jobStatus.State == apiv1.JobStateLost) &&
		len(jobStatus.SavepointLocation) > 0
}

func shouldUpdateJob(observed ObservedClusterState) bool {
	var jobStatus = observed.cluster.Status.Components.Job
	var readyToUpdate = jobStatus == nil || isJobStopped(jobStatus) || isSavepointUpToDate(observed.observeTime, *jobStatus)
	return isUpdateTriggered(observed.cluster.Status) && readyToUpdate
}

func getFromSavepoint(jobSpec batchv1.JobSpec) string {
	var jobArgs = jobSpec.Template.Spec.Containers[0].Args
	for i, arg := range jobArgs {
		if arg == "--fromSavepoint" && i < len(jobArgs)-1 {
			return jobArgs[i+1]
		}
	}
	return ""
}

// newRevision generates FlinkClusterSpec patch and makes new child ControllerRevision resource with it.
func newRevision(cluster *apiv1.FlinkCluster, revision int64, collisionCount *int32) (*appsv1.ControllerRevision, error) {
	var patch []byte
	var err error

	// Ignore fields not related to rendering job resource.
	if cluster.Spec.Job != nil {
		clusterClone := cluster.DeepCopy()
		clusterClone.Spec.Job.CleanupPolicy = nil
		clusterClone.Spec.Job.RestartPolicy = nil
		clusterClone.Spec.Job.CancelRequested = nil
		clusterClone.Spec.Job.SavepointGeneration = 0
		patch, err = getPatch(clusterClone)
	} else {
		patch, err = getPatch(cluster)
	}
	if err != nil {
		return nil, err
	}
	cr, err := history.NewControllerRevision(cluster,
		controllerKind,
		cluster.ObjectMeta.Labels,
		runtime.RawExtension{Raw: patch},
		revision,
		collisionCount)
	if err != nil {
		return nil, err
	}
	if cr.ObjectMeta.Annotations == nil {
		cr.ObjectMeta.Annotations = make(map[string]string)
	}
	for key, value := range cluster.Annotations {
		cr.ObjectMeta.Annotations[key] = value
	}
	cr.SetNamespace(cluster.GetNamespace())
	cr.GetLabels()[history.ControllerRevisionManagedByLabel] = cluster.GetName()
	return cr, nil
}

func getPatch(cluster *apiv1.FlinkCluster) ([]byte, error) {
	str := &bytes.Buffer{}
	err := unstructured.UnstructuredJSONScheme.Encode(cluster, str)

	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	_ = json.Unmarshal(str.Bytes(), &raw)
	objCopy := make(map[string]interface{})
	spec := raw["spec"].(map[string]interface{})
	objCopy["spec"] = spec
	spec["$patch"] = "replace"
	patch, err := json.Marshal(objCopy)
	return patch, err
}

func getNextRevisionNumber(revisions []*appsv1.ControllerRevision) int64 {
	count := len(revisions)
	if count <= 0 {
		return 1
	}
	return revisions[count-1].Revision + 1
}

func getCurrentRevisionName(status apiv1.FlinkClusterStatus) string {
	return status.CurrentRevision[:strings.LastIndex(status.CurrentRevision, "-")]
}

func getNextRevisionName(status apiv1.FlinkClusterStatus) string {
	return status.NextRevision[:strings.LastIndex(status.NextRevision, "-")]
}

// Compose revision in FlinkClusterStatus with name and number of ControllerRevision
func getRevisionWithNameNumber(cr *appsv1.ControllerRevision) string {
	return fmt.Sprintf("%v-%v", cr.Name, cr.Revision)
}

func getRetryCount(data map[string]string) (string, error) {
	var err error
	var retries, ok = data["retries"]
	if ok {
		retryCount, err := strconv.Atoi(retries)
		if err == nil {
			retryCount++
			retries = strconv.Itoa(retryCount)
		}
	} else {
		retries = "1"
	}
	return retries, err
}

func getNewUserControlStatus(controlName string) *apiv1.FlinkClusterControlStatus {
	var controlStatus = new(apiv1.FlinkClusterControlStatus)
	controlStatus.Name = controlName
	controlStatus.State = apiv1.ControlStateProgressing
	setTimestamp(&controlStatus.UpdateTime)
	return controlStatus
}

func getTriggeredSavepointStatus(jobID string, triggerID string, triggerReason string, message string, triggerSuccess bool) apiv1.SavepointStatus {
	var savepointStatus = apiv1.SavepointStatus{}
	var now string
	setTimestamp(&now)
	savepointStatus.JobID = jobID
	savepointStatus.TriggerID = triggerID
	savepointStatus.TriggerReason = triggerReason
	savepointStatus.TriggerTime = now
	savepointStatus.RequestTime = now
	savepointStatus.Message = message
	if triggerSuccess {
		savepointStatus.State = apiv1.SavepointStateInProgress
	} else {
		savepointStatus.State = apiv1.SavepointStateTriggerFailed
	}
	return savepointStatus
}

func getRequestedSavepointStatus(triggerReason string) *apiv1.SavepointStatus {
	var now string
	setTimestamp(&now)
	return &apiv1.SavepointStatus{
		State:         apiv1.SavepointStateNotTriggered,
		TriggerReason: triggerReason,
		RequestTime:   now,
	}
}

func savepointTimeout(s *apiv1.SavepointStatus) bool {
	if s.TriggerTime == "" {
		return false
	}
	tc := &TimeConverter{}
	triggerTime := tc.FromString(s.TriggerTime)
	validTime := triggerTime.Add(time.Duration(int64(SavepointTimeoutSec) * int64(time.Second)))
	return time.Now().After(validTime)
}

func getControlEvent(status apiv1.FlinkClusterControlStatus) (eventType string, eventReason string, eventMessage string) {
	var msg = status.Message
	if len(msg) > 100 {
		msg = msg[:100] + "..."
	}
	switch status.State {
	case apiv1.ControlStateProgressing:
		eventType = corev1.EventTypeNormal
		eventReason = "ControlRequested"
		eventMessage = fmt.Sprintf("Requested new user control %v", status.Name)
	case apiv1.ControlStateSucceeded:
		eventType = corev1.EventTypeNormal
		eventReason = "ControlSucceeded"
		eventMessage = fmt.Sprintf("Succesfully completed user control %v", status.Name)
	case apiv1.ControlStateFailed:
		eventType = corev1.EventTypeWarning
		eventReason = "ControlFailed"
		if status.Message != "" {
			eventMessage = fmt.Sprintf("User control %v failed: %v", status.Name, msg)
		} else {
			eventMessage = fmt.Sprintf("User control %v failed", status.Name)
		}
	}
	return
}

func getSavepointEvent(status apiv1.SavepointStatus) (eventType string, eventReason string, eventMessage string) {
	var msg = status.Message
	if len(msg) > 100 {
		msg = msg[:100] + "..."
	}
	var triggerReason = status.TriggerReason
	if triggerReason == apiv1.SavepointTriggerReasonJobCancel || triggerReason == apiv1.SavepointTriggerReasonUpdate {
		triggerReason = "for " + triggerReason
	}
	switch status.State {
	case apiv1.SavepointStateTriggerFailed:
		eventType = corev1.EventTypeWarning
		eventReason = "SavepointFailed"
		eventMessage = fmt.Sprintf("Failed to trigger savepoint %v: %v", triggerReason, msg)
	case apiv1.SavepointStateInProgress:
		eventType = corev1.EventTypeNormal
		eventReason = "SavepointTriggered"
		eventMessage = fmt.Sprintf("Triggered savepoint %v: triggerID %v.", triggerReason, status.TriggerID)
	case apiv1.SavepointStateSucceeded:
		eventType = corev1.EventTypeNormal
		eventReason = "SavepointCreated"
		eventMessage = "Successfully savepoint created"
	case apiv1.SavepointStateFailed:
		eventType = corev1.EventTypeWarning
		eventReason = "SavepointFailed"
		eventMessage = fmt.Sprintf("Savepoint creation failed: %v", msg)
	}
	return
}

func isJobActive(status *apiv1.JobStatus) bool {
	return status != nil &&
		(status.State == apiv1.JobStateRunning || status.State == apiv1.JobStatePending)
}

func isJobStopped(status *apiv1.JobStatus) bool {
	return status != nil &&
		(status.State == apiv1.JobStateSucceeded ||
			status.State == apiv1.JobStateFailed ||
			status.State == apiv1.JobStateCancelled ||
			status.State == apiv1.JobStateSuspended ||
			status.State == apiv1.JobStateLost)
}

func isJobCancelRequested(cluster apiv1.FlinkCluster) bool {
	var userControl = cluster.Annotations[apiv1.ControlAnnotation]
	var cancelRequested = cluster.Spec.Job.CancelRequested
	return userControl == apiv1.ControlNameJobCancel ||
		(cancelRequested != nil && *cancelRequested)
}

func isJobTerminated(restartPolicy *apiv1.JobRestartPolicy, jobStatus *apiv1.JobStatus) bool {
	return isJobStopped(jobStatus) && !shouldRestartJob(restartPolicy, jobStatus)
}

func isUpdateTriggered(status apiv1.FlinkClusterStatus) bool {
	return status.CurrentRevision != status.NextRevision
}

func isUserControlFinished(controlStatus *apiv1.FlinkClusterControlStatus) bool {
	return controlStatus.State == apiv1.ControlStateSucceeded ||
		controlStatus.State == apiv1.ControlStateFailed
}

// Check if the savepoint has been created recently.
func isSavepointUpToDate(now time.Time, jobStatus apiv1.JobStatus) bool {
	if jobStatus.SavepointLocation != "" && jobStatus.LastSavepointTime != "" {
		if !hasTimeElapsed(jobStatus.LastSavepointTime, now, SavepointAgeForJobUpdateSec) {
			return true
		}
	}
	return false
}

// Check time has passed
func hasTimeElapsed(timeToCheckStr string, now time.Time, intervalSec int) bool {
	tc := &TimeConverter{}
	timeToCheck := tc.FromString(timeToCheckStr)
	intervalPassedTime := timeToCheck.Add(time.Duration(int64(intervalSec) * int64(time.Second)))
	return now.After(intervalPassedTime)
}

// isComponentUpdated checks whether the component updated.
// If the component is observed as well as the next revision name in status.nextRevision and component's label `daas.tenxcloud.com/hash` are equal, then it is updated already.
// If the component is not observed and it is required, then it is not updated yet.
// If the component is not observed and it is optional, but it is specified in the spec, then it is not updated yet.
func isComponentUpdated(component runtime.Object, cluster apiv1.FlinkCluster) bool {
	if !isUpdateTriggered(cluster.Status) {
		return true
	}
	switch o := component.(type) {
	case *appsv1.Deployment:
		if o == nil {
			return false
		}
	case *appsv1.StatefulSet:
		if o == nil {
			return false
		}
	case *corev1.ConfigMap:
		if o == nil {
			return false
		}
	case *corev1.Service:
		if o == nil {
			return false
		}
	case *batchv1.Job:
		if o == nil {
			return cluster.Spec.Job == nil
		}
	case *extensionsv1beta1.Ingress:
		if o == nil {
			return cluster.Spec.JobManager.Ingress == nil
		}
	}

	var labels, err = meta.NewAccessor().Labels(component)
	var nextRevisionName = getNextRevisionName(cluster.Status)
	if err != nil {
		return false
	}
	return labels[RevisionNameLabel] == nextRevisionName
}

func areComponentsUpdated(components []runtime.Object, cluster apiv1.FlinkCluster) bool {
	for _, c := range components {
		if !isComponentUpdated(c, cluster) {
			return false
		}
	}
	return true
}

// isClusterUpdateToDate checks whether all cluster components are replaced to next revision.
func isClusterUpdateToDate(observed ObservedClusterState) bool {
	if !isUpdateTriggered(observed.cluster.Status) {
		return true
	}
	components := []runtime.Object{
		observed.configMap,
		observed.jmStatefulSet,
		observed.tmStatefulSet,
		observed.jmService,
	}
	return areComponentsUpdated(components, *observed.cluster)
}

// isFlinkAPIReady checks whether cluster is ready to submit job.
func isFlinkAPIReady(observed ObservedClusterState) bool {
	// If the observed Flink job status list is not nil (e.g., emtpy list),
	// it means Flink REST API server is up and running. It is the source of
	// truth of whether we can submit a job.
	return observed.flinkJobStatus.flinkJobList != nil
}

func getUpdateState(observed ObservedClusterState) UpdateState {
	var recordedJobStatus = observed.cluster.Status.Components.Job
	if !isUpdateTriggered(observed.cluster.Status) {
		return ""
	}
	if isJobActive(recordedJobStatus) {
		return UpdateStatePreparing
	}
	if isClusterUpdateToDate(observed) {
		return UpdateStateFinished
	}
	return UpdateStateInProgress
}

func getNonLiveHistory(revisions []*appsv1.ControllerRevision, historyLimit int) []*appsv1.ControllerRevision {

	history := append([]*appsv1.ControllerRevision{}, revisions...)
	nonLiveHistory := make([]*appsv1.ControllerRevision, 0)

	historyLen := len(history)
	if historyLen <= historyLimit {
		return nonLiveHistory
	}

	nonLiveHistory = append(nonLiveHistory, history[:(historyLen-historyLimit)]...)
	return nonLiveHistory
}

func getFlinkJobDeploymentState(flinkJobState string) string {
	switch flinkJobState {
	case "INITIALIZING", "CREATED", "RUNNING", "FAILING", "CANCELLING", "RESTARTING", "RECONCILING":
		return apiv1.JobStateRunning
	case "FINISHED":
		return apiv1.JobStateSucceeded
	case "CANCELED":
		return apiv1.JobStateCancelled
	case "FAILED":
		return apiv1.JobStateFailed
	case "SUSPENDED":
		return apiv1.JobStateSuspended
	default:
		return ""
	}
}

// getFlinkJobSubmitLog extract submit result from the pod termination log.
func getFlinkJobSubmitLog(observedPod *corev1.Pod) (*FlinkJobSubmitLog, error) {
	if observedPod == nil {
		return nil, fmt.Errorf("no job pod found, even though submission completed")
	}
	var containerStatuses = observedPod.Status.ContainerStatuses
	if len(containerStatuses) == 0 ||
		containerStatuses[0].State.Terminated == nil ||
		containerStatuses[0].State.Terminated.Message == "" {
		return nil, fmt.Errorf("job pod found, but no termination log found even though submission completed")
	}

	// The job submission script writes the submission log to the pod termination log at the end of execution.
	// If the job submission is successful, the extracted job ID is also included.
	// The job submit script writes the submission result in YAML format,
	// so parse it here to get the ID - if available - and log.
	// Note: https://kubernetes.io/docs/tasks/debug-application-cluster/determine-reason-pod-failure/
	var rawJobSubmitResult = containerStatuses[0].State.Terminated.Message
	var result = new(FlinkJobSubmitLog)
	var err = yaml.Unmarshal([]byte(rawJobSubmitResult), result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
