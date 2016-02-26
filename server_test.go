package generpc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvalidContentType(t *testing.T) {
	r, err := http.NewRequest("POST", "/", strings.NewReader("invalid request"))
	r.Header.Add("Content-Type", "invalid/type")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	NewServer().ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)

	want := `media type "invalid/type" is not supported` + "\n"
	assert.Equal(t, want, w.Body.String())
}
