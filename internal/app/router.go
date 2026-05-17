package app

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"math"
	mathrand "math/rand"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apispec "github.com/vancanhuit/go-httpbin/api"
	"github.com/vancanhuit/go-httpbin/internal/api"
	"github.com/vancanhuit/go-httpbin/internal/config"
)

type server struct {
	cfg config.Config
}

func NewRouter(cfg config.Config) http.Handler {
	s := &server{cfg: cfg}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(slogRequestLogger(slog.Default()))
	r.Use(slogRecoverer(slog.Default()))
	if cfg.TrustProxyHeaders {
		r.Use(middleware.RealIP)
	}
	if cfg.HandlerTimeout > 0 {
		r.Use(middleware.Timeout(cfg.HandlerTimeout))
	}
	if cfg.EnableCompression {
		r.Use(middleware.Compress(5, "application/json", "text/plain", "text/html", "application/xml", "image/svg+xml"))
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeHTTPError(w, http.StatusNotFound, "not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	api.HandlerWithOptions(s, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeHTTPError(w, http.StatusBadRequest, err.Error())
		},
	})
	r.HandleFunc("/anything/*", s.echoWithBody(true))
	r.Get("/base64/*", s.base64)

	return r
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "go-httpbin",
		"message": "HTTP request and response testing service",
	})
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) openAPIYAML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(apispec.OpenAPIYAML)
}

func (s *server) docs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerUIHTML))
}

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>go-httpbin API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fff; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>
`

func (s *server) echoWithBody(includeMethod bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.writeEcho(w, r, true, includeMethod)
	}
}

func (s *server) writeEcho(w http.ResponseWriter, r *http.Request, includeBody bool, includeMethod bool) {
	resp := map[string]any{
		"args":    queryArgs(r.URL.Query()),
		"headers": requestHeaders(r),
		"origin":  s.origin(r),
		"url":     s.requestURL(r),
	}
	if includeMethod {
		resp["method"] = r.Method
	}
	if includeBody {
		body, err := readLimitedBody(w, r, s.cfg.MaxBodyBytes)
		if err != nil {
			writeHTTPError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		data, form, files, jsonValue := parseBody(r.Header.Get("Content-Type"), body)
		resp["data"] = data
		resp["files"] = files
		resp["form"] = form
		resp["json"] = jsonValue
	}
	writeJSON(w, http.StatusOK, resp)
}

func readLimitedBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		return nil, err
	}
	return body, nil
}

func parseBody(contentType string, body []byte) (string, map[string]any, map[string]any, any) {
	form := map[string]any{}
	files := map[string]any{}
	if len(body) == 0 {
		return "", form, files, nil
	}

	mediaType, params, _ := mime.ParseMediaType(contentType)
	switch mediaType {
	case "application/json", "text/json":
		var value any
		if err := json.Unmarshal(body, &value); err == nil {
			return string(body), form, files, value
		}
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err == nil {
			form = queryArgs(values)
		}
	case "multipart/form-data":
		if boundary := params["boundary"]; boundary != "" {
			mr := multipart.NewReader(bytes.NewReader(body), boundary)
			form, files = parseMultipart(mr)
		}
	}
	return string(body), form, files, nil
}

func parseMultipart(mr *multipart.Reader) (map[string]any, map[string]any) {
	form := map[string]any{}
	files := map[string]any{}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			break
		}
		content, _ := io.ReadAll(part)
		name := part.FormName()
		if name == "" {
			continue
		}
		if filename := part.FileName(); filename != "" {
			files[name] = map[string]any{
				"filename": filename,
				"size":     len(content),
			}
			continue
		}
		addMultiValue(form, name, string(content))
	}
	return form, files
}

func addMultiValue(m map[string]any, key, value string) {
	existing, ok := m[key]
	if !ok {
		m[key] = value
		return
	}
	switch values := existing.(type) {
	case []string:
		m[key] = append(values, value)
	case string:
		m[key] = []string{values, value}
	}
}

func (s *server) headers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"headers": requestHeaders(r)})
}

func (s *server) ip(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"origin": s.origin(r)})
}

func (s *server) userAgent(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"user-agent": r.UserAgent()})
}

func (s *server) status(w http.ResponseWriter, r *http.Request) {
	code, ok := chooseStatus(chi.URLParam(r, "codes"))
	if !ok {
		writeHTTPError(w, http.StatusBadRequest, "invalid status code")
		return
	}
	if code == http.StatusTeapot {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(code)
		_, _ = w.Write([]byte("I'm a teapot\n"))
		return
	}
	w.WriteHeader(code)
}

func chooseStatus(value string) (int, bool) {
	parts := strings.Split(value, ",")
	codes := make([]int, 0, len(parts))
	for _, part := range parts {
		codePart := strings.SplitN(strings.TrimSpace(part), ":", 2)[0]
		code, err := strconv.Atoi(codePart)
		if err != nil || code < 100 || code > 599 {
			return 0, false
		}
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return 0, false
	}
	return codes[mathrand.Intn(len(codes))], true
}

func (s *server) responseHeaders(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{}
	for key, values := range r.URL.Query() {
		if key == "" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
		resp[key] = collapseValues(values)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) cache(w http.ResponseWriter, r *http.Request) {
	seconds := "0"
	if value := chi.URLParam(r, "seconds"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			writeHTTPError(w, http.StatusBadRequest, "invalid cache duration")
			return
		}
		seconds = strconv.Itoa(n)
	}
	w.Header().Set("Cache-Control", "public, max-age="+seconds)
	if r.Header.Get("If-Modified-Since") != "" || r.Header.Get("If-None-Match") != "" {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	s.writeEcho(w, r, false, false)
}

func (s *server) etag(w http.ResponseWriter, r *http.Request) {
	etag := quoteETag(chi.URLParam(r, "etag"))
	w.Header().Set("ETag", etag)
	if matchETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	s.writeEcho(w, r, false, false)
}

func quoteETag(value string) string {
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return value
	}
	return `"` + value + `"`
}

func matchETag(header, etag string) bool {
	for part := range strings.SplitSeq(header, ",") {
		if strings.TrimSpace(part) == etag || strings.TrimSpace(part) == "*" {
			return true
		}
	}
	return false
}

func (s *server) basicAuth(hidden bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := chi.URLParam(r, "user")
		password := chi.URLParam(r, "password")
		gotUser, gotPassword, ok := r.BasicAuth()
		if !ok || gotUser != user || gotPassword != password {
			if hidden {
				writeHTTPError(w, http.StatusNotFound, "not found")
				return
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="go-httpbin"`)
			writeHTTPError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"user":          user,
		})
	}
}

