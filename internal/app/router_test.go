package app

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vancanhuit/go-httpbin/internal/config"
	buildinfo "github.com/vancanhuit/go-httpbin/internal/version"
)

func testRouter() http.Handler {
	return NewRouter(config.Config{
		Addr:              ":8080",
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
		HandlerTimeout:    time.Second,
		MaxBodyBytes:      1024,
		MaxDelay:          100 * time.Millisecond,
		MaxStreamItems:    10,
		MaxRandomBytes:    128,
		TrustProxyHeaders: true,
		EnableCompression: false,
	})
}

func TestHealthEndpoints(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz"} {
		rec := request(t, http.MethodGet, path, nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("%s Content-Type = %q, want application/json", path, got)
		}
		var body map[string]string
		decodeJSON(t, rec, &body)
		if body["status"] != "ok" {
			t.Fatalf("%s status body = %#v, want ok", path, body)
		}
	}
}

func TestVersionEndpoint(t *testing.T) {
	oldVersion := buildinfo.Version
	buildinfo.Version = "v9.8.7-test"
	t.Cleanup(func() {
		buildinfo.Version = oldVersion
	})

	rec := request(t, http.MethodGet, "/version", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]string
	decodeJSON(t, rec, &body)
	if body["version"] != "v9.8.7-test" {
		t.Fatalf("version = %q, want injected version", body["version"])
	}
}

func TestDefaultErrorsAreJSON(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/missing", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		var body map[string]any
		decodeJSON(t, rec, &body)
		if body["error"] != "not found" || body["status_code"] != float64(http.StatusNotFound) {
			t.Fatalf("body = %#v", body)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		rec := request(t, http.MethodPost, "/get", nil, nil)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		var body map[string]any
		decodeJSON(t, rec, &body)
		if body["error"] != "method not allowed" || body["status_code"] != float64(http.StatusMethodNotAllowed) {
			t.Fatalf("body = %#v", body)
		}
	})
}

func TestOpenAPIDocs(t *testing.T) {
	t.Run("spec", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/openapi.yaml", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/yaml; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want application/yaml", got)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "openapi: 3.0.3") || !strings.Contains(body, "/docs:") {
			t.Fatalf("unexpected OpenAPI body: %q", body)
		}
	})

	t.Run("swagger ui", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/docs", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want text/html", got)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "SwaggerUIBundle") || !strings.Contains(body, `url: "/openapi.yaml"`) {
			t.Fatalf("unexpected docs body: %q", body)
		}
	})
}

