#!/usr/bin/env bash

# Copyright The KMCP Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# install-substrate-kind.sh — stand up Agent Substrate on a local kind cluster
# entirely from published release artifacts. No kagent-dev/substrate checkout
# is required: the control plane installs from the published Helm charts and
# all images (including the ateom-gvisor worker image) are pulled from
# ghcr.io/kagent-dev/substrate. The only thing this script reproduces from the
# fork's own hack/ scripts is the kind cluster shape (certificate feature gates
# + proxy-ARP + a local registry for the kmcp controller image).
#
# Usage:
#   hack/install-substrate-kind.sh                 # create cluster + install substrate
#   hack/install-substrate-kind.sh --no-cluster    # install onto the current kube-context
#   hack/install-substrate-kind.sh --delete        # delete the kind cluster + registry
#
# Environment overrides:
#   KIND_CLUSTER_NAME    kind cluster name              (default: kmcp-substrate)
#   KIND_NODE_IMAGE      kindest/node image             (default: v1.35.0 pin)
#   SUBSTRATE_VERSION    published chart/image version  (default: 0.0.6)
#   SUBSTRATE_CHART_REPO OCI repo holding the charts     (default: oci://ghcr.io/kagent-dev/substrate/helm)
#   ATE_NAMESPACE        control-plane namespace         (default: ate-system)
#   REGISTRY_PORT        host port for the local registry (default: 5001)

set -o errexit -o nounset -o pipefail

# Substrate's mTLS install path needs the PodCertificateRequest (v1alpha1) API,
# which is only served by Kubernetes >= 1.35. kind's default node image lags,
# so pin a known-good v1.35.0 image (override with KIND_NODE_IMAGE).
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kmcp-substrate}"
KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.35.0@sha256:4613778f3cfcd10e615029370f5786704559103cf27bef934597ba562b269661}"
SUBSTRATE_VERSION="${SUBSTRATE_VERSION:-0.0.6}"
SUBSTRATE_CHART_REPO="${SUBSTRATE_CHART_REPO:-oci://ghcr.io/kagent-dev/substrate/helm}"
ATE_NAMESPACE="${ATE_NAMESPACE:-ate-system}"
PODCERT_NAMESPACE="podcertificate-controller-system"
REGISTRY_NAME="kind-registry"
REGISTRY_PORT="${REGISTRY_PORT:-5001}"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mWARN:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

require() {
  for bin in "$@"; do
    command -v "$bin" >/dev/null 2>&1 || die "required tool not found on PATH: $bin"
  done
}

delete_cluster() {
  log "Deleting kind cluster '${KIND_CLUSTER_NAME}'..."
  kind delete cluster --name "${KIND_CLUSTER_NAME}" || true
  if [ "$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null || true)" = "true" ]; then
    log "Removing local registry '${REGISTRY_NAME}'..."
    docker rm -f "${REGISTRY_NAME}" || true
  fi
  log "Done."
}

create_registry() {
  log "Ensuring local docker registry '${REGISTRY_NAME}' on port ${REGISTRY_PORT}..."
  if [ "$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null || true)" != "true" ]; then
    docker run -d --restart=always \
      -p "127.0.0.1:${REGISTRY_PORT}:5000" \
      --network bridge --name "${REGISTRY_NAME}" \
      registry:3 >/dev/null
  fi
}

