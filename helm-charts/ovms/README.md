# OpenVINO Model Server deployment via a helm chart
The helm chart provided here can be used to install OpenVINO Model Server in a Kubernetes cluster. 
It has the same parameters as the operator so it could be used directly as an alternative deployment method.
The helm chart is managing the Model Server instance which represents a kubernetes deployment and a kubernetes service with exposed REST and gRPC inference endpoints. This guide assumes you already have a functional Kubernetes cluster and helm installed (see below for instructions on installing helm).
The steps below describe how to setup a model repository, use helm to launch the inference server and then send inference requests to the running server.

## Installing Helm
Please refer to [Helm installation guide](https://helm.sh/docs/intro/install).

## Usage guide

### Deploy OpenVINO Model Server with a Single Model
Deploy Model Server using _helm_. Please include the required model name and model path. You can also adjust other parameters defined in [values.yaml](values.yaml).
```shell script
helm install ovms-app ovms --set models_settings.model_name=<model_name>,models_settings.model_path=gs://<bucket>/<model>
```
Use _kubectl_ to see the status and wait until the Model Server pod is running:
```shell script
kubectl get pods
NAME                    READY   STATUS    RESTARTS   AGE
ovms-app-5fd8d6b845-w87jl   1/1     Running   0          27s
```
By default, Model Server is deployed with 1 instance. If you would like to scale up additional replicas, override the value in values.yaml file or by passing _--set_ flag to _helm install_:
```shell script
helm install ovms-app ovms --set models_settings.model_name=<model_name>,models_settings.model_path=gs://<bucket>/<model>,deployment_parameters.replicas=3
```

### Deploy OpenVINO Model Server with Multiple Models Defined in a Configuration File
To serve multiple models, you can run Model Server with a configuration file as described in [model server documentation](https://docs.openvino.ai/latest/ovms_docs_serving_model.html#serving-multiple-models).
Follow the above documentation to create a configuration file named _config.json_ and fill it with proper information.
To deploy with config file stored in the Kubernetes ConfigMap:
* create a ConfigMap resource from this file with a chosen name (here _ovms-config_):
```shell script     
kubectl create configmap ovms-config --from-file config.json
```
* deploy Model Server with parameters `models_settings.single_model_mode` and `models_settings.config_configmap_name` (without `model_name` and `model_path`):
```shell script
helm install ovms-app ovms --set models_settings.config_configmap_name=ovms-config,models_settings.single_model_mode=false
```

### GCS
Bucket permissions can be set with the _GOOGLE_APPLICATION_CREDENTIALS_ environment variable. Please follow the steps below:
* Generate Google service account JSON file with permissions: _Storage Legacy Bucket Reader_, _Storage Legacy Object Reader_, _Storage Object Viewer_. Name a file for example: _gcp-creds.json_ 
(you can follow these instructions to [create a Service Account](https://cloud.google.com/docs/authentication/getting-started#creating_a_service_account) and download JSON)
* Create a Kubernetes secret from this JSON file:
```shell script
kubectl create secret generic gcpcreds --from-file gcp-creds.json
```
* When deploying Model Server, provide the model path to GCS bucket and name for the secret created above. Make sure to provide `gcp_creds_secret_name` when deploying:
```shell script
helm install ovms-app ovms --set models_settings.model_name=<model_name>,models_settings.model_path=gs://<bucket>y/<model>,models_repository.gcp_creds_secret_name=gcpcreds
```
### S3
For S3 you must provide an AWS Access Key ID, the content of that key (AWS Secret Access Key) and the AWS region when deploying: `aws_access_key_id`, `aws_secret_access_key` and `aws_region` (see below).
```shell script
helm install ovms-app ovms --set models_settings.model_name=<model_name>,models_settings.model_path=s3://<bucket>/<model>,models_repository.aws_access_key_id=<...>,models_repository.aws_secret_access_key=<...>,models_repository.aws_region=<...>
```
In case you would like to use custom S3 service with compatible API (e.g. MinIO), you need to also provide endpoint 
to that service. Please provide it by supplying `s3_compat_api_endpoint`:
```shell script
helm install ovms-app ovms --set models_settings.model_name=icnet-camvid-ava-0001,models_settings.model_path=s3://<bucket>/<model>,models_repository.aws_access_key_id=<...>,models_repository.aws_secret_access_key=<...>,models_repository.s3_compat_api_endpoint=<...>
```
### Azure Storage
Use OVMS with models stored on azure blob storage by providing `azure_storage_connection_string` parameter. Model path should follow `az` scheme like below:
```shell script
helm install ovms-app ovms --set models_settings.model_name=resnet,models_settings.model_path=az://<container>/<model_path>,models_repository.azure_storage_connection_string="DefaultEndpointsProtocol=https;AccountName=azure_account_name;AccountKey=smp/hashkey==;EndpointSuffix=core.windows.net"
```
 
### Local Node Storage
Beside the cloud storage, models could be stored locally on the kubernetes nodes filesystem.
Use the parameter `models_repository.models_host_path` with the local path on the nodes. It will be mounted in the OVMS container as `/models` folder.
While the models folder is mounted in the OVMS container, the parameter `models_settings.model_path` should refer to the path starting with /models/... and point to the folder with the model versions.
Note that the OVMS container starts, by default, with the security context of account `ovms` with pid 5000 and group 5000. 
If the mounted models have restricted access permissions, change the security context of the OVMS service or adjust permissions to the models. OVMS requires read permissions on the model files and 
list permission on the model version folders.
### Persistent Volume
It is possible to deploy OVMS using Kubernetes [persistent volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).
That opens a possibility of storing the models for OVMS on all Kubernetes [supported filesystems](https://kubernetes.io/docs/concepts/storage/storage-classes/).
In the helm set the parameter `models_repository.models_volume_claim` with the name of the `PersistentVolumeClaim` record with the models. While set, it will be mounted as `/models` folder inside the OVMS container.
Note that parameter `models_repository.models_volume_claim` is mutually exclusive with `models_repository.models_host_path`. Only one of them should be set.
### Assigning Resource Specs
By default, there are no restrictions, but can restrict assigned cluster resources to the OVMS container by setting the parameters:
- `deployment_parameters.resources.limits.cpu` - maximum cpu allocation
- `deployment_parameters.resources.limits.memory` - maximum memory allocation
- `deployment_parameters.resources.limits.xpu_device` - accelerator name like configured in the device plugin
- `deployment_parameters.resources.limits.xpu_device_quantity` - number of accelerators
- `deployment_parameters.resources.requests.cpu` - reserved cpu allocation
- `deployment_parameters.resources.requests.memory` - reserved memory allocation
- `deployment_parameters.resources.requests.xpu_device` - accelerator name like configured in the device plugin - should be the same like set in limits or empty
- `deployment_parameters.resources.requests.xpu_device_quantity` - number of accelerators - should be the same like set in limits or empty
Below is the snippet example from the helm chart values.yaml file:
```yaml
deployment_parameters:
  resources:
    limits:
      cpu: 8.0
      memory: 512Mi
```
Beside setting the CPU and memory resources, the same parameter can be used to assign AI accelerators like iGPU, or VPU.
That assumes using adequate Kubernetes device plugin from [Intel Device Plugin for Kubernetes](https://github.com/intel/intel-device-plugins-for-kubernetes).
```yaml
deployment_parameters:
  resources:
    limits:
      xpu_device: gpu.intel.com/i915
      xpu_device_quantity: "1"
```
### Security Context
OVMS, by default, starts with the security context of `ovms` account which has the pid 5000 and gid 5000. In some cases, it can prevent importing models
stored on the file system with restricted access.
It might require adjusting the security context of OVMS service. It can be changed using the parameters  `models_repository.runAsUser` and `models_repository.runAsGroup`.
An example of the values is presented below:
```yaml
models_repository:
  runAsUser: 5000
  runAsGroup: 5000
``` 
The security configuration could be also adjusted further with all options specified in [Kubernetes documentation](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) 
### Service Type
The helm chart creates the Kubernetes `service` as part of the OVMS deployment. Depending on the cluster infrastructure you can adjust
the [service type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types).
In the cloud environment you might set `LoadBalancer` type to expose the service externally. `NodePort` could expose a static port
of the node IP address. `ClusterIP` would keep the OVMS service internal to the cluster applications.  
    
## Demo of Using Model Server with a single model

In this demonstration, it is assumed there is available a Kubernetes or OpenShift cluster with configured security context in the KUBECONFIG. Helm 3 binary and kubectl 1.23 should be also installed to run the commands.
An examplary model server instance with a public [ResNet model](https://github.com/openvinotoolkit/open_model_zoo/tree/master/models/public/resnet-50-tf) can be deployed via a commands:

```
git clone https://github.com/openvinotoolkit/operator
cd operator/helm-charts
helm install ovms-app ovms --set models_settings.model_name=resnet,models_settings.model_path=gs://<bucket_name>/<model_dir>
```

Now that the server is running you can send HTTP or gRPC requests to perform inference. 
By default, the service is exposed with a `ClusterIP` service type. 
```shell script
kubectl get service
NAME                    TYPE           CLUSTER-IP      EXTERNAL-IP    PORT(S)                AGE
ovms-app                ClusterIP      10.98.164.11    <none>         8080/TCP,8081/TCP      5m30s
```
The server exposes an gRPC endpoint on 8080 port and REST endpoint on 8081 port.
The service name deployed via the helm chart is defined by the application name. In addition to that, the service
gets a suffix `-ovms`, in case the application name doesn't include `ovms` phrase. It avoids a risk of the service name
conflicts with other application.
Below is described an example how the model service can be used inside the cluster.
Make an interactive session on the docker container with python installed:
```
kubectl create deployment client-test --image=python:3.8.13 -- sleep infinity
kubectl exec -it $(kubectl get pod -o jsonpath="{.items[0].metadata.name}" -l app=client-test) -- bash
```
REST API response can be verified inside the client container with a simple `curl` command listing the served models:
```
curl http://ovms-app:8081/v1/config
{
"resnet" :
{
 "model_version_status": [
  {
   "version": "1",
   "state": "AVAILABLE",
   "status": {
    "error_code": "OK",
    "error_message": "OK"
   }
  }
 ]
}
```
You can also test a prediction via gRPC interface.
Inside the containers run the following commands to install the client package and download an image to classify:
```
pip install ovmsclient
wget https://raw.githubusercontent.com/openvinotoolkit/model_server/main/demos/common/static/images/bee.jpeg
```
Create a python script with a basic client content:
```
cat >> predict.py <<EOL
from ovmsclient import make_grpc_client
import numpy as np
client = make_grpc_client("ovms-app:8080")
with open("bee.jpeg", "rb") as f:
    data = f.read()
inputs = {"map/TensorArrayStack/TensorArrayGatherV3:0": data}
results = client.predict(inputs=inputs, model_name="resnet")
print("Detected class:", np.argmax(results))
EOL
```
Run the prediction via a command:
```
python client.py
Detected class: 310
```
Class 310 represents a bee in the [Imagenet dataset](https://image-net.org/).

## Demo - deployment of the Model Server with a Vehicle analysis pipeline
This demonstration deploys the model server serving a directed acyclic graph with [vehicle analysis](https://github.com/openvinotoolkit/model_server/tree/main/demos/vehicle_analysis_pipeline/python) in Kubernetes.
Requirements:
- Kubernetes or OpenShift cluster with configured security context in the KUBECONFIG
- helm 3
- kubectl 1.23
- mc binary and access to S3 compatible bucket - [quick start with Minio](https://docs.min.io/docs/minio-quickstart-guide.html)

---
### Quick standalone minio setup
If you don't have a minio in place, you can move forward with simple, standalone setup. Run:
```
kubectl apply https://github.com/openvinotoolkit/operator/helm-charts/minio/minio-standalone.yaml
```
---

Prepare all dependencies for the pipeline with a vehicle analysis pipelines:
```
git clone https://github.com/openvinotoolkit/model_server
cd model_server/demos/vehicle_analysis_pipeline/python
make
```
The command above downloads the models and builds the customer library for the pipeline and places them in workspace folder. Copy the models to the shared storage accessible in the cluster. Here the S3 server alias is `mys3`:
```
mc cp --recursive workspace/vehicle-detection-0202 mys3/models-repository/
mc cp --recursive workspace/vehicle-attributes-recognition-barrier-0042 mys3/models-repository/
mc ls -r mys3
43MiB models-repository/vehicle-attributes-recognition-barrier-0042/1/vehicle-attributes-recognition-barrier-0042.bin
118KiB models-repository/vehicle-attributes-recognition-barrier-0042/1/vehicle-attributes-recognition-barrier-0042.xml
7.1MiB models-repository/vehicle-detection-0202/1/vehicle-detection-0202.bin
331KiB models-repository/vehicle-detection-0202/1/vehicle-detection-0202.xml
```
In the initially created model server config file `workspace/config.json`, several adjustments are needed to change the models and custom node library base paths.
Commands below set the models path to S3 bucket.
```
sed -i 's/\/workspace\/vehicle-detection-0202/s3:\/\/models-repository\/vehicle-detection-0202/g' workspace/config.json
sed -i 's/\/workspace\/vehicle-attributes-recognition-barrier-0042/s3:\/\/models-repository\/vehicle-attributes-recognition-barrier-0042/g' workspace/config.json
```

Next, add the config file  to a config map:
```
kubectl create configmap ovms-pipeline --from-file=config.json=workspace/config.json
```

From the context of the helm chart folder in the operator repo deploy the model server. Change the credentials and S3 endpoint as needed in your environment:
```
git clone https://github.com/openvinotoolkit/operator
cd operator/helm-charts
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_REGION=us-east-1
export S3_COMPAT_API_ENDPOINT=http://minio-service:9000
helm install ovms-pipeline ovms --set models_settings.config_configmap_name=ovms-pipeline,models_settings.single_model_mode=false,models_repository.aws_access_key_id=$AWS_ACCESS_KEY_ID,models_repository.aws_secret_access_key=$AWS_SECRET_ACCESS_KEY,models_repository.aws_region=us-east-1,models_repository.s3_compat_api_endpoint=$S3_COMPAT_API_ENDPOINT

$ kubectl get service
NAME                     TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)              AGE
ovms-pipeline            ClusterIP      10.99.53.175     <none>        8080/TCP,8081/TCP    26m
```
Now we are ready to test the pipeline from the client container. Make an interactive session on the docker container with python installed:
```
kubectl create deployment client-test --image=python:3.8.13 -- sleep infinity
kubectl exec -it $(kubectl get pod -o jsonpath="{.items[0].metadata.name}" -l app=client-test) -- bash
```

Inside the containers run the following commands to install the client package and download an image to classify:
```
pip install ovmsclient
wget https://raw.githubusercontent.com/openvinotoolkit/model_server/main/demos/common/static/images/cars/road1.jpg
```
Create a python script with a basic client content:
```
cat >> pipeline.py <<EOL
from ovmsclient import make_grpc_client
import numpy as np
client = make_grpc_client("ovms-pipeline:8080")
with open("road1.jpg", "rb") as f:
    data = f.read()
inputs = {"image": data}
results = client.predict(inputs=inputs, model_name="multiple_vehicle_recognition")
print("Returned outputs:",results.keys())
EOL
```
Run the prediction via a command:
```
$ python pipeline.py
Returned outputs: dict_keys(['colors', 'vehicle_coordinates', 'types', 'vehicle_images', 'confidence_levels'])
```


## Cleanup
Once you've finished using the server you should use helm to uninstall the chart:
```shell script
$ helm ls
NAME                    NAMESPACE               REVISION        UPDATED                                         STATUS          CHART           APP VERSION
ovms-app                default                 1               2022-04-11 13:39:44.11018803 +0200 CEST         deployed        ovms-4.0.0
ovms-pipeline           default                 1               2022-04-11 15:12:28.279846055 +0200 CEST        deployed        ovms-4.0.0

$ helm uninstall ovms-app
release "ovms-app" uninstalled
$ helm uninstall ovms-pipeline
release "ovms-pipeline" uninstalled
```

***
Check also:
- [Model Server parameters explained](../../docs/modelserver_params.md)