func TestGetEcho(t *testing.T) {
	rec := request(t, http.MethodGet, "/get?name=go&name=chi", nil, map[string]string{
		"User-Agent":        "go-test",
		"X-Forwarded-For":   "203.0.113.9, 10.0.0.1",
		"X-Forwarded-Proto": "https",
	})

	var body map[string]any
	decodeJSON(t, rec, &body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if body["origin"] != "203.0.113.9" {
		t.Fatalf("origin = %v, want forwarded ip", body["origin"])
	}
	if body["url"] != "https://example.com/get?name=go&name=chi" {
		t.Fatalf("url = %v", body["url"])
	}
	args := body["args"].(map[string]any)
	names := args["name"].([]any)
	if names[0] != "go" || names[1] != "chi" {
		t.Fatalf("args[name] = %#v", args["name"])
	}
}

func TestPostJSONEcho(t *testing.T) {
	rec := request(t, http.MethodPost, "/post", strings.NewReader(`{"hello":"world"}`), map[string]string{
		"Content-Type": "application/json",
	})

	var body map[string]any
	decodeJSON(t, rec, &body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if body["data"] != `{"hello":"world"}` {
		t.Fatalf("data = %v", body["data"])
	}
	jsonBody := body["json"].(map[string]any)
	if jsonBody["hello"] != "world" {
		t.Fatalf("json.hello = %v", jsonBody["hello"])
	}
}

func TestPostFormEcho(t *testing.T) {
	rec := request(t, http.MethodPost, "/post", strings.NewReader("a=1&a=2&b=3"), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	})

	var body map[string]any
	decodeJSON(t, rec, &body)
	form := body["form"].(map[string]any)
	if form["b"] != "3" {
		t.Fatalf("form[b] = %v", form["b"])
	}
	values := form["a"].([]any)
	if values[0] != "1" || values[1] != "2" {
		t.Fatalf("form[a] = %#v", form["a"])
	}
}

func TestPostMultipartEcho(t *testing.T) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	if err := writer.WriteField("field", "value"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("upload", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("hello"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	rec := request(t, http.MethodPost, "/post", &b, map[string]string{
		"Content-Type": writer.FormDataContentType(),
	})

	var body map[string]any
	decodeJSON(t, rec, &body)
	form := body["form"].(map[string]any)
	files := body["files"].(map[string]any)
	if form["field"] != "value" {
		t.Fatalf("form[field] = %v", form["field"])
	}
	upload := files["upload"].(map[string]any)
	if upload["filename"] != "test.txt" {
		t.Fatalf("filename = %v", upload["filename"])
	}
}

func TestAnythingIncludesMethod(t *testing.T) {
	rec := request(t, http.MethodPatch, "/anything/a/b?x=1", strings.NewReader("raw"), map[string]string{
		"Content-Type": "text/plain",
	})

	var body map[string]any
	decodeJSON(t, rec, &body)
	if body["method"] != http.MethodPatch {
		t.Fatalf("method = %v", body["method"])
	}
	if body["data"] != "raw" {
		t.Fatalf("data = %v", body["data"])
	}
}

func TestRequestInspection(t *testing.T) {
	t.Run("headers", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/headers", nil, map[string]string{"X-Test": "ok"})
		var body map[string]map[string]string
		decodeJSON(t, rec, &body)
		if body["headers"]["X-Test"] != "ok" {
			t.Fatalf("headers[X-Test] = %q", body["headers"]["X-Test"])
		}
	})

	t.Run("ip", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/ip", nil, map[string]string{"X-Forwarded-For": "198.51.100.5"})
		var body map[string]string
		decodeJSON(t, rec, &body)
		if body["origin"] != "198.51.100.5" {
			t.Fatalf("origin = %q", body["origin"])
		}
	})

	t.Run("user-agent", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/user-agent", nil, map[string]string{"User-Agent": "agent"})
		var body map[string]string
		decodeJSON(t, rec, &body)
		if body["user-agent"] != "agent" {
			t.Fatalf("user-agent = %q", body["user-agent"])
		}
	})
}

func TestStatusEndpoint(t *testing.T) {
	rec := request(t, http.MethodGet, "/status/418", nil, nil)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "teapot") {
		t.Fatalf("body = %q", rec.Body.String())
	}

	rec = request(t, http.MethodGet, "/status/999", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCacheAndETag(t *testing.T) {
	rec := request(t, http.MethodGet, "/cache/60", nil, nil)
	if rec.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Fatalf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}

	rec = request(t, http.MethodGet, "/etag/test", nil, map[string]string{"If-None-Match": `"test"`})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", rec.Code)
	}
}

func TestResponseHeaders(t *testing.T) {
	rec := request(t, http.MethodGet, "/response-headers?X-Test=ok", nil, nil)
	if rec.Header().Get("X-Test") != "ok" {
		t.Fatalf("X-Test header = %q", rec.Header().Get("X-Test"))
	}
	var body map[string]string
	decodeJSON(t, rec, &body)
	if body["X-Test"] != "ok" {
		t.Fatalf("body X-Test = %q", body["X-Test"])
	}
}

func TestAuthEndpoints(t *testing.T) {
	t.Run("basic ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/basic-auth/user/pass", nil)
		req.SetBasicAuth("user", "pass")
		rec := httptest.NewRecorder()
		testRouter().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("basic fail", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/basic-auth/user/pass", nil, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("hidden fail", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/hidden-basic-auth/user/pass", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("bearer", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/bearer", nil, map[string]string{"Authorization": "Bearer abc"})
		var body map[string]any
		decodeJSON(t, rec, &body)
		if body["token"] != "abc" {
			t.Fatalf("token = %v", body["token"])
		}
	})
}

