package corehttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/ipfs/go-ipfs/repo/config"
)

type testcasecheckversion struct {
	userAgent    string
	shouldHandle bool
	responseBody string
	responseCode int
}

func (tc testcasecheckversion) body() string {
	if !tc.shouldHandle && tc.responseBody == "" {
		return fmt.Sprintf("%s (%s != %s)\n", errApiVersionMismatch, config.ApiVersion, tc.userAgent)
	}

	return tc.responseBody
}

func TestCheckVersionOption(t *testing.T) {
	tcs := []testcasecheckversion{
		{"/go-ipfs/0.1/", false, "", http.StatusBadRequest},
		{config.ApiVersion, true, "all tests pass", http.StatusOK},
		{"Mozilla Firefox/no go ipfs note", true, "all tests pass", http.StatusOK},
	}

	for _, tc := range tcs {
		r := httptest.NewRequest("POST", "/test", nil)
		r.Header.Add("User-Agent", tc.userAgent) // old version, should fail

		called := false
		inner := http.NewServeMux()
		inner.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			called = true
			if !tc.shouldHandle {
				t.Error("handler was called even though version didn't match")
			} else {
				io.WriteString(w, "all tests pass")
			}
		})

		mux, err := CheckVersionOption()(nil, nil, inner)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if tc.shouldHandle && !called {
			t.Error("handler wasn't called even though version matched")
		}

		if w.Code != tc.responseCode {
			t.Errorf("expected code %d but got %d", tc.responseCode, w.Code)
		}

		if w.Body.String() != tc.body() {
			t.Errorf("expected error message %q, got %q", tc.body(), w.Body.String())
		}
	}
}
