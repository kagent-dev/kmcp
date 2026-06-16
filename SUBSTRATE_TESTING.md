# Substrate runtime — manual end-to-end testing

This is the manual end-to-end test procedure for kmcp's substrate (serverless)
runtime, plus the recorded validation results. It complements the automated
suite in `test/e2e/substrate_e2e_test.go` (gated by `KMCP_SUBSTRATE_E2E=true`)
and the design/implementation notes in `SUBSTRATE_PLAN.md` and
`devel/design/substrate-runtime.md`.

Substrate runs fully on kind — no GKE/gVisor nodes needed: runsc is downloaded
by substrate's atelet at runtime, and actor snapshots go to an in-cluster
S3-compatible rustfs.

## 1. Cluster + substrate install

The substrate control plane and the gVisor worker image install entirely from
published release artifacts (`ghcr.io/kagent-dev/substrate`), so **no
kagent-dev/substrate checkout is required**:

```bash
cd /Users/jm/Codebase/kmcp
hack/install-substrate-kind.sh
```

The script:

- creates a kind cluster (`kmcp-substrate`) pinned to a **v1.35.0** node image
  (kind's default lags and can't serve the `PodCertificateRequest` API mtls
  needs) with the certificate feature gates (`ClusterTrustBundle`,
  `ClusterTrustBundleProjection`, `PodCertificateRequest`) + the
  `certificates.k8s.io/v1beta1` API, enables proxy-ARP for gVisor loopback pod
  networking, and wires a local registry on `localhost:5001`;
- installs the substrate CRDs and control plane from
  `oci://ghcr.io/kagent-dev/substrate/helm` at `v0.0.6` with `auth.mode=mtls`
  (the mode kmcp's ate-api client is validated against — it dials with TLS and
  `InsecureSkipVerify`);
- **bootstraps the mtls signing material the chart does not create**: the
  `session-id-jwt-pool`, `session-id-ca-pool`, `service-dns-ca-pool`,
  `pod-identity-ca-pool`, and derived `valkey-ca-certs` secrets, plus the
  `redis.clientCert`/`redis.tlsServerName` values ate-api needs to reach valkey.
  It prefers `kubectl ate admin make-*-pool` when that CLI is present and
  otherwise generates equivalent pools with `openssl` (no fork checkout needed);
- waits for valkey, the rustfs bucket init, ate-api, ate-controller,
  atenet-router/dns, atelet, and the pod-certificate controller.

Override the cluster name / node image / version with `KIND_CLUSTER_NAME` /
`KIND_NODE_IMAGE` / `SUBSTRATE_VERSION`; install onto an existing context with
`--no-cluster`; tear the cluster down with `--delete`.

Verify:

```bash
kubectl get pods -n ate-system   # api-server, controller, atelet, router, dns, valkey x6, rustfs all Ready
```

## 2. Test namespace resources (WorkerPool + secret)

> **Worker image caveat (the only fork build needed).** The published
> `ateom-gvisor:v0.0.6` image predates substrate's veth-networking rewrite and
> fails gVisor restore (scale-from-zero) with `PACKET_FANOUT: invalid argument`.
> Until a release newer than v0.0.6 ships it, build the worker image from fork
> HEAD into the local registry and point the WorkerPool at that tag instead:
>
> ```bash
> # in the kagent-dev/substrate checkout
> KO_DOCKER_REPO=localhost:5001/ateom-gvisor ko build --bare \
>   --tags head --platform linux/$(go env GOARCH) ./cmd/ateom-gvisor
> ```
>
> The veth rewrite is worker-only (`cmd/ateom-gvisor` + `internal/serverboot`),
> so **only the worker needs a fork build** — the published substrate control
> plane (router/api/controller) works as-is; do not rebuild it.
>
> **Host-kernel requirement.** `PACKET_FANOUT` restore also depends on the
> Docker Desktop VM kernel: it failed on `6.10.14-linuxkit` even with the
> fork-HEAD worker, and was fixed by updating Docker Desktop to
> `6.12.76-linuxkit`. If restore fails with `PACKET_FANOUT: invalid argument`
> on the fork worker, update Docker Desktop (check `docker run --rm alpine
> uname -r`).

