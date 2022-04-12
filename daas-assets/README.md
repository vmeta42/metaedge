# How to use flink cluster operator
1、Create a namespace to do the test, you can also use an existing namespace and skip this step.
```
kubectl create ns flink-test
```
2、Create a service account to deploy flink cluster operator and grant the required permissions using RBAC. Modify the namespace、name as needed。
```
kubectl create sa flink-cluster-operator -n flink-test
kubectl apply -f flinkoperator_role.yaml
kubectl apply -f flinkoperator_rolebinding.yaml
```
3、Create CRDs of flink cluster
```
kubectl apply -f flinkcluster_crd.yaml
```
4、Deply the flink cluster operator to this namespace，modify the name as needed.
```
kubectl apply -f flinkcluster_operator.yaml
```
5、Prepare the configuration files for flink cluster to deploy.

1）Edit flink configuration file flinkcluster_example.yaml, you can also update it as needed.
```
kubectl apply -f flinkcluster_config.yaml
```
3）Create a service accout to deploy flink cluster and grant to permissions using RBAC.
* sa flink-sa is newly added for specially usage in flink cluster, if it not exists you should create it manually
```
kubectl create sa flink-sa -n flink-test
kubectl apply -f flinkcluster_rbac.yaml
```
6、Deploy the flink cluster, we support flink 5.7.22 and 8.0.20，8.0.20 will be used by default.
* You can change the image version in the flinkcluster_example.yaml file.
* Add the storageClassName to the spec if you need a specified one.
```
kubectl apply -f flinkcluster_example.yaml
```
