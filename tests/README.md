## Preparing K8S environment for tests

```
sudo apt install -y kubeadm kubelet
sudo kubeadm reset -f
sudo kubeadm init --pod-network-cidr=10.244.0.0/16
sudo cp -f /etc/kubernetes/admin.conf ${HOME}/.kube/config
sudo chown ${id} ${HOME}/.kube/config
kubectl apply -f https://github.com/coreos/flannel/raw/master/Documentation/kube-flannel.yml
kubectl taint nodes --all node-role.kubernetes.io/master-
kubectl get pod --all-namespaces
echo "cluster installed"
```

Install OLM
```
echo "installing olm"
operator-sdk olm install
operator-sdk olm status
```

Configure ImageStream and BuildConfig CRDs
```
kubectl apply -f tests/os-crds.yaml
```

