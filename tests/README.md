## Preparing K8S environment for tests

NOTE: The procedure was tested for both Ubuntu 20.04 and Ubuntu 22.04.

Before preparing K8S environment, make sure you meet the prerequisites 
defined in [developer guide](https://github.com/openvinotoolkit/operator/blob/main/docs/developer_guide.md). 

1. Install required CLI tools as described in [Kubernetes documentation](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#installing-kubeadm-kubelet-and-kubectl).
To install specific version, check current stable version using ```apt policy <package_name>```, e.g. ```apt policy kubectl```.

```bash
set -e
sudo apt-get update -y
sudo apt-get install -y apt-transport-https ca-certificates curl gpg
sudo mkdir -p -m 755 /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.28/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.28/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update -y
sudo apt-get install -y kubelet=1.28.7-1.1 kubeadm=1.28.7-1.1 kubectl=1.28.7-1.1
sudo apt-mark hold kubelet kubeadm kubectl
echo "CLI tools installed"
```

2. Setup containerd

If you already have any version of containerd installed, do the cleanup: 

```bash
sudo apt remove containerd
sudo rm /etc/containerd/config.toml
```

Set up Docker's apt repository as described in [Docker documentation](https://docs.docker.com/engine/install/ubuntu/#install-using-the-repository).
```bash
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update -y
```

Then, install containerd:

```bash
sudo apt install containerd.io
sudo systemctl restart containerd
```

3. Initialize a Kubernetes control-plane node

```bash
set -e
sudo swapoff -a
sudo kubeadm reset -f
sudo kubeadm init --pod-network-cidr=10.244.0.0/16
sudo cp -f /etc/kubernetes/admin.conf ${HOME}/.kube/config
sudo chown ${id} ${HOME}/.kube/config
kubectl apply -f https://github.com/coreos/flannel/raw/master/Documentation/kube-flannel.yml
kubectl taint nodes --all node-role.kubernetes.io/control-plane:NoSchedule-
kubectl get pod --all-namespaces
echo "cluster installed"
```

4. Install OLM in K8S cluster manually
```bash
operator-sdk olm install --version v0.27.0
operator-sdk olm status
echo "OLM installed"
```

or using Makefile

```bash
cd ..
make cluster_clean
echo "OLM installed"
```


5. Configure ImageStream and BuildConfig CRDs
```bash
kubectl apply -f os-crds.yaml
kubectl get crds
echo "CRDs installed"
```
