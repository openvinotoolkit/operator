# Developer guide

## Prerequisites
- docker
- opm https://github.com/operator-framework/operator-registry/releases
- operator-sdk https://github.com/operator-framework/operator-sdk/releases
- golang 1.17
- make
- kubectl
- ubuntu20 or redhat8 OS

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

## Build docker image
```bash
make docker-build IMG=registry.toolbox.iotg.sclab.intel.com/cpp/operator:latest
```

## Install CRD
```bash
make install
```

## Deploy the operator in K8S
```bash
make deploy IMG=registry.toolbox.iotg.sclab.intel.com/cpp/operator:latest
```


## OLM development flow
```
make olm_reset
make k8s_all TAG=<branch>-k8s
```

## K8S release process
```
make docker-build IMG=quay.io/openvino/ovms-operator:<version>
make docker-push IMG=quay.io/openvino/ovms-operator:<version>
```
Make a PR to https://github.com/k8s-operatorhub/community-operators/tree/main/operators/ovms-operator with the [bundle](../bundle) content.

## Openshift flow

```
make openshift-all TAG=<branch>-openshift
```
Manually install the operator from the GUI interface using the test catalog

## OpenShift release
Update the OVMS image to the latest tag in RH registry
```
make docker-build
```
Certify new version of the operator image
```
make bundle_openshift_build IMG=registry.connect.redhat.com/intel/ovms-operator TAG=0.3.0
```
Certify the bundle image






