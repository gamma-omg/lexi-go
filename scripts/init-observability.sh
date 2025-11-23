#!/bin/sh

set -e

helm repo add grafana https://grafana.github.io/helm-charts || true
helm repo update

# loki
echo "[*] Deploying Loki Stack..."
helm upgrade \
    --install loki grafana/loki \
    --namespace observability \
    --values deploy/observability/dev/loki-values.yml
echo "[*] Loki Stack deployed."

# alloy
echo "[*] Deploying Alloy..."
helm upgrade \
    --install alloy grafana/alloy \
    --namespace observability \
    --values deploy/observability/dev/alloy-values.yml
echo "[*] Alloy deployed."

# grafana
echo "[*] Deploying Grafana..."
helm upgrade \
    --install grafana grafana/grafana \
    --namespace observability \
    --values deploy/observability/dev/grafana-values.yml
echo "[*] Grafana deployed."