package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Link(t *testing.T) {
	tests := []struct {
		desc             string
		returnStatusCode int
		wantClusterID    string
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			desc:             "cluster successfully linked",
			returnStatusCode: http.StatusOK,
			wantClusterID:    "1",
			wantErr:          assert.NoError,
		},
		{
			desc:             "failed to link cluster",
			returnStatusCode: http.StatusTeapot,
			wantErr:          assert.Error,
			wantClusterID:    "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			var callCount int

			mux := http.NewServeMux()
			mux.HandleFunc("/link", func(rw http.ResponseWriter, req *http.Request) {
				callCount++

				if req.Method != http.MethodPost {
					http.Error(rw, fmt.Sprintf("unsupported to method: %s", req.Method), http.StatusMethodNotAllowed)
					return
				}

				if req.Header.Get("Authorization") != "Bearer 123" {
					http.Error(rw, "Invalid token", http.StatusUnauthorized)
					return
				}

				b, err := io.ReadAll(req.Body)
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
					return
				}

				if !bytes.Equal([]byte(`{"kubeId":"1"}`), b) {
					http.Error(rw, fmt.Sprintf("invalid body: %s", string(b)), http.StatusBadRequest)
					return
				}

				rw.WriteHeader(test.returnStatusCode)
				_, _ = rw.Write([]byte(`{"clusterId":"1"}`))
			})

			srv := httptest.NewServer(mux)

			t.Cleanup(srv.Close)

			c := NewClient(srv.URL, "123")
			c.httpClient = srv.Client()

			neoClusterID, err := c.Link(context.Background(), "1")
			test.wantErr(t, err)

			require.Equal(t, 1, callCount)

			assert.Equal(t, test.wantClusterID, neoClusterID)
		})
	}
}

func TestClient_GetConfig(t *testing.T) {
	tests := []struct {
		desc             string
		returnStatusCode int
		wantConfig       Config
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			desc:             "get config succeeds",
			returnStatusCode: http.StatusOK,
			wantConfig: Config{
				Topology: TopologyConfig{
					GitProxyHost: "git-proxy-host",
					GitOrgName:   "git-org-name",
					GitRepoName:  "git-repo-name",
				},
				Metrics: MetricsConfig{
					Interval: time.Minute,
					Tables:   []string{"1m", "10m"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			desc:             "get config fails",
			returnStatusCode: http.StatusTeapot,
			wantConfig:       Config{},
			wantErr:          assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			var callCount int

			mux := http.NewServeMux()
			mux.HandleFunc("/config", func(rw http.ResponseWriter, req *http.Request) {
				callCount++

				if req.Method != http.MethodGet {
					http.Error(rw, fmt.Sprintf("unsupported to method: %s", req.Method), http.StatusMethodNotAllowed)
					return
				}

				if req.Header.Get("Authorization") != "Bearer 123" {
					http.Error(rw, "Invalid token", http.StatusUnauthorized)
					return
				}

				rw.WriteHeader(test.returnStatusCode)
				_ = json.NewEncoder(rw).Encode(test.wantConfig)
			})

			srv := httptest.NewServer(mux)

			t.Cleanup(srv.Close)

			c := NewClient(srv.URL, "123")
			c.httpClient = srv.Client()

			agentCfg, err := c.GetConfig(context.Background())
			test.wantErr(t, err)

			require.Equal(t, 1, callCount)

			assert.Equal(t, test.wantConfig, agentCfg)
		})
	}
}