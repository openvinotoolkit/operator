# Openshift and Kubernetes operator

The Operator installs and manages [OpenVINO model servers](https://github.com/openvinotoolkit/model_server) in an OpenShift cluster and upstream Kubernetes. It enables inference execution at scale and exposes AI models via gRPC and REST API interfaces.
The operator is using a [model server helm chart](helm-charts/ovms/) which can be also used directly.

The Operator also integrates with the JupyterHub Spawner in Red Hat OpenShift Data Science and Open Data Hub. See detailed instructions below.

![logos](logos.png)

[Operator installation](docs/operator_installation.md)

[Installing and managing OpenVINO model server via the operator](docs/modelserver.md)

[Model server parameters](docs/modelserver_params.md)

[Autoscalability](docs/autoscaling.md)

[Performance tuning](docs/recommendations.md)

[OpenVINO notebook integration with RHODS](docs/notebook_in_rhods.md)

[Notebook parameters](docs/notebook_params.md)

[Helm chart](helm-charts/ovms/README.md)


## Contact
If you have a question, a feature request, or a bug report, feel free to submit a Github issue.

* Other names and brands may be claimed as the property of others.