```yaml
apiVersion: ate.dev/v1alpha1
kind: WorkerPool
metadata: {name: mcp-pool, namespace: kmcp-test}
spec:
  replicas: 3
  ateomImage: localhost:5001/ateom-gvisor:head   # see caveat above; v0.0.6 published image fails restore
---
apiVersion: v1
kind: Secret
metadata: {name: mcp-test-secret, namespace: kmcp-test}
stringData: {API_KEY: dummy}   # exercises SecretRefs → secretKeyRef expansion
```

```bash
kubectl create namespace kmcp-test
kubectl apply -f <the manifest above>
```

## 3. Install kmcp with substrate enabled

```bash
cd /Users/jm/Codebase/kmcp
make docker-build && kind load docker-image <controller-image> --name kmcp-substrate
helm install kmcp-crds helm/kmcp-crds
helm install kmcp helm/kmcp -n kmcp-system --create-namespace \
  --set substrate.enabled=true \
  --set substrate.ateApiEndpoint=dns:///api.ate-system.svc:443 \
  --set substrate.ateApiInsecure=true \   # ate-api uses podcertificate-signed TLS; skip verify on kind
  --set substrate.defaultWorkerPool.namespace=kmcp-test \
  --set substrate.defaultWorkerPool.name=mcp-pool \
  --set substrate.snapshots.locationPrefix=s3://ate-snapshots \
  --set "substrate.runsc.amd64.url=gs://gvisor/releases/nightly/2026-05-19/x86_64/runsc" \
  --set "substrate.runsc.amd64.sha256=a397be1abc2420d26bce6c70e6e2ff96c73aaaab929756c56f5e2089ea842b63" \
  --set "substrate.runsc.arm64.url=gs://gvisor/releases/nightly/2026-05-19/aarch64/runsc" \
  --set "substrate.runsc.arm64.sha256=1ba2366ae2efceba166046f51a4104f9261c9cb72c6db8f5b3fe2dc57dea86b9" \
  --set "substrate.adapterBinary.amd64.url=https://github.com/agentgateway/agentgateway/releases/download/v0.9.0/agentgateway-linux-amd64" \
  --set "substrate.adapterBinary.amd64.sha256=576375f1588b2cbe1743ecf2a03c7d22e2a46dfc594325fd53e98b706dbbd05d" \
  --set "substrate.adapterBinary.arm64.url=https://github.com/agentgateway/agentgateway/releases/download/v0.9.0/agentgateway-linux-arm64" \
  --set "substrate.adapterBinary.arm64.sha256=43c350adf189d1c5f9e9a702450ec545178587d3900ea9a75bad647024404166" \
  --set "substrate.envSourceNamespaces={kmcp-test}"
```