create_cluster() {
  if kind get clusters 2>/dev/null | grep -qx "${KIND_CLUSTER_NAME}"; then
    log "kind cluster '${KIND_CLUSTER_NAME}' already exists; reusing it."
  else
    log "Creating kind cluster '${KIND_CLUSTER_NAME}' with certificate feature gates..."
    # mTLS install path (the one kmcp's ate-api client is validated against)
    # depends on the off-by-default ClusterTrustBundle / PodCertificateRequest
    # gates and the v1beta1 certificates API.
    kind create cluster --name "${KIND_CLUSTER_NAME}" --config=- <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: ${KIND_NODE_IMAGE}
featureGates:
  ClusterTrustBundle: true
  ClusterTrustBundleProjection: true
  PodCertificateRequest: true
runtimeConfig:
  "certificates.k8s.io/v1beta1": "true"
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
EOF
  fi

  # Proxy-ARP on every node for gVisor loopback pod-to-pod networking.
  log "Enabling proxy-ARP on kind nodes..."
  for node in $(kind get nodes --name "${KIND_CLUSTER_NAME}"); do
    docker exec "${node}" sysctl -w net.ipv4.conf.all.proxy_arp=1 >/dev/null
  done

  # Wire the local registry into the nodes + cluster network so kind load works.
  log "Wiring local registry into the cluster..."
  local reg_dir="/etc/containerd/certs.d/localhost:${REGISTRY_PORT}"
  for node in $(kind get nodes --name "${KIND_CLUSTER_NAME}"); do
    docker exec "${node}" mkdir -p "${reg_dir}"
    echo "[host.\"http://${REGISTRY_NAME}:5000\"]" \
      | docker exec -i "${node}" cp /dev/stdin "${reg_dir}/hosts.toml"
  done
  if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REGISTRY_NAME}")" = "null" ]; then
    docker network connect "kind" "${REGISTRY_NAME}" || true
  fi
}

# --- mTLS CA/JWT pool bootstrap -------------------------------------------
# In mtls mode the chart does NOT create the CA/JWT signing pools — without
# them the podcertificate-controller can't start, so no PodCertificateRequest
# is ever signed and valkey/ate-api/atenet-router hang on credential mounts.
# The fork's installer creates them with `kubectl ate admin make-*-pool`; we
# prefer that tool when present and otherwise generate equivalent pools with
# openssl (the pool wire format accepts PEM key material).

apply_pool_secret() { # $1=name $2=namespace $3=json
  kubectl create secret generic "$1" -n "$2" --from-literal=pool="$3" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
}

make_ca_pool() { # $1=name $2=namespace
  if kubectl get secret -n "$2" "$1" >/dev/null 2>&1; then return; fi
  if command -v kubectl-ate >/dev/null 2>&1; then
    kubectl-ate admin make-ca-pool --ca-id=1 --name="$1" --secret-namespace="$2" >/dev/null
    return
  fi
  local dir key cert json
  dir="$(mktemp -d)"
  openssl ecparam -name prime256v1 -genkey -noout -out "${dir}/key.pem" 2>/dev/null
  openssl req -new -x509 -key "${dir}/key.pem" -days 3650 -subj "/CN=ate-$1" -out "${dir}/cert.pem" 2>/dev/null
  json="$(jq -cn --rawfile k "${dir}/key.pem" --rawfile c "${dir}/cert.pem" \
    '{CAs:[{ID:"1",SigningKeyPEM:$k,RootCertificatePEM:$c}]}')"
  rm -rf "${dir}"
  apply_pool_secret "$1" "$2" "${json}"
}

make_jwt_pool() { # $1=name $2=namespace
  if kubectl get secret -n "$2" "$1" >/dev/null 2>&1; then return; fi
  if command -v kubectl-ate >/dev/null 2>&1; then
    kubectl-ate admin make-jwt-pool --key-id=1 --name="$1" --secret-namespace="$2" >/dev/null
    return
  fi
  local dir json
  dir="$(mktemp -d)"
  openssl ecparam -name prime256v1 -genkey -noout -out "${dir}/key.pem" 2>/dev/null
  json="$(jq -cn --rawfile k "${dir}/key.pem" \
    '{Authorities:[{ID:"1",Algorithm:"ES256",SigningKeyPEM:$k}]}')"
  rm -rf "${dir}"
  apply_pool_secret "$1" "$2" "${json}"
}

