#!/usr/bin/env bash
set -euo pipefail

cluster_name="${KIND_CLUSTER_NAME:-go-httpbin-ci}"
app_manifest="${KUBE_MANIFEST:-deploy/kubernetes/go-httpbin.yaml}"
gateway_manifest="${GATEWAY_MANIFEST:-deploy/kubernetes/gateway.yaml}"
namespace="${KUBE_NAMESPACE:-go-httpbin}"
image="${KIND_IMAGE:-ghcr.io/vancanhuit/go-httpbin:latest}"
hostname="${GATEWAY_HOSTNAME:-go-httpbin.local}"
create_cluster="${KIND_CREATE_CLUSTER:-true}"
provider_log="${RUNNER_TEMP:-/tmp}/cloud-provider-kind.log"
cluster_created=0
provider_pid=""

cleanup() {
  if [[ -n "${provider_pid}" ]]; then
    kill "${provider_pid}" 2>/dev/null || true
  fi

  if [[ "${cluster_created}" == "1" ]]; then
    kind delete cluster --name "${cluster_name}"
  fi
}

trap cleanup EXIT

for command in cloud-provider-kind docker kind kubectl curl; do
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

kubectl label node --all node.kubernetes.io/exclude-from-external-load-balancers- >/dev/null 2>&1 || true

cloud-provider-kind --gateway-channel standard >"${provider_log}" 2>&1 &
provider_pid="$!"

gatewayclass_ready=0
for _ in $(seq 1 60); do
  if kubectl get gatewayclass cloud-provider-kind >/dev/null 2>&1; then
    gatewayclass_ready=1
    break
  fi

  if ! kill -0 "${provider_pid}" 2>/dev/null; then
    echo "cloud-provider-kind exited before GatewayClass became available" >&2
    cat "${provider_log}" >&2 || true
    exit 1
  fi

  sleep 2
done

if [[ "${gatewayclass_ready}" != "1" ]]; then
  echo "timed out waiting for cloud-provider-kind GatewayClass" >&2
  cat "${provider_log}" >&2 || true
  exit 1
fi

kind load docker-image "${image}" --name "${cluster_name}"

kubectl apply -f "${app_manifest}"
kubectl -n "${namespace}" rollout status deployment/go-httpbin --timeout=120s
kubectl -n "${namespace}" wait --for=condition=ready pod -l app.kubernetes.io/name=go-httpbin --timeout=120s

kubectl apply -f "${gateway_manifest}"
kubectl -n "${namespace}" wait --for=condition=Programmed gateway/go-httpbin --timeout=120s

conditions=""
for _ in $(seq 1 60); do
  conditions="$(kubectl -n "${namespace}" get httproute/go-httpbin -o jsonpath='{range .status.parents[*].conditions[*]}{.type}={.status}{"\n"}{end}')"
  if [[ "${conditions}" == *"Accepted=True"* ]] && [[ "${conditions}" == *"ResolvedRefs=True"* ]]; then
    break
  fi

  sleep 2
done

if [[ "${conditions}" != *"Accepted=True"* ]] || [[ "${conditions}" != *"ResolvedRefs=True"* ]]; then
  echo "HTTPRoute conditions did not report Accepted=True and ResolvedRefs=True" >&2
  echo "${conditions}" >&2
  exit 1
fi

gateway_address=""
for _ in $(seq 1 60); do
  gateway_address="$(kubectl -n "${namespace}" get gateway/go-httpbin -o jsonpath='{.status.addresses[0].value}' 2>/dev/null || true)"
  if [[ -n "${gateway_address}" ]]; then
    break
  fi

  if ! kill -0 "${provider_pid}" 2>/dev/null; then
    echo "cloud-provider-kind exited before Gateway address became available" >&2
    cat "${provider_log}" >&2 || true
    exit 1
  fi

  sleep 2
done

if [[ -z "${gateway_address}" ]]; then
  echo "timed out waiting for Gateway address" >&2
  kubectl -n "${namespace}" get gateway go-httpbin -o yaml >&2 || true
  cat "${provider_log}" >&2 || true
  exit 1
fi

for _ in $(seq 1 30); do
  health_response="$(curl --fail --silent --header "Host: ${hostname}" "http://${gateway_address}/healthz" 2>/dev/null || true)"
  if [[ "${health_response}" == *'"status":"ok"'* ]]; then
    curl --fail --silent --show-error --header "Host: ${hostname}" "http://${gateway_address}/get" >/dev/null
    exit 0
  fi

  sleep 2
done

echo "timed out waiting for go-httpbin through Gateway API at ${gateway_address}" >&2
kubectl -n "${namespace}" get gateway,httproute
kubectl -n "${namespace}" describe gateway/go-httpbin
kubectl -n "${namespace}" describe httproute/go-httpbin
kubectl -n "${namespace}" logs deployment/go-httpbin --tail=100 || true
cat "${provider_log}" >&2 || true
exit 1
