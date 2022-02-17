# Build the manager binary
FROM docker.io/golang:1.17 as builder
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
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

ENV HOME=/opt/openvino \
    USER_NAME=openvino \
    USER_UID=5000

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}
USER ${USER_UID}

COPY --from=builder /workspace/build/openvino-operator /usr/local/bin/openvino-operator
COPY watches.yaml watches.yaml
COPY helm-charts helm-charts
COPY LICENSE .
COPY licenses /licenses

ENTRYPOINT ["/usr/local/bin/openvino-operator", "run", "--watches-file=./watches.yaml"]