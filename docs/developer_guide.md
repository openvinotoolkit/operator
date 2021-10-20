# Developer guide

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


##OLM development flow
```
make olm_clean
make olm_install
make docker-build 
make bundle_k8s_build
make bundle_image_push
make k8s_catalog_build
make k8s_catalog_push
make bundle_deploy_k8s
```

##K8S release process
```
make docker-build IMG=quay.io/openvino/ovms-operator:<version>
make docker-push IMG=quay.io/openvino/ovms-operator:<version>
```
Make a PR to https://github.com/k8s-operatorhub/community-operators/tree/main/operators/ovms-operator withthe [bundle](../bundle) content.

##Openshift flow

```
make docker-build 
make bundle_openshift_build
make bundle_image_push
make openshift_catalog_build
make openshift_catalog_push
make catalog_deploy_openshift
```

##OpenShift release
Update the OVMS image to the laters tag in RH registry
```
make docker-build
```
Certify new version of the operator image
```
make bundle_openshift_build IMG=registry.connect.redhat.com/intel/ovms-operator TAG=0.3.0
```
Certify the bundle image






