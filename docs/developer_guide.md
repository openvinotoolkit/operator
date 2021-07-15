# Develper guide
## Building  the operator
```
make build
```
Results are stored in build folder.

## Testing the operator integration with Kubernetes

Start the operator
```bash
./build/openvino-operator run --watches-file watches.yaml
```

## Build docker image

```bash
build -f docker/Dockerfile .
```