func (s *server) bearer(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok || token == "" {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeHTTPError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"token":         token,
	})
}

func (s *server) uuid(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"uuid": newUUID()})
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *server) bytes(w http.ResponseWriter, r *http.Request) {
	n, ok := s.boundedIntParam(w, r, "n", s.cfg.MaxRandomBytes)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	writeRandom(w, n)
}

func (s *server) streamBytes(w http.ResponseWriter, r *http.Request) {
	n, ok := s.boundedIntParam(w, r, "n", s.cfg.MaxRandomBytes)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	writeRandomStream(r.Context(), w, n)
}

func (s *server) stream(w http.ResponseWriter, r *http.Request) {
	n, ok := s.boundedIntParam(w, r, "n", s.cfg.MaxStreamItems)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	for i := range n {
		if err := r.Context().Err(); err != nil {
			return
		}
		line := map[string]any{
			"id":      i,
			"args":    queryArgs(r.URL.Query()),
			"headers": requestHeaders(r),
			"origin":  s.origin(r),
			"url":     s.requestURL(r),
		}
		if err := encoder.Encode(line); err != nil {
			return
		}
		flush(w)
	}
}

func (s *server) delay(w http.ResponseWriter, r *http.Request) {
	seconds, err := strconv.ParseFloat(chi.URLParam(r, "delay"), 64)
	if err != nil || seconds < 0 {
		writeHTTPError(w, http.StatusBadRequest, "invalid delay")
		return
	}
	delay := time.Duration(seconds * float64(time.Second))
	if delay > s.cfg.MaxDelay {
		writeHTTPError(w, http.StatusBadRequest, "delay exceeds configured maximum")
		return
	}
	if !sleepContext(r.Context(), delay) {
		return
	}
	s.writeEcho(w, r, false, false)
}

func (s *server) drip(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	duration := parseFloatDefault(q.Get("duration"), 2)
	delay := parseFloatDefault(q.Get("delay"), 0)
	numBytes := parseIntDefault(q.Get("numbytes"), 10)
	code := parseIntDefault(q.Get("code"), http.StatusOK)
	if duration < 0 || delay < 0 || numBytes < 0 || code < 100 || code > 599 {
		writeHTTPError(w, http.StatusBadRequest, "invalid drip parameters")
		return
	}
	totalDelay := time.Duration((duration + delay) * float64(time.Second))
	if totalDelay > s.cfg.MaxDelay {
		writeHTTPError(w, http.StatusBadRequest, "drip duration exceeds configured maximum")
		return
	}
	if numBytes > s.cfg.MaxRandomBytes {
		writeHTTPError(w, http.StatusBadRequest, "numbytes exceeds configured maximum")
		return
	}
	if !sleepContext(r.Context(), time.Duration(delay*float64(time.Second))) {
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(code)
	if numBytes == 0 {
		return
	}
	interval := time.Duration(duration * float64(time.Second) / float64(numBytes))
	for i := range numBytes {
		if _, err := w.Write([]byte("*")); err != nil {
			return
		}
		flush(w)
		if interval > 0 && i < numBytes-1 && !sleepContext(r.Context(), interval) {
			return
		}
	}
}

func (s *server) base64(w http.ResponseWriter, r *http.Request) {
	value := chi.URLParam(r, "*")
	if value == "" {
		value = chi.URLParam(r, "value")
	}
	decoded, err := decodeBase64(value)
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, "invalid base64 data")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(decoded)
}

