package app

import (
	"net/http"

	"github.com/vancanhuit/go-httpbin/internal/api"
)

var _ api.ServerInterface = (*server)(nil)

func (s *server) Index(w http.ResponseWriter, r *http.Request) { s.index(w, r) }

func (s *server) Healthz(w http.ResponseWriter, r *http.Request) { s.health(w, r) }

func (s *server) Readyz(w http.ResponseWriter, r *http.Request) { s.health(w, r) }

func (s *server) Version(w http.ResponseWriter, r *http.Request) { s.version(w, r) }

func (s *server) OpenAPIYAML(w http.ResponseWriter, r *http.Request) {
	s.openAPIYAML(w, r)
}

func (s *server) Docs(w http.ResponseWriter, r *http.Request) { s.docs(w, r) }

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

func (s *server) AnythingTrace(w http.ResponseWriter, r *http.Request) {
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

func (s *server) AnythingPathTrace(w http.ResponseWriter, r *http.Request, _ api.Path) {
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

func (s *server) StatusPatch(w http.ResponseWriter, r *http.Request, _ api.Codes) {
	s.status(w, r)
}

func (s *server) StatusDelete(w http.ResponseWriter, r *http.Request, _ api.Codes) {
	s.status(w, r)
}

func (s *server) StatusTrace(w http.ResponseWriter, r *http.Request, _ api.Codes) {
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

func (s *server) DigestAuth(w http.ResponseWriter, r *http.Request, qop api.QOP, user api.User, password api.Password) {
	s.digestAuth(w, r, qop, user, password, "MD5", "never")
}

func (s *server) DigestAuthAlgorithm(
	w http.ResponseWriter,
	r *http.Request,
	qop api.QOP,
	user api.User,
	password api.Password,
	algorithm api.Algorithm,
) {
	s.digestAuth(w, r, qop, user, password, algorithm, "never")
}

func (s *server) DigestAuthAlgorithmStale(
	w http.ResponseWriter,
	r *http.Request,
	qop api.QOP,
	user api.User,
	password api.Password,
	algorithm api.Algorithm,
	staleAfter api.StaleAfter,
) {
	s.digestAuth(w, r, qop, user, password, algorithm, staleAfter)
}

func (s *server) Bearer(w http.ResponseWriter, r *http.Request) { s.bearer(w, r) }

func (s *server) UUID(w http.ResponseWriter, r *http.Request) { s.uuid(w, r) }

func (s *server) Bytes(w http.ResponseWriter, r *http.Request, _ api.N) { s.bytes(w, r) }

func (s *server) StreamBytes(w http.ResponseWriter, r *http.Request, _ api.N) {
	s.streamBytes(w, r)
}

func (s *server) Stream(w http.ResponseWriter, r *http.Request, _ api.N) { s.stream(w, r) }

func (s *server) Range(w http.ResponseWriter, r *http.Request, _ api.NumBytes) { s.rangeBytes(w, r) }

func (s *server) Delay(w http.ResponseWriter, r *http.Request, _ api.Delay) { s.delay(w, r) }

func (s *server) DelayPost(w http.ResponseWriter, r *http.Request, _ api.Delay) { s.delay(w, r) }

func (s *server) DelayPut(w http.ResponseWriter, r *http.Request, _ api.Delay) { s.delay(w, r) }

func (s *server) DelayPatch(w http.ResponseWriter, r *http.Request, _ api.Delay) { s.delay(w, r) }

func (s *server) DelayDelete(w http.ResponseWriter, r *http.Request, _ api.Delay) { s.delay(w, r) }

func (s *server) DelayTrace(w http.ResponseWriter, r *http.Request, _ api.Delay) { s.delay(w, r) }

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

func (s *server) RedirectToPost(w http.ResponseWriter, r *http.Request) { s.redirectTo(w, r) }

func (s *server) RedirectToPut(w http.ResponseWriter, r *http.Request) { s.redirectTo(w, r) }

func (s *server) RedirectToPatch(w http.ResponseWriter, r *http.Request) { s.redirectTo(w, r) }

func (s *server) RedirectToDelete(w http.ResponseWriter, r *http.Request) { s.redirectTo(w, r) }

func (s *server) RedirectToTrace(w http.ResponseWriter, r *http.Request) { s.redirectTo(w, r) }

func (s *server) Gzip(w http.ResponseWriter, r *http.Request) { s.gzip(w, r) }

func (s *server) Deflate(w http.ResponseWriter, r *http.Request) { s.deflate(w, r) }

func (s *server) Brotli(w http.ResponseWriter, r *http.Request) { s.brotli(w, r) }

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

func (s *server) ImageWEBP(w http.ResponseWriter, r *http.Request) { s.webpImage(w, r) }

func (s *server) Links(w http.ResponseWriter, r *http.Request, _ api.N, _ api.Offset) {
	s.links(w, r)
}