bootstrap_mtls_pools() {
  log "Bootstrapping mTLS CA/JWT pools ($(command -v kubectl-ate >/dev/null 2>&1 && echo kubectl-ate || echo openssl))..."
  # Namespaces are created by the control-plane chart (podcertificate-controller-system
  # is a chart-owned template), so this runs after the chart install.
  make_jwt_pool session-id-jwt-pool   "${ATE_NAMESPACE}"
  make_ca_pool  session-id-ca-pool    "${ATE_NAMESPACE}"
  make_ca_pool  service-dns-ca-pool   "${PODCERT_NAMESPACE}"
  make_ca_pool  pod-identity-ca-pool  "${PODCERT_NAMESPACE}"

  # valkey verifies client certs against the service-dns CA root.
  if ! kubectl get secret -n "${ATE_NAMESPACE}" valkey-ca-certs >/dev/null 2>&1; then
    local pool der pem
    pool="$(kubectl get secret -n "${PODCERT_NAMESPACE}" service-dns-ca-pool -o jsonpath='{.data.pool}' | base64 --decode)"
    pem="$(echo "${pool}" | jq -r '.CAs[0].RootCertificatePEM // empty')"
    if [ -z "${pem}" ]; then
      der="$(echo "${pool}" | jq -r '.CAs[0].RootCertificateDER')"
      pem="$(echo "${der}" | base64 --decode | openssl x509 -inform der -outform pem)"
    fi
    kubectl create secret generic valkey-ca-certs --from-literal=ca.crt="${pem}" \
      -n "${ATE_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  fi
}

install_substrate() {
  log "Installing substrate CRDs (chart ${SUBSTRATE_VERSION})..."
  helm upgrade --install substrate-crds "${SUBSTRATE_CHART_REPO}/substrate-crds" \
    --version "${SUBSTRATE_VERSION}" --wait

  # Release name MUST be 'substrate': the chart's fullname helper only emits the
  # bare service name 'api' (which kmcp dials as dns:///api.ate-system.svc:443)
  # when the release name equals the chart name. redis.clientCert /
  # redis.tlsServerName are required in mtls mode so ate-api presents a client
  # cert to valkey (which demands one); the chart leaves them empty by default.
  # The chart creates the ate-system and podcertificate-controller-system
  # namespaces, so it must run before the pool bootstrap writes secrets there.
  log "Installing substrate control plane (chart ${SUBSTRATE_VERSION}, auth.mode=mtls)..."
  helm upgrade --install substrate "${SUBSTRATE_CHART_REPO}/substrate" \
    --version "${SUBSTRATE_VERSION}" \
    --namespace "${ATE_NAMESPACE}" --create-namespace \
    --set auth.mode=mtls \
    --set redis.clientCert=/run/servicedns.podcert.ate.dev/credential-bundle.pem \
    --set redis.tlsServerName=valkey-cluster."${ATE_NAMESPACE}".svc

  # Pods above stay Pending on credential mounts until the CA/JWT pools exist.
  bootstrap_mtls_pools

  # The podcert controller may have been scheduled before its CA pools existed;
  # restart it so it picks them up and starts signing without the long mount
  # back-off, then everything else unblocks.
  kubectl rollout restart -n "${PODCERT_NAMESPACE}" deployment/podcertificate-controller >/dev/null 2>&1 || true

  log "Waiting for the control plane to become ready..."
  kubectl rollout status -n "${PODCERT_NAMESPACE}" deployment/podcertificate-controller --timeout=5m || \
    warn "podcertificate-controller not ready — mTLS certs cannot be issued; check the CA pool secrets."
  kubectl rollout status -n "${ATE_NAMESPACE}" statefulset/valkey-cluster --timeout=5m
  kubectl wait --for=condition=complete -n "${ATE_NAMESPACE}" job/valkey-cluster-init --timeout=5m
  kubectl wait --for=condition=complete -n "${ATE_NAMESPACE}" job/rustfs-bucket-init --timeout=5m
  # ate-api may have crash-looped while valkey/certs came up; nudge it.
  kubectl rollout restart -n "${ATE_NAMESPACE}" deployment/ate-api-server-deployment >/dev/null 2>&1 || true
  for d in ate-api-server-deployment ate-controller atenet-router dns rustfs; do
    kubectl rollout status -n "${ATE_NAMESPACE}" "deployment/${d}" --timeout=5m
  done
  kubectl rollout status -n "${ATE_NAMESPACE}" daemonset/atelet --timeout=5m
}

