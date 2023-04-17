/*
Copyright (C) 2022-2023 Traefik Labs

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetUserGroups(t *testing.T) {
	wantGroups := []string{"group-1", "group-2"}

	mux := http.NewServeMux()
	mux.HandleFunc("/users/test@example.com/groups", func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(rw, fmt.Sprintf("unexpected method: %s", req.Method), http.StatusMethodNotAllowed)
			return
		}

		if req.Header.Get("Authorization") != "Bearer "+testToken {
			http.Error(rw, "Invalid token", http.StatusUnauthorized)
			return
		}

		rw.WriteHeader(http.StatusOK)
		err := json.NewEncoder(rw).Encode(wantGroups)
		require.NoError(t, err)
	})

	srv := httptest.NewServer(mux)

	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, testToken)
	require.NoError(t, err)
	c.httpClient = srv.Client()

	gotGroups, err := c.GetUserGroups(context.Background(), "test@example.com")
	require.NoError(t, err)

	assert.Equal(t, wantGroups, gotGroups)
}