func decodeBase64(value string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("invalid base64")
}

func (s *server) cookies(w http.ResponseWriter, r *http.Request) {
	cookies := map[string]string{}
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	writeJSON(w, http.StatusOK, map[string]any{"cookies": cookies})
}

func (s *server) setCookiesFromQuery(w http.ResponseWriter, r *http.Request) {
	for name, values := range r.URL.Query() {
		for _, value := range values {
			http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/"})
		}
	}
	http.Redirect(w, r, "/cookies", http.StatusFound)
}

func (s *server) setCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:  chi.URLParam(r, "name"),
		Value: chi.URLParam(r, "value"),
		Path:  "/",
	})
	http.Redirect(w, r, "/cookies", http.StatusFound)
}

func (s *server) deleteCookies(w http.ResponseWriter, r *http.Request) {
	for name := range r.URL.Query() {
		http.SetCookie(w, &http.Cookie{Name: name, Path: "/", MaxAge: -1})
	}
	http.Redirect(w, r, "/cookies", http.StatusFound)
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request) {
	n, ok := parseRedirectCount(w, chi.URLParam(r, "n"))
	if !ok {
		return
	}
	if n == 0 {
		http.Redirect(w, r, "/get", http.StatusFound)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/relative-redirect/%d", n-1), http.StatusFound)
}

func (s *server) relativeRedirect(w http.ResponseWriter, r *http.Request) {
	n, ok := parseRedirectCount(w, chi.URLParam(r, "n"))
	if !ok {
		return
	}
	if n == 0 {
		http.Redirect(w, r, "/get", http.StatusFound)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/relative-redirect/%d", n-1), http.StatusFound)
}

func (s *server) absoluteRedirect(w http.ResponseWriter, r *http.Request) {
	n, ok := parseRedirectCount(w, chi.URLParam(r, "n"))
	if !ok {
		return
	}
	target := "/get"
	if n > 0 {
		target = fmt.Sprintf("/absolute-redirect/%d", n-1)
	}
	http.Redirect(w, r, s.absoluteURL(r, target), http.StatusFound)
}

func parseRedirectCount(w http.ResponseWriter, value string) (int, bool) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 || n > 50 {
		writeHTTPError(w, http.StatusBadRequest, "invalid redirect count")
		return 0, false
	}
	return n, true
}

func (s *server) redirectTo(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		writeHTTPError(w, http.StatusBadRequest, "url is required")
		return
	}
	code := parseIntDefault(r.URL.Query().Get("status_code"), http.StatusFound)
	if code < 300 || code > 399 {
		writeHTTPError(w, http.StatusBadRequest, "status_code must be a redirect")
		return
	}
	http.Redirect(w, r, target, code)
}

func (s *server) gzip(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(http.StatusOK)
	gz := gzip.NewWriter(w)
	defer func() {
		_ = gz.Close()
	}()
	_ = json.NewEncoder(gz).Encode(encodedResponse(true, r, s.origin(r)))
}

func (s *server) deflate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "deflate")
	w.WriteHeader(http.StatusOK)
	zw := zlib.NewWriter(w)
	defer func() {
		_ = zw.Close()
	}()
	_ = json.NewEncoder(zw).Encode(encodedResponse(false, r, s.origin(r)))
}

func encodedResponse(gzipped bool, r *http.Request, origin string) map[string]any {
	return map[string]any{
		"gzipped":  gzipped,
		"deflated": !gzipped,
		"headers":  requestHeaders(r),
		"method":   r.Method,
		"origin":   origin,
	}
}

func (s *server) json(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"slideshow": map[string]any{
			"title":  "Sample Slide Show",
			"author": "Yours Truly",
			"slides": []map[string]any{
				{"title": "Wake up to WonderWidgets!", "type": "all"},
				{"title": "Overview", "type": "all", "items": []string{"Why WonderWidgets are great", "Who buys WonderWidgets"}},
			},
		},
	})
}

