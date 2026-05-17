# go-httpbin

`go-httpbin` is a Go + Chi port of common `httpbin` behavior for testing HTTP clients, proxies, service meshes, and Kubernetes workloads.

The module path is:

```text
github.com/vancanhuit/go-httpbin
```

## Development

Run the test suite:

```sh
npm ci
go generate ./...
golangci-lint fmt
golangci-lint run
go test -v -cover -race ./...
docker build -t go-httpbin:dev .
```

Install Git hooks:

```sh
npm run prepare
```

Husky runs Commitlint on local commit messages. CI validates pull request titles with the same conventional commit rules.

Run the server locally:

```sh
go run ./cmd/server
```

The default address is `:8080`.
Swagger UI is available at `http://localhost:8080/docs`. The OpenAPI document is served at `http://localhost:8080/openapi.yaml`.
Application logs are written to stdout as structured JSON with `log/slog`.

## Container

Build and run the container image:

```sh
docker build -t go-httpbin:dev .
docker run --rm -p 8080:8080 go-httpbin:dev
```

Inject a build version into `/version`:

```sh
docker build --build-arg VERSION=dev-local -t go-httpbin:dev .
curl http://127.0.0.1:8080/version
```

Build a multi-platform image archive:

```sh
docker buildx build --platform linux/amd64,linux/arm64 -t go-httpbin:dev --output=type=oci,dest=/tmp/go-httpbin-image.tar .
```

Pull request builds inject `pr-<number>-<short-sha>` into `/version` and archive the OCI tarball as the `go-httpbin-oci-image` workflow artifact.

Pushes to `main` publish multi-platform images to GHCR as:

```text
ghcr.io/vancanhuit/go-httpbin:latest
ghcr.io/vancanhuit/go-httpbin:main
ghcr.io/vancanhuit/go-httpbin:<git-sha>
ghcr.io/vancanhuit/go-httpbin:main-<short-sha>
```

Main branch images inject `main-<short-sha>` into `/version`.

The build stage uses the Go Debian trixie image. The runtime image uses the Debian 13 distroless nonroot base, and keeps configuration in environment variables.

## Release

Push a SemVer tag to create a GitHub Release with `git-cliff` release notes and publish a multi-platform GHCR image:

```sh
git tag -a v0.1.0 -m v0.1.0
git push origin v0.1.0
```

Release images are tagged with the pushed tag, the SemVer version without the `v` prefix, and the git SHA. Stable releases also update `latest`, `v<major>`, and `v<major>.<minor>`. The pushed tag is injected into release binaries and images and is available from `/version`.

Each release also attaches `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, and `windows/arm64` binary archives with per-archive SHA-256 files plus a combined `checksums.txt`.

## Kubernetes

Apply the sample manifest:

```sh
kubectl apply -f deploy/kubernetes/go-httpbin.yaml
kubectl -n go-httpbin rollout status deployment/go-httpbin
kubectl -n go-httpbin port-forward service/go-httpbin 8080:80
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/version
```

The manifest deploys `ghcr.io/vancanhuit/go-httpbin:latest` with readiness and liveness probes, a `ClusterIP` service, non-root security settings, and environment-based configuration. Pull request CI verifies it in a kind cluster.

Gateway API resources for kind are available in `deploy/kubernetes/gateway.yaml`. They target the `cloud-provider-kind` GatewayClass and route `go-httpbin.local` to the `go-httpbin` service.

## OpenAPI code generation

Routes are described in `api/openapi.yaml`, and Chi server glue is generated with `oapi-codegen`.

```sh
go generate ./...
```

The generator is pinned as a Go tool dependency in `go.mod`.

## Configuration

Configuration follows 12-factor principles and is loaded from environment variables with `github.com/caarlos0/env`.

| Variable | Default | Description |
| --- | --- | --- |
| `ADDR` | `:8080` | HTTP listen address |
| `READ_TIMEOUT` | `5s` | HTTP server read timeout |
| `WRITE_TIMEOUT` | `30s` | HTTP server write timeout |
| `IDLE_TIMEOUT` | `120s` | HTTP server idle timeout |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `HANDLER_TIMEOUT` | `30s` | Per-request handler timeout |
| `MAX_BODY_BYTES` | `10MiB` | Maximum request body size for echo endpoints |
| `MAX_DELAY` | `10s` | Maximum delay/drip duration |
| `MAX_STREAM_ITEMS` | `100` | Maximum JSON stream records |
| `MAX_RANDOM_BYTES` | `1048576` | Maximum random byte response size |
| `TRUST_PROXY_HEADERS` | `false` | Trust forwarded headers for origin and URL reporting |
| `ENABLE_COMPRESSION` | `true` | Enable Chi response compression middleware |
| `LOG_LEVEL` | `info` | JSON log level: `debug`, `info`, `warn`, or `error` |
| `LOG_ADD_SOURCE` | `false` | Include source file and line in JSON logs |

## Implemented endpoint groups

- Request echo: `/get`, `/post`, `/put`, `/patch`, `/delete`, `/anything`, `/anything/*`, including `TRACE` on `/anything`
- Request inspection: `/headers`, `/ip`, `/user-agent`
- Response behavior: `/status/{codes}`, `/response-headers`, `/cache`, `/cache/{seconds}`, `/etag/{etag}`, `/range/{numbytes}`
- Auth: `/basic-auth/{user}/{password}`, `/hidden-basic-auth/{user}/{password}`, `/digest-auth/{qop}/{user}/{password}`, `/bearer`
- Dynamic data: `/uuid`, `/bytes/{n}`, `/stream-bytes/{n}`, `/stream/{n}`, `/delay/{delay}`, `/drip`, `/base64/{value}`
- Cookies and redirects: `/cookies`, `/cookies/set`, `/cookies/set/{name}/{value}`, `/cookies/delete`, `/redirect/{n}`, `/relative-redirect/{n}`, `/absolute-redirect/{n}`, `/redirect-to`
- Formats and media: `/gzip`, `/deflate`, `/brotli`, `/json`, `/xml`, `/html`, `/robots.txt`, `/deny`, `/encoding/utf8`, `/image`, `/image/png`, `/image/jpeg`, `/image/gif`, `/image/svg`, `/image/webp`, `/links/{n}/{offset}`
- Operations: `/healthz`, `/readyz`, `/version`
- API docs: `/openapi.yaml`, `/docs`

## Compatibility notes

This project targets practical parity with `httpbin`, not byte-for-byte cloning. WebP responses use a small embedded fixture image, and Digest auth uses stateless nonce validation for Kubernetes-safe test deployments.
