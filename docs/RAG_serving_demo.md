# Retrieval Augmented Generation with OpenVINO Model Server demo

This demo shows how to deploy a service based on OpenVINO Model Server in Kubernetes for generative use cases. OpenVINO Model Server can be used to expose `chat/completions` OpenAI API which is the key building block of a RAG application. 


Here are the steps we are going to follow:
- downloading LLM model from Hugging Faces and quantization for better execution efficiency
- uploading the model to the Persistent Volume Claim
- deploying the service via the operator and ModelServer custom resource
- running text generation client and the RAG application with the OpenVINO Model Server as the endpoint for text generation


## Downloading LLM model from Hugging Faces and quantization for better execution efficiency

In this step the original Pytorch LLM model and the tokenizer will be converted to IR format and optionally quantized.
That ensures faster initialization time, better performance and lower memory consumption.
Here, we will also define the LLM engine parameters inside the `graph.pbtxt`.

Install python dependencies for the conversion script:
```bash
export PIP_EXTRA_INDEX_URL="https://download.pytorch.org/whl/cpu https://storage.openvinotoolkit.org/simple/wheels/nightly"
pip3 install --pre "optimum-intel[nncf,openvino]"@git+https://github.com/huggingface/optimum-intel.git openvino-tokenizers
```

Run optimum-cli to download and quantize the model:
```bash
cd demos/continuous_batching
optimum-cli export openvino --disable-convert-tokenizer --model meta-llama/Meta-Llama-3-8B-Instruct --weight-format int8 Meta-Llama-3-8B-Instruct
convert_tokenizer -o Meta-Llama-3-8B-Instruct --with-detokenizer --skip-special-tokens --streaming-detokenizer --not-add-special-tokens meta-llama/Meta-Llama-3-8B-Instruct
```
> **Note:** Before downloading the model, access must be requested. Follow the instructions on the [HuggingFace model page](https://huggingface.co/meta-llama/Meta-Llama-3-8B) to request access. When access is granted, create an authentication token in the HuggingFace account -> Settings -> Access Tokens page. Issue the following command and enter the authentication token. Authenticate via `huggingface-cli login`.

Copy the graph to the model folder. 
```bash
cat <<EOF > Meta-Llama-3-8B-Instruct/graph.pbtxt
input_stream: "HTTP_REQUEST_PAYLOAD:input"
output_stream: "HTTP_RESPONSE_PAYLOAD:output"

node: {
  name: "LLMExecutor"
  calculator: "HttpLLMCalculator"
  input_stream: "LOOPBACK:loopback"
  input_stream: "HTTP_REQUEST_PAYLOAD:input"
  input_side_packet: "LLM_NODE_RESOURCES:llm"
  output_stream: "LOOPBACK:loopback"
  output_stream: "HTTP_RESPONSE_PAYLOAD:output"
  input_stream_info: {
    tag_index: 'LOOPBACK:0',
    back_edge: true
  }
  node_options: {
      [type.googleapis.com / mediapipe.LLMCalculatorOptions]: {
          models_path: "./",
          cache_size: 20
      }
  }
  input_stream_handler {
    input_stream_handler: "SyncSetInputStreamHandler",
    options {
      [mediapipe.SyncSetInputStreamHandlerOptions.ext] {
        sync_set {
          tag_index: "LOOPBACK:0"
        }
      }
    }
  }
}
EOF
```

The result should be like below:
```bash
tree Meta-Llama-3-8B-Instruct/
Meta-Llama-3-8B-Instruct
├── config.json
├── generation_config.json
├── graph.pbtxt
├── openvino_detokenizer.bin
├── openvino_detokenizer.xml
├── openvino_model.bin
├── openvino_model.xml
├── openvino_tokenizer.bin
├── openvino_tokenizer.xml
├── special_tokens_map.json
├── tokenizer_config.json
├── tokenizer.json
└── tokenizer.model
```
## Uploading the model to Persistent Volume Claim

The model content is expected to be stored in the PVC in the cluster. It will be accessible in the model server instances for quick initialization. That way the model server replicas don't need to download the model from Internet and rerun the quantization or compression during every start up.

In this demo there will be created hostPath PersistentVolume. In production use cases it should be replaced with difference storage class like NFS or Amazon Elastic Block Store.

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolume
metadata:
  name: llm-pv-volume
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 50Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/opt/data"
EOF
```

Create PersistentVolumeClaim
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: llm-pv-claim
spec:
  storageClassName: manual
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
EOF
```

Copy the model to the volume.

```bash
cp -R Meta-Llama-3-8B-Instruct /opt/data/
```

## Deploying the service via the operator and ModelServer custom resource

The first step will be to add the model server config to the configmap.
It is followed by create the ModelServer resource. It is using the created configmap and the PVC.

```bash
cat <<EOF > config.json
{
    "model_config_list": [],
    "mediapipe_config_list": [
        {
            "name": "meta-llama/Meta-Llama-3-8B-Instruct",
            "base_path": "/models/Meta-Llama-3-8B-Instruct"
        }
    ]
}
EOF

kubectl create configmap llm-config --from-file=config.json=config.json

kubectl apply -f - <<EOF
apiVersion: intel.com/v1alpha1
kind: ModelServer
metadata:
  name: ovms-llm
spec:
  image_name: openvino/model_server:latest
  deployment_parameters:
    replicas: 1
  models_settings:
    single_model_mode: false
    config_configmap_name: "llm-config"
  server_settings:
    log_level: "INFO"
  service_parameters:
    grpc_port: 8080
    rest_port: 8081
    service_type: NodePort
  models_repository:
    models_volume_claim: llm-pv-claim
EOF
```

It will deploy the model server pod with a linked service
```bash
kubectl get deploy
NAME       READY   UP-TO-DATE   AVAILABLE   AGE
ovms-llm   1/1     1            1           15m

kubectl get service ovms-llm
NAME       TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)                         AGE
ovms-llm   NodePort   10.233.23.238   <none>        8080:30018/TCP,8081:30023/TCP   15m
```

Inside the cluster, the service can be accessible via URL http://ovms-llm:8081 or externally over NodePort http://10.233.23.238:30023.

Here is a simple test how the endpoint can be used just to generate text:

```bash
curl -s http://ovms-llm:8081/v3/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Meta-Llama-3-8B-Instruct",
    "max_tokens":30,
    "stream":false,
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "What is OpenVINO?"
      }
    ]
  }'| jq .

{
  "choices": [
    {
      "finish_reason": "stop",
      "index": 0,
      "logprobs": null,
      "message": {
        "content": "OpenVINO is an open-source software framework developed by Intel that enables developers to optimize and deploy artificial intelligence (AI) and computer vision (CV)",
        "role": "assistant"
      }
    }
  ],
  "created": 1719921018,
  "model": "meta-llama/Meta-Llama-3-8B-Instruct",
  "object": "chat.completion"
}

```

## Running the RAG application with the OpenVINO Model Server as the endpoint for text generation

Deployed above model server with OpenAI API is a fundamental part of the RAG chain. 
It can be employed by just pointing the RAG chain to the correct URL representing OpenVINO Model Server endpoint.

Start the jupyter notebook via a command:
```bash
 jupyter-lab
```
Open and console in your browser.
Import the notebook [rag_demo.ipynb](https://github.com/openvinotoolkit/model_server/blob/main/demos/continuous_batching/rag/rag_demo.ipynb) and 
and [requirements.txt](https://github.com/openvinotoolkit/model_server/blob/main/demos/continuous_batching/rag/requirements.txt)

In the notebook script just adjust `openai_api_base` parameter to the exposed or internal IP or DNS name, used port and followed by `v3`.
For example `http://ovms-llm:8081/v3` or `http://10.233.23.238:30023/v3`.