func TestDigestAuthEndpoints(t *testing.T) {
	t.Run("md5 auth", func(t *testing.T) {
		challenge := digestChallenge(t, "/digest-auth/auth/user/pass")
		auth := digestAuthorization(t, challenge, http.MethodGet, "/digest-auth/auth/user/pass", "user", "pass", nil)

		rec := request(t, http.MethodGet, "/digest-auth/auth/user/pass", nil, map[string]string{
			"Authorization": auth,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		decodeJSON(t, rec, &body)
		if body["authenticated"] != true || body["user"] != "user" {
			t.Fatalf("body = %#v", body)
		}
	})

	t.Run("sha256 auth-int", func(t *testing.T) {
		target := "/digest-auth/auth-int/user/pass/SHA-256"
		challenge := digestChallenge(t, target)
		auth := digestAuthorization(t, challenge, http.MethodGet, target, "user", "pass", nil)

		rec := request(t, http.MethodGet, target, nil, map[string]string{
			"Authorization": auth,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("wrong credentials", func(t *testing.T) {
		challenge := digestChallenge(t, "/digest-auth/auth/user/pass")
		auth := digestAuthorization(t, challenge, http.MethodGet, "/digest-auth/auth/user/pass", "user", "wrong", nil)

		rec := request(t, http.MethodGet, "/digest-auth/auth/user/pass", nil, map[string]string{
			"Authorization": auth,
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Digest ") {
			t.Fatalf("WWW-Authenticate = %q, want Digest challenge", got)
		}
	})
}

func TestDynamicEndpoints(t *testing.T) {
	t.Run("uuid", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/uuid", nil, nil)
		var body map[string]string
		decodeJSON(t, rec, &body)
		if len(body["uuid"]) != 36 {
			t.Fatalf("uuid = %q", body["uuid"])
		}
	})

	t.Run("bytes", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/bytes/16", nil, nil)
		if rec.Code != http.StatusOK || rec.Body.Len() != 16 {
			t.Fatalf("status/body = %d/%d, want 200/16", rec.Code, rec.Body.Len())
		}
	})

	t.Run("stream", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/stream/3", nil, nil)
		lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
		if len(lines) != 3 {
			t.Fatalf("lines = %d, want 3: %q", len(lines), rec.Body.String())
		}
	})

	t.Run("delay", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/delay/0", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("drip", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/drip?duration=0&numbytes=4&code=201", nil, nil)
		if rec.Code != http.StatusCreated || rec.Body.String() != "****" {
			t.Fatalf("status/body = %d/%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("base64", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/base64/aGVsbG8=", nil, nil)
		if rec.Code != http.StatusOK || rec.Body.String() != "hello" {
			t.Fatalf("status/body = %d/%q", rec.Code, rec.Body.String())
		}
	})
}

func TestRangeEndpoint(t *testing.T) {
	t.Run("full response", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/range/10", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if rec.Body.String() != "abcdefghij" {
			t.Fatalf("body = %q, want alphabet range", rec.Body.String())
		}
		if got := rec.Header().Get("Accept-Ranges"); got != "bytes" {
			t.Fatalf("Accept-Ranges = %q, want bytes", got)
		}
		if got := rec.Header().Get("Content-Range"); got != "bytes 0-9/10" {
			t.Fatalf("Content-Range = %q, want bytes 0-9/10", got)
		}
		if got := rec.Header().Get("ETag"); got != "range10" {
			t.Fatalf("ETag = %q, want range10", got)
		}
	})

	t.Run("partial response", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/range/10", nil, map[string]string{
			"Range": "bytes=2-5",
		})
		if rec.Code != http.StatusPartialContent {
			t.Fatalf("status = %d, want 206", rec.Code)
		}
		if rec.Body.String() != "cdef" {
			t.Fatalf("body = %q, want cdef", rec.Body.String())
		}
		if got := rec.Header().Get("Content-Range"); got != "bytes 2-5/10" {
			t.Fatalf("Content-Range = %q, want bytes 2-5/10", got)
		}
	})

	t.Run("invalid range", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/range/10", nil, map[string]string{
			"Range": "bytes=10-20",
		})
		if rec.Code != http.StatusRequestedRangeNotSatisfiable {
			t.Fatalf("status = %d, want 416", rec.Code)
		}
		if got := rec.Header().Get("Content-Range"); got != "bytes */10" {
			t.Fatalf("Content-Range = %q, want bytes */10", got)
		}
	})
}

