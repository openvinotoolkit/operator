#
# Copyright (c) 2022 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Build the manager binary
FROM docker.io/golang:1.21 as builder
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN GOOS=linux GOARCH=$TARGETARCH make build

# Unit tests
RUN GOOS=linux GOARCH=$TARGETARCH make test-unit

# Final image.
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
### Required OpenShift Labels 
LABEL name="OpenVINO toolkit operator" \
      maintainer="dariusz.trawinski@intel.com" \
      vendor="Intel Corporation" \
      version="1.1.0" \
      release="1.1" \
      summary="OpenVINO(TM) Toolkit Operator" \
      description="An Operator for managing OpenVINO Toolkit in OpenShift and Kubernetes"

ENV HOME=/opt/openvino \
    USER_NAME=openvino \
    USER_UID=5000

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}


COPY --from=builder /workspace/build/openvino-operator /usr/local/bin/openvino-operator
COPY watches.yaml watches.yaml
COPY watches_k8s.yaml watches_k8s.yaml
COPY helm-charts helm-charts
COPY LICENSE .
COPY licenses /licenses

USER ${USER_UID}
ENTRYPOINT ["/usr/local/bin/openvino-operator", "run"]