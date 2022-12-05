# Steps

### 1. Prometheus preparation
##### Deploy prometheus
```bash
git clone https://github.com/prometheus-operator/kube-prometheus.git
cd kube-prometheus
kubectl create -f manifests/setup
kubectl apply -f manifests/
```
##### Remove alert-manager
```bash
kubectl edit alertmanagers.monitoring.coreos.com -n monitoring
# spec:
#   paused: true
kubectl delete statefulset -n monitoring alertmanager-main
```
##### Deploy NodePort for remote ssh.
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/kepler/model-dev/manifests/vm-sample/prometheus-np.yaml
```
##### Deploy Kepler service monitor
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/kepler/model-dev/manifests/vm-sample/service-monitor.yaml
```
##### Remote ssh for prometheus query
```bash
ssh -L 9090:localhost:30090 singlecpu-vm
ssh -L 9091:localhost:30090 singlecpu
```
- VM
http://localhost:9090 

- BM
http://localhost:9091
### 2. Deploy Kepler
#### Local LR
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/kepler/model-dev/manifests/vm-sample/kepler-local.yaml
```
#### Sidecar
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/kepler/model-dev/manifests/vm-sample/kepler-sidecar.yaml
```
add the following environment variable to estimator sidecar container to see the sidecar log
```yaml
        env:
        - name: PYTHONUNBUFFERED
          value: "1
```
#### BM
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/kepler/model-dev/manifests/vm-sample/kepler-bm.yaml
```
### 3.Deploy workload
##### Deploy CPE operator
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/energy-measurement-data/main/tool/cpe_deploy.yaml
```
##### Deploy workload operator
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/energy-measurement-data/main/tool/coremark/cpe_v1_none_operator.yaml
```
##### Deploy workload
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/energy-measurement-data/main/tool/coremark/cpe_v1_coremark.yaml
```
See more info:
- Clone tool repository
```bash
git clone https://github.com/sunya-ch/energy-measurement-data.git
cd energy-measurement-data/tool
```
- Use cpec tool to check log/status
```bash
export KUBECONFIG_FILE=~/.kube/config
./cpec-$(go env GOOS)-$(go env GOARCH) help
# if go is not installed, use the compatible bin file
# ./cpec-linux-amd64 help
	# Usage
	# 	./cpec status                    - get controller status (KUBECONFIG_FILE required)
	# 	./cpec tune                      - get tuning configure (KUBECONFIG_FILE required)
 	# 	./cpec blogs [blogs options]     - get benchmark log (COS_CONFIG_FILE required)
	# 	./cpec clogs [clogs options]     - get controller log (KUBECONFIG_FILE required)
	# 	./cpec reset [controller|parser] - reset controller or parser
	# 	./cpec save [save options]       - save current benchmark (KUBECONFIG_FILE required)
```


# DNS Issue fixing CentOS
```
modprobe br_netfilter
echo '1' > /proc/sys/net/bridge/bridge-nf-call-iptables
```

https://github.com/kubernetes/kubernetes/issues/21613#issuecomment-343190401

Then, restart coredns pods.

Check,

```
kubectl apply -f https://k8s.io/examples/admin/dns/dnsutils.yaml
kubectl exec -i -t dnsutils -- nslookup kubernetes.default
```