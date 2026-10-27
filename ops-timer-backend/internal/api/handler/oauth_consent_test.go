package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/service"
)

func TestOAuthConsentPreservesSameOriginFormSubmission(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusUnauthorized} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "https://tasks.example/oauth/authorize", nil)
		oauthHeaders(c)
		h := &OAuthHandler{}
		h.renderConsent(c, &service.OAuthAuthorization{
			RedirectURI: "https://client.example/callback",
			Resource:    "https://tasks.example/mcp",
			Scope:       "data:read",
		}, "signed-consent", "Client", "", status)
		if w.Code != status || w.Header().Get("Referrer-Policy") != "same-origin" {
			t.Fatalf("consent page must preserve the POST Origin, including login retries: status=%d headers=%v", w.Code, w.Header())
		}
		if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "form-action 'self'") {
			t.Fatal("consent response lost its cache or form restrictions")
		}
		if !strings.Contains(w.Body.String(), `form action="/oauth/authorize" method="post"`) {
			t.Fatal("consent must use a same-origin native POST form")
		}
	}

	// Non-page OAuth responses continue to suppress referrers entirely.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	oauthHeaders(c)
	if w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("changed referrer policy outside consent pages")
	}
}
