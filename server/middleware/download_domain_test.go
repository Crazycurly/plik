package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func newTestHandler() (http.Handler, *bool) {
	called := false
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}), &called
}

func TestRestrictDownloadDomain_NoDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.True(t, *called, "should pass through when no download domain configured")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestRestrictDownloadDomain_NotOnDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.True(t, *called, "should pass through when not on download domain")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestRestrictDownloadDomain_FileEndpoint(t *testing.T) {
	config := common.NewConfiguration()
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	for _, path := range []string{
		"/file/abc123/def456/test.txt",
		"/stream/abc123/def456/test.txt",
		"/archive/abc123/test.zip",
	} {
		*called = false
		req := httptest.NewRequest("GET", path, nil)
		req.Host = "dl.plik.root.gg"
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		require.True(t, *called, "file endpoint %s should pass through on download domain", path)
		require.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestRestrictDownloadDomain_HealthEndpoint(t *testing.T) {
	config := common.NewConfiguration()
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	req := httptest.NewRequest("GET", "/health", nil)
	req.Host = "dl.plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.True(t, *called, "health endpoint should pass through on download domain")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestRestrictDownloadDomain_RedirectWithPlikDomain(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	req := httptest.NewRequest("GET", "/config", nil)
	req.Host = "dl.plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.False(t, *called, "non-file endpoint should not pass through on download domain")
	require.Equal(t, http.StatusFound, rr.Code)
	require.Equal(t, "https://plik.root.gg/config", rr.Header().Get("Location"))
}

func TestRestrictDownloadDomain_RedirectPreservesPath(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	req := httptest.NewRequest("GET", "/upload/abc123?foo=bar", nil)
	req.Host = "dl.plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.False(t, *called)
	require.Equal(t, http.StatusFound, rr.Code)
	require.Equal(t, "https://plik.root.gg/upload/abc123?foo=bar", rr.Header().Get("Location"))
}

func TestRestrictDownloadDomain_OnlyDownloadDomain_PassesThrough(t *testing.T) {
	config := common.NewConfiguration()
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	// Non-file request on download domain WITHOUT PlikDomain set → pass through (backward compat)
	req := httptest.NewRequest("GET", "/config", nil)
	req.Host = "dl.plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.True(t, *called, "should pass through when PlikDomain is not set (backward compat)")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestRestrictDownloadDomain_Alias(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	config.DownloadDomainAlias = []string{"https://dl2.plik.root.gg"}
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	// Request on alias domain
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "dl2.plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.False(t, *called, "alias domain should also be restricted")
	require.Equal(t, http.StatusFound, rr.Code)
	require.Equal(t, "https://plik.root.gg/", rr.Header().Get("Location"))
}

func TestRestrictDownloadDomain_WebappRoot(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	// Browsing webapp root on download domain
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "dl.plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.False(t, *called, "webapp root should be blocked on download domain")
	require.Equal(t, http.StatusFound, rr.Code)
	require.Equal(t, "https://plik.root.gg/", rr.Header().Get("Location"))
}

func TestRestrictDownloadDomain_RedirectWithPath(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	config.Path = "/sub"
	require.NoError(t, config.Initialize())

	handler, called := newTestHandler()
	middleware := RestrictDownloadDomain(config)(handler)

	// Non-file request on download domain — should redirect to PlikDomain.
	// The request path is already stripped of the prefix by the server's StripPrefix
	// middleware before it reaches the download_domain middleware. So the Location
	// header should be PlikDomain + the already-stripped RequestURI (no double-prefix).
	req := httptest.NewRequest("GET", "/config", nil)
	req.Host = "dl.plik.root.gg"
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	require.False(t, *called, "non-file endpoint should not pass through on download domain")
	require.Equal(t, http.StatusFound, rr.Code)
	require.Equal(t, "https://plik.root.gg/config", rr.Header().Get("Location"),
		"redirect should point to PlikDomain + request path (Path already stripped by StripPrefix middleware)")
}
