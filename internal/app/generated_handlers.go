package app

import (
	"net/http"

	"github.com/vancanhuit/go-httpbin/internal/api"
)

var _ api.ServerInterface = (*server)(nil)

func (s *server) Index(w http.ResponseWriter, r *http.Request) { s.index(w, r) }

func (s *server) Healthz(w http.ResponseWriter, r *http.Request) { s.health(w, r) }

func (s *server) Readyz(w http.ResponseWriter, r *http.Request) { s.health(w, r) }

func (s *server) GetEcho(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, false, false)
}

func (s *server) PostEcho(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, false)
}

func (s *server) PutEcho(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, false)
}

func (s *server) PatchEcho(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, false)
}

func (s *server) DeleteEcho(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, false)
}

func (s *server) AnythingGet(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, false, true)
}

func (s *server) AnythingPost(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, true)
}

func (s *server) AnythingPut(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, true)
}

func (s *server) AnythingPatch(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, true)
}

func (s *server) AnythingDelete(w http.ResponseWriter, r *http.Request) {
	s.writeEcho(w, r, true, true)
}

func (s *server) AnythingPathGet(w http.ResponseWriter, r *http.Request, _ api.Path) {
	s.writeEcho(w, r, false, true)
}

func (s *server) AnythingPathPost(w http.ResponseWriter, r *http.Request, _ api.Path) {
	s.writeEcho(w, r, true, true)
}

func (s *server) AnythingPathPut(w http.ResponseWriter, r *http.Request, _ api.Path) {
	s.writeEcho(w, r, true, true)
}

func (s *server) AnythingPathPatch(w http.ResponseWriter, r *http.Request, _ api.Path) {
	s.writeEcho(w, r, true, true)
}

func (s *server) AnythingPathDelete(w http.ResponseWriter, r *http.Request, _ api.Path) {
	s.writeEcho(w, r, true, true)
}

func (s *server) Headers(w http.ResponseWriter, r *http.Request) { s.headers(w, r) }

func (s *server) IP(w http.ResponseWriter, r *http.Request) { s.ip(w, r) }

func (s *server) UserAgent(w http.ResponseWriter, r *http.Request) { s.userAgent(w, r) }

func (s *server) StatusGet(w http.ResponseWriter, r *http.Request, _ api.Codes) {
	s.status(w, r)
}

func (s *server) StatusPost(w http.ResponseWriter, r *http.Request, _ api.Codes) {
	s.status(w, r)
}

func (s *server) StatusPut(w http.ResponseWriter, r *http.Request, _ api.Codes) {
	s.status(w, r)
}

func (s *server) StatusDelete(w http.ResponseWriter, r *http.Request, _ api.Codes) {
	s.status(w, r)
}

func (s *server) ResponseHeadersGet(w http.ResponseWriter, r *http.Request) {
	s.responseHeaders(w, r)
}

func (s *server) ResponseHeadersPost(w http.ResponseWriter, r *http.Request) {
	s.responseHeaders(w, r)
}

func (s *server) Cache(w http.ResponseWriter, r *http.Request) { s.cache(w, r) }

func (s *server) CacheSeconds(w http.ResponseWriter, r *http.Request, _ api.Seconds) {
	s.cache(w, r)
}

func (s *server) ETag(w http.ResponseWriter, r *http.Request, _ api.ETag) { s.etag(w, r) }

func (s *server) BasicAuth(w http.ResponseWriter, r *http.Request, _ api.User, _ api.Password) {
	s.basicAuth(false)(w, r)
}

func (s *server) HiddenBasicAuth(w http.ResponseWriter, r *http.Request, _ api.User, _ api.Password) {
	s.basicAuth(true)(w, r)
}

func (s *server) Bearer(w http.ResponseWriter, r *http.Request) { s.bearer(w, r) }

func (s *server) UUID(w http.ResponseWriter, r *http.Request) { s.uuid(w, r) }

func (s *server) Bytes(w http.ResponseWriter, r *http.Request, _ api.N) { s.bytes(w, r) }

func (s *server) StreamBytes(w http.ResponseWriter, r *http.Request, _ api.N) {
	s.streamBytes(w, r)
}

func (s *server) Stream(w http.ResponseWriter, r *http.Request, _ api.N) { s.stream(w, r) }

func (s *server) Delay(w http.ResponseWriter, r *http.Request, _ api.Delay) { s.delay(w, r) }

func (s *server) Drip(w http.ResponseWriter, r *http.Request) { s.drip(w, r) }

func (s *server) Base64Decode(w http.ResponseWriter, r *http.Request, _ api.Value) {
	s.base64(w, r)
}

func (s *server) Cookies(w http.ResponseWriter, r *http.Request) { s.cookies(w, r) }

func (s *server) SetCookiesFromQuery(w http.ResponseWriter, r *http.Request) {
	s.setCookiesFromQuery(w, r)
}

func (s *server) SetCookie(w http.ResponseWriter, r *http.Request, _ api.Name, _ api.Value) {
	s.setCookie(w, r)
}

func (s *server) DeleteCookies(w http.ResponseWriter, r *http.Request) {
	s.deleteCookies(w, r)
}

func (s *server) Redirect(w http.ResponseWriter, r *http.Request, _ api.N) { s.redirect(w, r) }

func (s *server) RelativeRedirect(w http.ResponseWriter, r *http.Request, _ api.N) {
	s.relativeRedirect(w, r)
}

func (s *server) AbsoluteRedirect(w http.ResponseWriter, r *http.Request, _ api.N) {
	s.absoluteRedirect(w, r)
}

func (s *server) RedirectTo(w http.ResponseWriter, r *http.Request) { s.redirectTo(w, r) }

func (s *server) Gzip(w http.ResponseWriter, r *http.Request) { s.gzip(w, r) }

func (s *server) Deflate(w http.ResponseWriter, r *http.Request) { s.deflate(w, r) }

func (s *server) JSON(w http.ResponseWriter, r *http.Request) { s.json(w, r) }

func (s *server) XML(w http.ResponseWriter, r *http.Request) { s.xml(w, r) }

func (s *server) HTML(w http.ResponseWriter, r *http.Request) { s.html(w, r) }

func (s *server) Robots(w http.ResponseWriter, r *http.Request) { s.robots(w, r) }

func (s *server) Deny(w http.ResponseWriter, r *http.Request) { s.deny(w, r) }

func (s *server) UTF8(w http.ResponseWriter, r *http.Request) { s.utf8(w, r) }

func (s *server) Image(w http.ResponseWriter, r *http.Request) { s.image(w, r) }

func (s *server) ImagePNG(w http.ResponseWriter, r *http.Request) { s.pngImage(w, r) }

func (s *server) ImageJPEG(w http.ResponseWriter, r *http.Request) { s.jpegImage(w, r) }

func (s *server) ImageGIF(w http.ResponseWriter, r *http.Request) { s.gifImage(w, r) }

func (s *server) ImageSVG(w http.ResponseWriter, r *http.Request) { s.svgImage(w, r) }

func (s *server) Links(w http.ResponseWriter, r *http.Request, _ api.N, _ api.Offset) {
	s.links(w, r)
}
