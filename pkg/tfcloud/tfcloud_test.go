package tfcloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateCreatesOrPatchesEnvVar(t *testing.T) {
	const description = "Auto-Injected by vault-token-injector"

	tests := []struct {
		name       string
		list       string
		wantMethod string
		wantPath   string
		wantKey    bool
	}{
		{
			name:       "updates existing",
			list:       `{"data":[{"id":"var-1","type":"vars","attributes":{"key":"VAULT_TOKEN","category":"env","sensitive":true}}]}`,
			wantMethod: http.MethodPatch,
			wantPath:   "/api/v2/workspaces/ws-1/vars/var-1",
		},
		{
			name: "updates env when a terraform var shares the key",
			list: `{"data":[` +
				`{"id":"var-tf","type":"vars","attributes":{"key":"VAULT_TOKEN","category":"terraform"}},` +
				`{"id":"var-env","type":"vars","attributes":{"key":"VAULT_TOKEN","category":"env","sensitive":true}}` +
				`]}`,
			wantMethod: http.MethodPatch,
			wantPath:   "/api/v2/workspaces/ws-1/vars/var-env",
		},
		{
			name:       "creates env when only a terraform var exists",
			list:       `{"data":[{"id":"var-tf","type":"vars","attributes":{"key":"VAULT_TOKEN","category":"terraform"}}]}`,
			wantMethod: http.MethodPost,
			wantPath:   "/api/v2/workspaces/ws-1/vars",
			wantKey:    true,
		},
		{
			name:       "creates missing",
			list:       `{"data":[]}`,
			wantMethod: http.MethodPost,
			wantPath:   "/api/v2/workspaces/ws-1/vars",
			wantKey:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath, gotListPath, gotAuth string
			var gotBody map[string]any

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				if r.Method == http.MethodGet {
					gotListPath = r.URL.Path
					gotAuth = r.Header.Get("Authorization")
					w.Write([]byte(tt.list))
					return
				}
				gotMethod = r.Method
				gotPath = r.URL.Path
				raw, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(raw, &gotBody); err != nil {
					t.Errorf("decode body: %v", err)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{"id":"var-1","type":"vars","attributes":{"key":"VAULT_TOKEN"}}}`))
			}))
			t.Cleanup(srv.Close)

			orig := tfeAddress
			tfeAddress = srv.URL
			t.Cleanup(func() { tfeAddress = orig })

			err := Variable{
				Workspace: "ws-1",
				Key:       "VAULT_TOKEN",
				Value:     "s.secret",
				Token:     "test-token",
				Sensitive: true,
			}.Update()
			if err != nil {
				t.Fatal(err)
			}
			if gotListPath != "/api/v2/workspaces/ws-1/vars" || gotAuth != "Bearer test-token" {
				t.Fatalf("list = %s auth %q", gotListPath, gotAuth)
			}
			if gotMethod != tt.wantMethod || gotPath != tt.wantPath {
				t.Fatalf("request = %s %s, want %s %s", gotMethod, gotPath, tt.wantMethod, tt.wantPath)
			}

			data := gotBody["data"].(map[string]any)
			if data["type"] != "vars" {
				t.Fatalf("type = %v", data["type"])
			}
			attrs := data["attributes"].(map[string]any)
			if attrs["value"] != "s.secret" || attrs["description"] != description || attrs["sensitive"] != true {
				t.Fatalf("attributes = %#v", attrs)
			}
			if _, ok := attrs["key"]; ok != tt.wantKey {
				t.Fatalf("key present = %v, want %v; attributes = %#v", ok, tt.wantKey, attrs)
			}
			if tt.wantKey && attrs["category"] != "env" {
				t.Fatalf("category = %v", attrs["category"])
			}
		})
	}
}

func TestUpdateRetriesRateLimit(t *testing.T) {
	var gets int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if r.Method == http.MethodGet {
			gets++
			if gets == 1 {
				w.Header().Set("Retry-After", "0.01")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.Write([]byte(`{"data":[]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"id":"var-1","type":"vars","attributes":{"key":"VAULT_TOKEN"}}}`))
	}))
	t.Cleanup(srv.Close)

	orig := tfeAddress
	tfeAddress = srv.URL
	t.Cleanup(func() { tfeAddress = orig })

	err := Variable{
		Workspace: "ws-1",
		Key:       "VAULT_TOKEN",
		Value:     "s.secret",
		Token:     "test-token",
		Sensitive: true,
	}.Update()
	if err != nil {
		t.Fatal(err)
	}
	if gets != 2 {
		t.Fatalf("list attempts = %d, want 2", gets)
	}
}

func TestUpdateFindsEnvVarOnLaterPage(t *testing.T) {
	var pages []string
	var gotMethod, gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if r.Method == http.MethodGet {
			page := r.URL.Query().Get("page[number]")
			pages = append(pages, page)
			if r.URL.Path != "/api/v2/workspaces/ws-1/vars" || r.URL.Query().Get("page[size]") != "100" {
				t.Errorf("list = %s?%s", r.URL.Path, r.URL.RawQuery)
			}
			if page == "1" {
				var b strings.Builder
				b.WriteString(`{"data":[`)
				b.WriteString(`{"id":"var-tf","type":"vars","attributes":{"key":"VAULT_TOKEN","category":"terraform"}}`)
				for i := 1; i < varPageSize; i++ {
					fmt.Fprintf(&b, `,{"id":"var-%d","type":"vars","attributes":{"key":"OTHER_%d","category":"env"}}`, i, i)
				}
				b.WriteString(`]}`)
				w.Write([]byte(b.String()))
				return
			}
			w.Write([]byte(`{"data":[{"id":"var-env","type":"vars","attributes":{"key":"VAULT_TOKEN","category":"env","sensitive":true}}]}`))
			return
		}
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"id":"var-env","type":"vars","attributes":{"key":"VAULT_TOKEN"}}}`))
	}))
	t.Cleanup(srv.Close)

	orig := tfeAddress
	tfeAddress = srv.URL
	t.Cleanup(func() { tfeAddress = orig })

	err := Variable{
		Workspace: "ws-1",
		Key:       "VAULT_TOKEN",
		Value:     "s.secret",
		Token:     "test-token",
		Sensitive: true,
	}.Update()
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || pages[0] != "1" || pages[1] != "2" {
		t.Fatalf("pages = %#v", pages)
	}
	if gotMethod != http.MethodPatch || gotPath != "/api/v2/workspaces/ws-1/vars/var-env" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	data := gotBody["data"].(map[string]any)
	attrs := data["attributes"].(map[string]any)
	if attrs["value"] != "s.secret" {
		t.Fatalf("attributes = %#v", attrs)
	}
	if _, ok := attrs["key"]; ok {
		t.Fatalf("key present in patch: %#v", attrs)
	}
}