func TestLimits(t *testing.T) {
	rec := request(t, http.MethodPost, "/post", strings.NewReader(strings.Repeat("x", 2048)), nil)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}

	rec = request(t, http.MethodGet, "/bytes/129", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	rec = request(t, http.MethodGet, "/delay/1", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCookiesAndRedirects(t *testing.T) {
	t.Run("cookies", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/cookies", nil, map[string]string{"Cookie": "a=1; b=2"})
		var body map[string]map[string]string
		decodeJSON(t, rec, &body)
		if body["cookies"]["a"] != "1" || body["cookies"]["b"] != "2" {
			t.Fatalf("cookies = %#v", body["cookies"])
		}
	})

	t.Run("set cookie", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/cookies/set/session/abc", nil, nil)
		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", rec.Code)
		}
		if rec.Header().Get("Location") != "/cookies" {
			t.Fatalf("Location = %q", rec.Header().Get("Location"))
		}
	})

	t.Run("redirect", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/redirect-to?url=/get&status_code=307", nil, nil)
		if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "/get" {
			t.Fatalf("status/location = %d/%q", rec.Code, rec.Header().Get("Location"))
		}
	})
}

func TestEncodedAndFormatEndpoints(t *testing.T) {
	t.Run("gzip", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/gzip", nil, nil)
		if rec.Header().Get("Content-Encoding") != "gzip" {
			t.Fatalf("Content-Encoding = %q", rec.Header().Get("Content-Encoding"))
		}
		reader, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = reader.Close()
		}()
		var body map[string]any
		if err := json.NewDecoder(reader).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["gzipped"] != true {
			t.Fatalf("gzipped = %v", body["gzipped"])
		}
	})

	t.Run("deflate", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/deflate", nil, nil)
		reader, err := zlib.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = reader.Close()
		}()
		var body map[string]any
		if err := json.NewDecoder(reader).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["deflated"] != true {
			t.Fatalf("deflated = %v", body["deflated"])
		}
	})

	t.Run("brotli", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/brotli", nil, nil)
		if rec.Header().Get("Content-Encoding") != "br" {
			t.Fatalf("Content-Encoding = %q, want br", rec.Header().Get("Content-Encoding"))
		}
		var body map[string]any
		if err := json.NewDecoder(brotli.NewReader(rec.Body)).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["brotli"] != true {
			t.Fatalf("brotli = %v", body["brotli"])
		}
	})

	t.Run("json", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/json", nil, nil)
		var body map[string]any
		decodeJSON(t, rec, &body)
		if body["slideshow"] == nil {
			t.Fatalf("missing slideshow: %#v", body)
		}
	})

	t.Run("image", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/image/png", nil, nil)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" || rec.Body.Len() == 0 {
			t.Fatalf("unexpected image response: status=%d content-type=%q len=%d", rec.Code, rec.Header().Get("Content-Type"), rec.Body.Len())
		}
	})

	t.Run("webp", func(t *testing.T) {
		rec := request(t, http.MethodGet, "/image/webp", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if rec.Header().Get("Content-Type") != "image/webp" {
			t.Fatalf("Content-Type = %q, want image/webp", rec.Header().Get("Content-Type"))
		}
		body := rec.Body.Bytes()
		if len(body) < 12 || string(body[:4]) != "RIFF" || string(body[8:12]) != "WEBP" {
			t.Fatalf("body does not look like WebP: len=%d", len(body))
		}
	})
}

