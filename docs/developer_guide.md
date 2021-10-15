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



