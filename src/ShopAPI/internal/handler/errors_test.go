package handler

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"shopapi/internal/service"
)

func TestErrorResponsesDoNotLeakDatabaseDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		err    error
		status int
	}{
		{errors.New("private database connection and SQL details"), 500},
		{service.ErrConflict, 409},
	} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		respondError(c, tc.err)
		if recorder.Code != tc.status || strings.Contains(recorder.Body.String(), "private") {
			t.Fatalf("unsafe error response: %d %s", recorder.Code, recorder.Body)
		}
	}
}