func TestCompatibilityMethods(t *testing.T) {
	t.Run("status patch and trace", func(t *testing.T) {
		for _, method := range []string{http.MethodPatch, http.MethodTrace} {
			rec := request(t, method, "/status/204", nil, nil)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("%s status = %d, want 204", method, rec.Code)
			}
		}
	})

	t.Run("delay post echoes body", func(t *testing.T) {
		rec := request(t, http.MethodPost, "/delay/0", strings.NewReader("hello"), map[string]string{
			"Content-Type": "text/plain",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body map[string]any
		decodeJSON(t, rec, &body)
		if body["data"] != "hello" {
			t.Fatalf("data = %v, want hello", body["data"])
		}
	})

	t.Run("redirect-to supports mutation methods", func(t *testing.T) {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodTrace} {
			rec := request(t, method, "/redirect-to?url=/get&status_code=307", nil, nil)
			if rec.Code != http.StatusTemporaryRedirect {
				t.Fatalf("%s status = %d, want 307", method, rec.Code)
			}
			if rec.Header().Get("Location") != "/get" {
				t.Fatalf("%s Location = %q, want /get", method, rec.Header().Get("Location"))
			}
		}
	})

	t.Run("anything trace", func(t *testing.T) {
		rec := request(t, http.MethodTrace, "/anything/a/b?x=1", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body map[string]any
		decodeJSON(t, rec, &body)
		if body["method"] != http.MethodTrace {
			t.Fatalf("method = %v, want TRACE", body["method"])
		}
	})
}

func TestSlogRequestLoggerWritesJSON(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := middleware.RequestID(slogRequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]string{"ok": "true"})
	})))

	req := httptest.NewRequest(http.MethodPost, "/log-test?x=1", nil)
	req.Header.Set("User-Agent", "logger-test")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry); err != nil {
		t.Fatalf("decode log JSON: %v; log=%q", err, logs.String())
	}
	if entry["msg"] != "http_request" {
		t.Fatalf("msg = %v, want http_request", entry["msg"])
	}
	if entry["method"] != http.MethodPost || entry["path"] != "/log-test" || entry["query"] != "x=1" {
		t.Fatalf("unexpected request fields: %#v", entry)
	}
	if entry["proto"] != "HTTP/1.1" {
		t.Fatalf("proto = %v, want HTTP/1.1", entry["proto"])
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Fatalf("status = %v, want 201", entry["status"])
	}
	if entry["user_agent"] != "logger-test" {
		t.Fatalf("user_agent = %v, want logger-test", entry["user_agent"])
	}
	if entry["request_id"] == "" {
		t.Fatalf("missing request_id: %#v", entry)
	}
	durationText, ok := entry["duration"].(string)
	if !ok {
		t.Fatalf("duration = %#v, want Chi-style duration string", entry["duration"])
	}
	duration, err := time.ParseDuration(durationText)
	if err != nil || duration <= 0 {
		t.Fatalf("duration = %q, want positive Go duration string", durationText)
	}
}

func request(t *testing.T, method, target string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, value any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), value); err != nil {
		t.Fatalf("decode JSON: %v; body=%q", err, rec.Body.String())
	}
}

func digestChallenge(t *testing.T, target string) map[string]string {
	t.Helper()
	rec := request(t, http.MethodGet, target, nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("initial digest status = %d, want 401", rec.Code)
	}
	challenge := parseAuthParams(strings.TrimPrefix(rec.Header().Get("WWW-Authenticate"), "Digest "))
	if challenge["realm"] == "" || challenge["nonce"] == "" || challenge["qop"] == "" || challenge["algorithm"] == "" {
		t.Fatalf("incomplete digest challenge: %#v", challenge)
	}
	return challenge
}

func digestAuthorization(t *testing.T, challenge map[string]string, method, target, user, password string, body []byte) string {
	t.Helper()
	algorithm := challenge["algorithm"]
	qop := challenge["qop"]
	uri := target
	nc := "00000001"
	cnonce := "test-cnonce"

	ha1 := testDigestHex(t, algorithm, fmt.Sprintf("%s:%s:%s", user, challenge["realm"], password))
	ha2Input := method + ":" + uri
	if qop == "auth-int" {
		ha2Input += ":" + testDigestHex(t, algorithm, string(body))
	}
	ha2 := testDigestHex(t, algorithm, ha2Input)
	response := testDigestHex(t, algorithm, fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, challenge["nonce"], nc, cnonce, qop, ha2))

	return fmt.Sprintf(
		`Digest username="%s", realm="%s", nonce="%s", uri="%s", algorithm=%s, response="%s", opaque="%s", qop=%s, nc=%s, cnonce="%s"`,
		user,
		challenge["realm"],
		challenge["nonce"],
		uri,
		algorithm,
		response,
		challenge["opaque"],
		qop,
		nc,
		cnonce,
	)
}

func testDigestHex(t *testing.T, algorithm, value string) string {
	t.Helper()
	switch strings.ToUpper(algorithm) {
	case "MD5":
		sum := md5.Sum([]byte(value))
		return hex.EncodeToString(sum[:])
	case "SHA-256":
		sum := sha256.Sum256([]byte(value))
		return hex.EncodeToString(sum[:])
	default:
		t.Fatalf("unsupported test digest algorithm %q", algorithm)
		return ""
	}
}

func parseAuthParams(value string) map[string]string {
	params := map[string]string{}
	for _, part := range splitAuthParams(value) {
		key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		params[key] = strings.Trim(strings.TrimSpace(val), `"`)
	}
	return params
}

func splitAuthParams(value string) []string {
	var parts []string
	var b strings.Builder
	inQuotes := false
	for _, r := range value {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				parts = append(parts, b.String())
				b.Reset()
				continue
			}
		}
		b.WriteRune(r)
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}
