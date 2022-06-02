![image](docs/image/MetaEdge_logo.png)

![image](https://img.shields.io/badge/license-MIT-green)   ![image](https://img.shields.io/badge/Kubernetes-1.15%2B-yellowgreen)  ![image](https://img.shields.io/badge/Flink-1.7%2B-orange)   ![image](https://img.shields.io/badge/kubectl-1.15%2B-yellow)

# Apache Flink On KubeEdge

**This is an flink k8s operator which support kubeedge.**   
Kubernetes Operator for Apache Flink is a control plane for running [Apache Flink](https://flink.apache.org/) on
[Kubeedge](https://kubeedge.io/).

## Project Status

*Beta*

The operator is under active development, backward compatibility of the APIs is not guaranteed for beta releases.

## Prerequisites

* Version >= 1.15 of Kubernetes
* Version >= 1.15 of kubectl (with kustomize)
* Version >= 1.7 of Apache Flink

## Overview

The Kubernetes Operator for Apache Flink extends the vocabulary (e.g., Pod, Service, etc) of the Kubernetes language
with custom resource definition [FlinkCluster](docs/crd.md) and runs a
[controller](controllers/flinkcluster_controller.go) Pod to keep watching the custom resources.
Once a FlinkCluster custom resource is created and detected by the controller, the controller creates the underlying
Kubernetes resources (e.g., JobManager Pod) based on the spec of the custom resource. With the operator installed in a
cluster, users can then talk to the cluster through the Kubernetes API and Flink custom resources to manage their Flink
clusters and jobs.

## Features

* Support for both Flink [job cluster](config/samples/flinkoperator_v1_flinkjobcluster.yaml) and
  [session cluster](config/samples/flinkoperator_v1_flinksessioncluster.yaml) depending on whether a job spec is
  provided
* Custom Flink images
* Flink and Hadoop configs and container environment variables
* Init containers and sidecar containers
* Remote job jar
* Configurable namespace to run the operator in
* Configurable namespace to watch custom resources in
* Configurable access scope for JobManager service
* Taking savepoints periodically
* Taking savepoints on demand
* Restarting failed job from the latest savepoint automatically
* Cancelling job with savepoint
* Cleanup policy on job success and failure
* Updating cluster or job
* Batch scheduling for JobManager and TaskManager Pods
* GCP integration (service account, GCS connector, networking)
* Support for Beam Python jobs

## Installation

The operator is still under active development, there is no Helm chart available yet. You can follow either
* [User Guide](docs/user_guide.md) to deploy a released operator image on `gcr.io/flink-operator` to your Kubernetes
  cluster or
* [Developer Guide](docs/developer_guide.md) to build an operator image first then deploy it to the cluster.

## Documentation

### Quickstart guides

* [User Guide](docs/user_guide.md)
* [Developer Guide](docs/developer_guide.md)
* [Flink-On-Kubeedge Guide](docs/flink-on-kubeedge.md)

### API

* [Custom Resource Definition (v1)](docs/crd.md)

### How to

* [Manage savepoints](docs/savepoints_guide.md)
* [Use remote job jars](config/samples/flinkoperator_v1_remotejobjar.yaml)
* [Run Apache Beam Python jobs](docs/beam_guide.md)
* [Use GCS connector](images/flink/README.md)
* [Test with Apache Kafka](docs/kafka_test_guide.md)
* [Create Flink job clusters with Helm Chart](docs/flink_job_cluster_guide.md)

### Tech talks

* CNCF Webinar: Apache Flink on Kubernetes Operator

## Contributing

Please check [CONTRIBUTING.md](CONTRIBUTING.md) and the [Developer Guide](docs/developer_guide.md) out.

## Build

```bash
 make build
 docker build -t 172.22.50.227/system_containers/flink-operator:0.1.6 .
 
 docker tag 172.22.50.223/system_containers/daas-flink-registry:0.1.6 172.22.50.227/system_containers/daas-flink-registry:0.1.6
```

## Build olm images

```bash
 docker build . -t dev-registry.tenxcloud.com/system_containers/flink-operator-bundle:0.1.6
 docker push dev-registry.tenxcloud.com/system_containers/flink-operator-bundle:0.1.6

 opm --skip-tls index add --bundles dev-registry.tenxcloud.com/system_containers/flink-operator-bundle:0.1.6 --tag dev-registry.tenxcloud.com/system_containers/daas-flink-registry:0.1.6 -c="docker"
```

## CatalogSource

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  labels:
    source-ns: system-tce
  name: daas-registry-flink
  namespace: operator-hub
spec:
  displayName: built-in flink registry
  icon:
    base64data: ""
    mediatype: ""
  image: dev-registry.tenxcloud.com/system_containers/daas-flink-registry:0.1.6
  publisher: system admin
  sourceType: grpc
```