print_next_steps() {
  cat <<EOF

$(log "Substrate ${SUBSTRATE_VERSION} is installed on kind cluster '${KIND_CLUSTER_NAME}'.")

Next steps (see SUBSTRATE_TESTING.md for the full walk-through):

  1. Create a WorkerPool. NOTE: the published worker image
     (ghcr.io/kagent-dev/substrate/ateom-gvisor:v${SUBSTRATE_VERSION}) predates the
     veth-networking rewrite and fails gVisor restore (scale-from-zero) with
     'PACKET_FANOUT: invalid argument'. Until a release newer than
     v${SUBSTRATE_VERSION} ships it, build the worker image from fork HEAD and push it to
     the local registry. The veth rewrite is worker-only, so ONLY the worker
     needs a fork build — the published control plane installed above is fine.
     Restore also needs a recent Docker Desktop VM kernel: it failed on
     6.10.14-linuxkit even with the fork worker and was fixed by 6.12.76-linuxkit
     (check 'docker run --rm alpine uname -r').

       KO_DOCKER_REPO=localhost:${REGISTRY_PORT}/ateom-gvisor ko build --bare \\
         --tags head --platform linux/\$(go env GOARCH) ./cmd/ateom-gvisor   # in the fork repo

       apiVersion: ate.dev/v1alpha1
       kind: WorkerPool
       metadata: {name: mcp-pool, namespace: kmcp-test}
       spec:
         replicas: 3
         ateomImage: localhost:${REGISTRY_PORT}/ateom-gvisor:head

  2. Build + load the kmcp controller image and install kmcp with substrate enabled:

       make docker-build && kind load docker-image <controller-image> --name ${KIND_CLUSTER_NAME}
       helm install kmcp-crds helm/kmcp-crds
       helm install kmcp helm/kmcp -n kmcp-system --create-namespace \\
         --set substrate.enabled=true \\
         --set substrate.ateApiEndpoint=dns:///api.${ATE_NAMESPACE}.svc:443 \\
         --set substrate.ateApiInsecure=true \\
         --set substrate.defaultWorkerPool.namespace=kmcp-test \\
         --set substrate.defaultWorkerPool.name=mcp-pool \\
         --set substrate.snapshots.locationPrefix=s3://ate-snapshots \\
         --set substrate.runsc.amd64.url=gs://gvisor/releases/nightly/2026-05-19/x86_64/runsc \\
         --set substrate.runsc.amd64.sha256=a397be1abc2420d26bce6c70e6e2ff96c73aaaab929756c56f5e2089ea842b63 \\
         --set substrate.runsc.arm64.url=gs://gvisor/releases/nightly/2026-05-19/aarch64/runsc \\
         --set substrate.runsc.arm64.sha256=1ba2366ae2efceba166046f51a4104f9261c9cb72c6db8f5b3fe2dc57dea86b9 \\
         --set substrate.adapterBinary.amd64.url=https://github.com/agentgateway/agentgateway/releases/download/v0.9.0/agentgateway-linux-amd64 \\
         --set substrate.adapterBinary.amd64.sha256=576375f1588b2cbe1743ecf2a03c7d22e2a46dfc594325fd53e98b706dbbd05d \\
         --set substrate.adapterBinary.arm64.url=https://github.com/agentgateway/agentgateway/releases/download/v0.9.0/agentgateway-linux-arm64 \\
         --set substrate.adapterBinary.arm64.sha256=43c350adf189d1c5f9e9a702450ec545178587d3900ea9a75bad647024404166

     IMPORTANT: the substrate.adapterBinary.* values are required. The actor's
     in-sandbox agentgateway adapter (bound to :80) is downloaded by the actor
     bootstrap script from this URL. If it is unset and the workload image does
     not bundle agentgateway at /usr/local/bin/agentgateway, the adapter exits
     during the golden bake; the checkpoint then captures only the 'pause'
     container, and every scale-from-zero restore fails with
     'inconsistent private memory files on restore: savedMFOwners = [pause:/]'.
EOF
}

main() {
  case "${1:-}" in
    --delete)
      require docker kind
      delete_cluster
      exit 0
      ;;
    --no-cluster)
      require kubectl helm jq openssl
      install_substrate
      print_next_steps
      ;;
    "" )
      require docker kind kubectl helm jq openssl
      create_registry
      create_cluster
      install_substrate
      print_next_steps
      ;;
    -h|--help)
      sed -n '17,36p' "$0"
      ;;
    *)
      die "unknown argument: $1 (try --help)"
      ;;
  esac
}

main "$@"
