# Steps

### 1. Prometheus preparation
##### Deploy prometheus
```bash
kubectl apply -f manifests/setup
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
##### Remote ssh for prometheus query
```bash
ssh -L 9090:localhost:30090 singlecpu-vm
```
http://localhost:9090

##### Deploy Kepler service monitor
```bash
kubectl apply -f https://raw.githubusercontent.com/sunya-ch/kepler/model-dev/manifests/vm-sample/service-monitor.yaml
```
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
