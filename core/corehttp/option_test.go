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
	uri          string
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
		{"/go-ipfs/0.1/", "/test/", false, "", http.StatusBadRequest},
		{"/go-ipfs/0.1/", "/version", true, "all tests pass", http.StatusOK},
		{config.ApiVersion, "/test", true, "all tests pass", http.StatusOK},
		{"Mozilla Firefox/no go-ipfs node", "/test", true, "all tests pass", http.StatusOK},
	}

	for _, tc := range tcs {
		t.Logf("%#v", tc)
		r := httptest.NewRequest("POST", tc.uri, nil)
		r.Header.Add("User-Agent", tc.userAgent) // old version, should fail

		called := false
		inner := http.NewServeMux()
		hfunc := func(w http.ResponseWriter, r *http.Request) {
			called = true
			if !tc.shouldHandle {
				t.Error("handler was called even though version didn't match")
			} else {
				io.WriteString(w, "all tests pass")
			}
		}
		inner.HandleFunc("/version", hfunc)
		inner.HandleFunc("/test", hfunc)

		mux, err := CheckVersionOption()(nil, nil, inner)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)

		if tc.shouldHandle && !called {
			t.Error("handler wasn't called even though it should have")
		}

		if w.Code != tc.responseCode {
			t.Errorf("expected code %d but got %d", tc.responseCode, w.Code)
		}

		if w.Body.String() != tc.body() {
			t.Errorf("expected error message %q, got %q", tc.body(), w.Body.String())
		}
	}
}

func TestServerNameOption(t *testing.T) {
	type testcase struct {
		name string
	}

	tcs := []testcase{
		{"go-ipfs/0.4.13"},
		{"go-ipfs/" + config.CurrentVersionNumber},
	}

	assert := func(name string, exp, got interface{}) {
		if got != exp {
			t.Errorf("%s: got %q, expected %q", name, got, exp)
		}
	}

	for _, tc := range tcs {
		t.Logf("%#v", tc)
		r := httptest.NewRequest("POST", "/", nil)

		inner := http.NewServeMux()
		inner.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// this block is intentionally left blank.
		})

		mux, err := ServerNameOption(tc.name)(nil, nil, inner)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		mux.ServeHTTP(w, r)
		srvHdr := w.Header().Get("Server")
		assert("Server header", tc.name, srvHdr)
	}
}
