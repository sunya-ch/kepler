# Model Module
This module has two components connecting to external servers.
1. Connector to model server (model)
    - Load trained model from [model server](https://github.com/sustainable-computing-io/kepler-model-server)
2. Connector to InfluxDB2 server (influxdb2)
    - Write train data to Influx DB which will be read by [model server](https://github.com/sustainable-computing-io/kepler-model-server)

### Requirements
- InfluxDB2 (default: host path `/var/lib/influxdb`)
    1. Set up persistent volume [ref.](https://github.com/influxdata/influxdata-operator#persistent-volumes)
        ```bash
        kubectl apply -f db-resources
        ```
    2. Install InfluxDB2 via HELM [ref.](https://github.com/influxdata/helm-charts)
        ```bash
        helm install --namespace influxdb \
        --set persistence.enabled=true,persistence.storageClass=local-storage,persistence.size=100Gi \
        influxdb-release influxdata/influxdb2
        ```

- Environment variables in kepler container
    1. Copy secret from influxdb namespace to monitoring (kepler) namespace
        ```bash
        kubectl get secret my-tlssecret --namespace=influxdb -o yaml | sed 's/namespace: .*/namespace: monitoring/' | kubectl apply -f -
        ```
    2. Add following environment varaibles
        ```yaml
            env:
            - name: INFLUXDB_ENDPOINT
            value: http://influxdb-release-influxdb2.influxdb.svc.cluster.local:80
            - name: INFLUXDB_TOKEN
            valueFrom:
                secretKeyRef:
                key: admin-token
                name: influxdb-release-influxdb2-auth
            - name: INFLUXDB_ORG
            value: influxdata
            - name: INFLUXDB_BUCKET
            value: default
        ```
    3. Use `ClusterFirstWithHostNet` for dnsPolicy
        ```yaml
        dnsPolicy: ClusterFirstWithHostNet
        ```