# Model server parameters

| Parameter        | Description  |
| ------------- |-------------|
|image_name| model server docker image. The default is the latest public docker image |
|deployment_parameters.replicas| number if model server replicas to be used. In case if enabled autoscaling, it defines the initial number of replicas|
|deployment_parameters.openshift_service_mesh| When the value is `true`, it adds the annotations enabling the models server deployment for [OpenShift Service Mesh](https://docs.openshift.com/container-platform/4.10/service_mesh/v2x/ossm-about.html)|
|service_parameters.grpc_port| gRPC service port; the default value is 8080|
|service_parameters.rest_port| REST API service port; the default value is 8081|
|service_parameters.service_type| [service type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types); the default value is ClusterIP|
|models_settings.single_model_mode| set `true` if one one model should be deployed; value `false` indicate that config.json file should be used to configure mulitple models |
|models_settings.config_configmap_name| Config map hosting the config.json file|
|models_settings.config_path| Path to the config file in case it was mounted in the container via a persistent volume claim |
|models_settings.model_name| Model name to be used on the client side in the remote calls |
|models_settings.model_path| Path to the model folder in the model repository; for example `gs://ovms-public-eu/resnet50-binary` |
|models_settings.nireq| The size of internal request queue. When set to 0 or no value is set value is calculated automatically based on available resources|
|models_settings.plugin_config| Adds OpenVINO plugin configuration for tuning the performance. Value `{\"PERFORMANCE_HINT\":\"LATENCY\"}` optimizes the inference latency with a single client scenario|
|models_settings.batch_size| change the model batch size |
|models_settings.shape| shape is optional and takes precedence over batch_size. The shape argument changes the model that is enabled in the model server to fit the parameters. shape accepts three forms of the values: a tuple, such as (-1,3,100-200,224) - The tuple defines the shape to use for all incoming requests for models with a single input. Each dimension can be a static value `3`, a range `100-200` or `-1` which is undefined value. A dictionary of shapes, such as {"input1":"(1,3,224,224)","input2":"(1,3,50,50)", "input3":"auto"} set shape for multiple inputs|
|models_settings.model_version_policy| '{"latest": { "num_versions":1 }}'|
|models_settings.layout| Change layout of the model input or output with image data; NCHW:NHWC changes the layout from NCHW to NHWC|
|models_settings.target_device| Any supported OpenVINO target device like CPU/GPU/HDDL/MULTI/HETERO/AUTO|
|models_settings.is_stateful| set `true` it the model is stateful|
|models_settings.idle_sequence_cleanup| If set to true, model will be subject to periodic sequence cleaner scans. See idle sequence cleanup|
|models_settings.low_latency_transformation| If set to true, model server will apply low latency transformation on model load|
|models_settings.max_sequence_number|Determines how many sequences can be handled concurrently by a model instance.|
|server_settings.file_system_poll_wait_seconds| Time interval between config and model versions changes detection in seconds. Default value is 1. Zero value disables changes monitoring.|
|server_settings.log_level| One of ERROR/WARNING/INFO/DEBUG|
|server_settings.grpc_workers| number of gRPC servers; default is 1|
|server_settings.rest_workers| number of REST server threads; default is calculated automatically|
|models_repository.https_proxy| proxy to be used to pull cloud storage models|
|models_repository.http_proxy|proxy to be used to pull cloud storage models|
|models_repository.storage_type| one of `google storage`, `s3`, `azure blob` or `cluster`|
|models_repository.models_host_path| Mounts node local path in container as /models folder | Path should be created on all nodes and populated with the data|
|models_repository.models_volume_claim| Mounts persistent volume claim in the container as /models; persistent Volume Claim should be create in the same namespace and populated with the model repository content|
|models_repository.runAsUser| account security context|
|models_repository.runAsGroup| group security context|
|models_repository.aws_secret_access_key| S3 storage secret key, use it with S3 storage for models|
|models_repository.aws_access_key_id| S3 storage access key id, use it with S3 storage for models|
|models_repository.aws_region| S3 storage secret key, use it with S3 storage for models|
|models_repository.s3_compat_api_endpoint| S3 compatibility api endpoint, use it with Minio storage for models|
|models_repository.gcp_creds_secret_name| secret resource including GCP credentials, use it with google storage for models; create it via `kubectl create secret generic <secret name> --from-file gcp-creds.json`|
|models_repository.azure_storage_connection_string|Connection string to the Azure Storage authentication account, use it with Azure storage for models|

Check an example of the [fully functional ModelServer resource](../config/samples/intel_v1alpha1_ovms.yaml)