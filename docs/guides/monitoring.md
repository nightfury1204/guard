---
title: Prometheus Monitoring | Guard
description: Prometheus Monitoring
menu:
  product_guard_{{ .version }}:
    identifier: prometheus-monitoring
    name: Monitoring
    parent: guides
    weight: 25
product_name: guard
menu_name: product_guard_{{ .version }}
section_menu_id: guides
---

# Prometheus Monitoring

## Service Monitor for Prometheus-Operator

Create a ServiceMonitor for [Prometheus-Operator](https://github.com/coreos/prometheus-operator) to automatically scrape Guard's metrics endpoint.

```
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: guard
  labels:
    app: guard
spec:
  endpoints:
  - interval: 30s
    path: /metrics
    port: api
    scheme: https
    tlsConfig:
      insecureSkipVerify: true
  namespaceSelector:
    any: true
  selector:
    matchLabels:
      app: guard
```

If prometheus-operator and kube-prometheus is installed using CoreOS's [helm charts](https://github.com/coreos/prometheus-operator/tree/master/helm), the serviceMonitor can be defined in kube-prometheus's values.yaml.

```
prometheus:
  serviceMonitors:
    - name: guard
      labels:
        prometheus: kube-prometheus
      selector:
        matchLabels:
          app: guard
      endpoints:
        - port: api
          interval: 30s
          path: /metrics
          scheme: https
          tlsConfig:
            insecureSkipVerify: true
      namespaceSelector:
        any: true
```

# Grafana Dashboard for Guard

A simple Grafana dashbord for Guard can be found [here](https://github.com/appscode/guard/raw/master/contrib/Guard-grafana-dashboard.json)
