# Operator instalation

*Note:* Operator, starting from version 1.0 includes non compatible changes in the CRD records of `ModelServer` and `Notebook`.
It is recommended to remove all those custom records before upgrading the operator from v0.2 to v1.0.

*Note:* This repository includes operator v1.0 which is to be published soon. The source code for older versions is stored in the [model server repo](https://github.com/openvinotoolkit/model_server/tree/main/extras)

## Openshift

In the OpenShift [web console](https://docs.openshift.com/container-platform/4.10/web_console/web-console.html) navigate to OperatorHub menu. Search for "OpenVINO™ Toolkit Operator". Then, click the `Install` button.

![installation](install.png)

## Kubernetes

Operator can be installed in Kubernetes cluster from the [OperatorHub](https://operatorhub.io/operator).

Find the `OpenVINO Model Server Operator` and click 'Install' button.

***

## Building and installation from sources

Check the [developer guide](developer_guide.md) if you would like to build the operator on your own.

***

Check also:
- [Deploying model servers via the operator](./modelserver.md)
- [Integrating OpenVINO notbook image wit Openshift RedHat Data Science operator](./notebook_in_rhods.md)