func (s *server) xml(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><slideshow title="Sample Slide Show"><slide type="all">Wake up to WonderWidgets!</slide></slideshow>`))
}

func (s *server) html(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><title>go-httpbin</title></head><body><h1>go-httpbin</h1><p>HTTP testing service.</p></body></html>`))
}

func (s *server) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("User-agent: *\nDisallow: /deny\n"))
}

func (s *server) deny(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("YOU SHOULDN'T BE HERE\n"))
}

func (s *server) utf8(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><html><body><h1>Unicode Demo</h1><p>こんにちは, Привет, مرحبا</p></body></html>"))
}

func (s *server) image(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	switch {
	case strings.Contains(accept, "image/svg+xml"):
		s.svgImage(w, r)
	case strings.Contains(accept, "image/jpeg"):
		s.jpegImage(w, r)
	case strings.Contains(accept, "image/gif"):
		s.gifImage(w, r)
	default:
		s.pngImage(w, r)
	}
}

func (s *server) pngImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	_ = png.Encode(w, testImage())
}

func (s *server) jpegImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	_ = jpeg.Encode(w, testImage(), &jpeg.Options{Quality: 85})
}

func (s *server) gifImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/gif")
	_ = gif.Encode(w, testImage(), nil)
}

func (s *server) svgImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128"><rect width="128" height="128" fill="#f5f5f5"/><circle cx="64" cy="64" r="42" fill="#2b7fff"/><text x="64" y="72" text-anchor="middle" font-size="20" font-family="sans-serif" fill="white">Go</text></svg>`))
}

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			if (x/8+y/8)%2 == 0 {
				img.Set(x, y, color.RGBA{R: 43, G: 127, B: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{R: 245, G: 245, B: 245, A: 255})
			}
		}
	}
	return img
}

func (s *server) links(w http.ResponseWriter, r *http.Request) {
	n, ok := s.boundedIntParam(w, r, "n", s.cfg.MaxStreamItems)
	if !ok {
		return
	}
	offset, err := strconv.Atoi(chi.URLParam(r, "offset"))
	if err != nil || offset < 0 || offset >= n {
		writeHTTPError(w, http.StatusBadRequest, "invalid offset")
		return
	}

	var b strings.Builder
	b.WriteString("<html><body>")
	for i := range n {
		if i == offset {
			fmt.Fprintf(&b, "%d ", i)
			continue
		}
		fmt.Fprintf(&b, `<a href="/links/%d/%d">%d</a> `, n, i, i)
	}
	b.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func (s *server) boundedIntParam(w http.ResponseWriter, r *http.Request, name string, max int) (int, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, name))
	if err != nil || n < 0 {
		writeHTTPError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	if n > max {
		writeHTTPError(w, http.StatusBadRequest, name+" exceeds configured maximum")
		return 0, false
	}
	return n, true
}

func requestHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string, len(r.Header)+1)
	for name, values := range r.Header {
		headers[http.CanonicalHeaderKey(name)] = strings.Join(values, ",")
	}
	if r.Host != "" {
		headers["Host"] = r.Host
	}
	return headers
}

func queryArgs(values url.Values) map[string]any {
	args := map[string]any{}
	for key, vals := range values {
		args[key] = collapseValues(vals)
	}
	return args
}

func collapseValues(values []string) any {
	if len(values) == 1 {
		return values[0]
	}
	return values
}

func (s *server) origin(r *http.Request) string {
	if s.cfg.TrustProxyHeaders {
		if forwarded := firstForwardedFor(r.Header.Get("Forwarded")); forwarded != "" {
			return forwarded
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
		if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
			return realIP
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func firstForwardedFor(header string) string {
	for field := range strings.SplitSeq(header, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
		if ok && strings.EqualFold(key, "for") {
			return strings.Trim(value, `"`)
		}
	}
	return ""
}

func (s *server) requestURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if s.cfg.TrustProxyHeaders {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
		}
	}
	return scheme + "://" + r.Host + r.URL.RequestURI()
}

func (s *server) absoluteURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if s.cfg.TrustProxyHeaders {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
		}
	}
	return scheme + "://" + r.Host + path
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeHTTPError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":       message,
		"status_code": status,
	})
}

func writeRandom(w io.Writer, n int) {
	if n == 0 {
		return
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	_, _ = w.Write(buf)
}

func writeRandomStream(ctx context.Context, w io.Writer, n int) {
	const chunkSize = 32 * 1024
	remaining := n
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return
		}
		size := int(math.Min(float64(chunkSize), float64(remaining)))
		writeRandom(w, size)
		remaining -= size
		flush(w)
	}
}

func flush(w io.Writer) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func parseFloatDefault(value string, fallback float64) float64 {
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return math.NaN()
	}
	return n
}

func parseIntDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return math.MinInt
	}
	return n
}
