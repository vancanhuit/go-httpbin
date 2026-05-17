#!/usr/bin/env bash
set -euo pipefail

cluster_name="${KIND_CLUSTER_NAME:-go-httpbin-ci}"
manifest="${KUBE_MANIFEST:-deploy/kubernetes/go-httpbin.yaml}"
namespace="${KUBE_NAMESPACE:-go-httpbin}"
image="${KIND_IMAGE:-ghcr.io/vancanhuit/go-httpbin:latest}"
local_port="${KUBE_LOCAL_PORT:-18080}"
create_cluster="${KIND_CREATE_CLUSTER:-true}"
port_forward_log="${RUNNER_TEMP:-/tmp}/go-httpbin-port-forward.log"
cluster_created=0
port_forward_pid=""

cleanup() {
  if [[ -n "${port_forward_pid}" ]]; then
    kill "${port_forward_pid}" 2>/dev/null || true
  fi

  if [[ "${cluster_created}" == "1" ]]; then
    kind delete cluster --name "${cluster_name}"
  fi
}

trap cleanup EXIT

for command in docker kind kubectl curl; do
  if ! command -v "${command}" >/dev/null 2>&1; then
    echo "missing required command: ${command}" >&2
    exit 1
  fi
done

docker build --tag "${image}" .

if [[ "${create_cluster}" == "true" ]]; then
  kind create cluster --name "${cluster_name}" --wait 120s
  cluster_created=1
fi

kind load docker-image "${image}" --name "${cluster_name}"

kubectl apply -f "${manifest}"
kubectl -n "${namespace}" rollout status deployment/go-httpbin --timeout=120s
kubectl -n "${namespace}" wait --for=condition=ready pod -l app.kubernetes.io/name=go-httpbin --timeout=120s

kubectl -n "${namespace}" port-forward --address 127.0.0.1 service/go-httpbin "${local_port}:80" >"${port_forward_log}" 2>&1 &
port_forward_pid="$!"

for _ in $(seq 1 30); do
  if ! kill -0 "${port_forward_pid}" 2>/dev/null; then
    echo "kubectl port-forward exited before go-httpbin became reachable" >&2
    cat "${port_forward_log}" >&2 || true
    exit 1
  fi

  health_response="$(curl --fail --silent "http://127.0.0.1:${local_port}/healthz" 2>/dev/null || true)"
  if [[ "${health_response}" == *'"status":"ok"'* ]]; then
    curl --fail --silent --show-error "http://127.0.0.1:${local_port}/get" >/dev/null
    exit 0
  fi

  sleep 2
done

echo "timed out waiting for go-httpbin through kubectl port-forward on 127.0.0.1:${local_port}" >&2
kubectl -n "${namespace}" get all
kubectl -n "${namespace}" describe deployment/go-httpbin
kubectl -n "${namespace}" logs deployment/go-httpbin --tail=100 || true
cat "${port_forward_log}" >&2 || true
exit 1
