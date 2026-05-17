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

## Implemented endpoint groups

- Request echo: `/get`, `/post`, `/put`, `/patch`, `/delete`, `/anything`, `/anything/*`
- Request inspection: `/headers`, `/ip`, `/user-agent`
- Response behavior: `/status/{codes}`, `/response-headers`, `/cache`, `/cache/{seconds}`, `/etag/{etag}`
- Auth: `/basic-auth/{user}/{password}`, `/hidden-basic-auth/{user}/{password}`, `/bearer`
- Dynamic data: `/uuid`, `/bytes/{n}`, `/stream-bytes/{n}`, `/stream/{n}`, `/delay/{delay}`, `/drip`, `/base64/{value}`
- Cookies and redirects: `/cookies`, `/cookies/set`, `/cookies/set/{name}/{value}`, `/cookies/delete`, `/redirect/{n}`, `/relative-redirect/{n}`, `/absolute-redirect/{n}`, `/redirect-to`
- Formats and media: `/gzip`, `/deflate`, `/json`, `/xml`, `/html`, `/robots.txt`, `/deny`, `/encoding/utf8`, `/image`, `/image/png`, `/image/jpeg`, `/image/gif`, `/image/svg`, `/links/{n}/{offset}`
- Operations: `/healthz`, `/readyz`
- API docs: `/openapi.yaml`, `/docs`

## Compatibility notes

This project targets practical parity with `httpbin`, not byte-for-byte cloning. Digest auth and WebP image generation are intentionally deferred to keep the first version small, dependency-light, and Kubernetes-safe.
