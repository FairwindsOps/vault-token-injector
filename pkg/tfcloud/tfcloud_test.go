package tfcloud

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
			name:       "creates missing",
			list:       `{"data":[]}`,
			wantMethod: http.MethodPost,
			wantPath:   "/api/v2/workspaces/ws-1/vars",
			wantKey:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody map[string]any

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				if r.Method == http.MethodGet {
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