(runsc URL/SHA values are gVisor nightly builds; refresh as needed. The chart
defaults `ateApiEndpoint` to `dns:///api.ate-system.svc:443`, matching the
published chart's `api` Service.)

> **The `substrate.adapterBinary.*` values are required for scale-from-zero.**
> The actor's in-sandbox agentgateway adapter (bound to `:80`) is downloaded by
> the actor bootstrap script from this URL. If it is unset — and the workload
> image does not already bundle `agentgateway` at `/usr/local/bin/agentgateway`
> — the adapter exits immediately during the golden bake. The checkpoint then
> captures only the `pause` container, and every later restore fails with
> `inconsistent private memory files on restore: savedMFOwners = [pause:/]`.
> The condition ladder still reaches Ready (the bake's `RunWorkload` succeeds),
> so the misconfiguration only surfaces on the first scale-from-zero request.

## 4. Deploy test MCPServers

HTTP (primary case — streamable HTTP server, digest-pinned image):

```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata: {name: everything-http, namespace: kmcp-test}
spec:
  runtime: substrate
  substrate: {}                      # default WorkerPool + snapshots
  transportType: http
  httpTransport: {targetPort: 3001, path: /mcp}
  deployment:
    image: <streamable-http MCP server image>@sha256:<digest>
    cmd: <server-cmd>
    args: [...]
    env: {PORT: "3001"}
    secretRefs: [{name: mcp-test-secret}]
```

stdio (parity case):

```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata: {name: everything-stdio, namespace: kmcp-test}
spec:
  runtime: substrate
  substrate: {}
  transportType: stdio
  deployment:
    image: node:24-alpine3.21@sha256:<digest>   # must be pinned; has sh/wget for bootstrap
    cmd: npx
    args: ["-y", "@modelcontextprotocol/server-everything"]
```

## 5. Validation checklist

The steps below use the `kmcp-test`/`everything-http` names from §2–4. Some
recorded runs were executed against the kagent-embedded instance instead
(MCPServer `everything-substrate` in the `kagent` namespace, actor ID
`mcp-kagent-everything-substrate`) — substitute namespace and names accordingly;
the procedure is identical.

1. **Conditions ladder**: `kubectl get mcpserver -n kmcp-test -o yaml` — watch Accepted → ResolvedRefs → ActorTemplateReady (golden snapshot bakes; first time takes minutes: runsc + agentgateway download) → ActorReady → Programmed → Ready.
2. **Generated resources**: ActorTemplate exists with owner ref, expanded secretKeyRef env, `/bin/sh -c` bootstrap command; `kubectl ate get actors` shows `mcp-kmcp-test-everything-http` (plus the template's golden actor). In managed-proxy mode (the default): a selector-less Service named after the MCPServer, an EndpointSlice `<name>-proxy` carrying the shared proxy pod IPs on port 8080, and a route block (`name: kmcp-test/everything-http`) in the shared `kmcp-substrate-ingress-proxy` ConfigMap in kmcp's namespace.
3. **Status contract**:
```bash
kubectl get mcpserver -n kmcp-test everything-http -o jsonpath='{.status.substrate}' | python3 -m json.tool
```
   All six fields must be populated and follow these shapes (verified values from the
   live `kagent/everything-substrate` instance, where the actor ID is `mcp-<ns>-<name>`):
```json
{
    "actorID": "mcp-kagent-everything-substrate",
    "actorHost": "mcp-kagent-everything-substrate.actors.resources.substrate.ate.dev",
    "actorTemplateRef": "everything-substrate",
    "mcpPath": "/mcp",
    "proxyEndpoint": "http://everything-substrate.kagent.svc:3000/mcp",
    "routerURL": "http://atenet-router.ate-system.svc:80"
}
```
   `proxyEndpoint` is present only in managed-proxy mode; `actorHost` + `routerURL` +
   `mcpPath` are the proxy-less contract an embedding platform (kagent) consumes — step 6
   uses them verbatim.
4. **Streamable HTTP through the shared proxy** (the headline test). The
   per-MCPServer Service has no selector (it targets the shared proxy via an
   EndpointSlice), so `kubectl port-forward svc/<name>` does not work —
   port-forward the shared proxy Deployment instead and pin the Host header
   (the proxy routes by hostname; in-cluster clients just use the Service
   URL, validated separately below):
```bash
kubectl port-forward -n kmcp-system deploy/kmcp-substrate-ingress-proxy 8080:8080 &
# new routes can take up to ~1 min to reach the proxy (kubelet ConfigMap sync)
```
   Initialize a session and note the `mcp-session-id` response header:
```bash
curl -si http://localhost:8080/mcp -X POST -H 'Host: everything-http.kmcp-test.svc' \
  -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}' \
  | grep -i '^mcp-session-id'
export SESSION_ID=<value from above>
```
   Complete the handshake:
```bash
curl -s http://localhost:8080/mcp -X POST -H 'Host: everything-http.kmcp-test.svc' \
  -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' \
  -H "mcp-session-id: $SESSION_ID" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'
```
   List tools:
```bash
curl -s http://localhost:8080/mcp -X POST -H 'Host: everything-http.kmcp-test.svc' \
  -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' \
  -H "mcp-session-id: $SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
```
   Then prove the real in-cluster Service path (Service → EndpointSlice →
   proxy) with a one-shot pod:
```bash
kubectl run curl-test -n kmcp-test --rm --attach --restart=Never --image=curlimages/curl:8.7.1 --command -- \
  curl -s -o /dev/null -w '%{http_code}' --max-time 120 -X POST http://everything-http.kmcp-test.svc:3000/mcp \
  -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}'
# expect 200
```
   Expect an SSE `data:` line containing the everything-server tool list (`echo`,
   `add`, ...). If the actor was suspended, the initialize stalls a few seconds
   while the router resumes it — that is scale-from-zero working, and
   `kubectl ate get actor <id>` flips SUSPENDED → RUNNING.
5. **Scale-from-zero**: `kubectl ate suspend actor mcp-kmcp-test-everything-http`; verify status shows SUSPENDED but MCPServer stays Ready; rerun the tools/list call — first request buffers while the actor resumes, then succeeds. Repeat a `tools/call` to confirm session state survives where applicable.
   Then soak the suspend path (this is what catches the ateom zombie/reaper regression — unpatched it wedges within a few cycles on a multi-process actor). Run this cycle 6 times:
```bash
kubectl ate resume actor mcp-kmcp-test-everything-http
kubectl ate get actor mcp-kmcp-test-everything-http     # repeat until STATUS_RUNNING
kubectl ate suspend actor mcp-kmcp-test-everything-http
kubectl ate get actor mcp-kmcp-test-everything-http     # repeat until STATUS_SUSPENDED
```
   Then check the worker logs:
```bash
kubectl logs -n kmcp-test -l ate.dev/worker-pool=mcp-pool --since=20m | grep -E "CheckpointWorkload|WARN"
```
   Pass criteria: every suspend reaches SUSPENDED on the first try and the logs show
   one sub-second `CheckpointWorkload` per cycle. On the published worker image an
   occasional `WARN Failed to clean up runsc containers after checkpoint ...
   exit status 128` is expected and benign — the fork downgraded the old
   permanent SUSPENDING wedge to a best-effort cleanup warning; the suspend
   itself still lands.
6. **Proxy-less path** (kagent-embedding contract). The `status.substrate`
   contract works in every mode and can be tested directly against the
   router; to additionally verify mode-switch cleanup, flip the installation
   to `--set substrate.ingress.mode=none` (proxy Deployment, per-server
   Services/EndpointSlices, and `proxyEndpoint` all disappear; servers stay
   Ready) and back (everything is recreated). Contract test:
```bash
kubectl port-forward -n ate-system svc/atenet-router 8000:80 &
curl -H "Host: <status.substrate.actorHost>" http://localhost:8000/mcp -X POST \
  -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}'
```
7. **stdio parity**: repeat 4–5 against `everything-stdio`.
8. **Validation rejections**: apply an MCPServer with an unpinned image (CEL admission error) and one with `volumes` set (Accepted=False/UnsupportedField).
9. **Deletion**: `kubectl delete mcpserver everything-http` — actor deleted via ate-api (incl. golden actor), ActorTemplate and ingress objects GC'd, finalizer removed; `kubectl ate get actors` empty.
10. **Regression**: deploy a `runtime: kubernetes` (default) MCPServer and confirm behavior is unchanged.

## Validation results

Recorded results from running this procedure live in
`SUBSTRATE_TESTING_VALIDATION.md` (newest first).
