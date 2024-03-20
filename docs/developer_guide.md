# Developer guide

## Prerequisites
- Docker
- opm https://github.com/operator-framework/operator-registry/releases
- operator-sdk https://github.com/operator-framework/operator-sdk/releases
- olm https://github.com/operator-framework/operator-lifecycle-manager/releases
- golang 1.21
- make
- kubectl
- Ubuntu 20.04/Ubuntu 22.04 or RedHat8 OS

## Building  the operator
```
make build
```
Results are stored in build folder.

## Starting the operator process

Start the operator
```bash
make run
```

Start the operator for Kubernetes
```bash
make run_k8s
```

## Build docker image
```bash
make docker-build
```

## Install CRDs
```bash
make install
```

## Deploy the operator in K8S
```bash
make deploy IMG=registry.toolbox.iotg.sclab.intel.com/cpp/operator:latest
```


## OLM development flow
```
make cluster_clean
make build_all_images
make deploy_catalog
make deploy_operator
```

## K8S release process
```
make docker-build OPERATOR_IMAGE=quay.io/openvino/ovms-operator IMAGE_TAG=<version> 
make docker-push OPERATOR_IMAGE=quay.io/openvino/ovms-operator IMAGE_TAG=<version>
```
Make a PR to https://github.com/k8s-operatorhub/community-operators/tree/main/operators/ovms-operator with the [bundle](../bundle_k8s) content.

## Openshift development flow

```
make build_all_images TARGET_PLATFORM=openshift
make deploy_catalog TARGET_PLATFORM=openshift

```
Manually install the operator from the GUI interface using the test catalog

## OpenShift release
Update the OVMS image to the latest tag in RH registry
```
make docker-build TARGET_PLATFORM=openshift
```
Certify new version of the operator image
```
make bundle_build OPERATOR_IMAGE=registry.connect.redhat.com/intel/ovms-operator IMAGE_TAG=0.3.0
```
Make a PR to https://github.com/redhat-openshift-ecosystem/certified-operators with the [bundle](../bundle) content.






