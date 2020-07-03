/*
Copyright The Guard Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	auth "github.com/appscode/guard/auth/providers/azure"
	"github.com/appscode/guard/authz"
	"github.com/appscode/guard/authz/providers/azure/data"
	authzOpts "github.com/appscode/guard/authz/providers/azure/options"
	"github.com/appscode/guard/authz/providers/azure/rbac"
	"github.com/appscode/pat"

	"github.com/stretchr/testify/assert"
	authzv1 "k8s.io/api/authorization/v1"
)

const (
	loginResp = `{ "token_type": "Bearer", "expires_in": 8459, "access_token": "%v"}`
)

func clientSetup(serverUrl, mode string) (*Authorizer, error) {
	c := &Authorizer{}
	opts := authzOpts.Options{
		AuthzMode:                      mode,
		ResourceId:                     "resourceId",
		ARMCallLimit:                   2000,
		SkipAuthzCheck:                 []string{"alpha, tango, charlie"},
		AuthzResolveGroupMemberships:   true,
		SkipAuthzForNonAADUsers:        true,
		AllowNonResDiscoveryPathAccess: true,
	}

	authOpts := auth.Options{
		ClientID:     "client_id",
		ClientSecret: "client_secret",
		TenantID:     "tenant_id",
	}

	authzInfo := rbac.AuthzInfo{
		AADEndpoint: serverUrl + "/login/",
		ARMEndPoint: serverUrl + "/arm/",
	}

	c.rbacClient, err = rbac.New(opts, authOpts, &authzInfo)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func serverSetup(loginResp, checkaccessResp string, loginStatus, checkaccessStatus int) (*httptest.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, err
	}

	m := pat.New()

	m.Post("/login/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(loginStatus)
		_, _ = w.Write([]byte(loginResp))
	}))

	m.Post("/arm/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(checkaccessStatus)
		_, _ = w.Write([]byte(checkaccessResp))
	}))

	srv := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: m},
	}
	srv.Start()

	return srv, nil
}

func getServerAndClient(t *testing.T, loginResp, checkaccessResp string) (*httptest.Server, *Authorizer, authz.Store) {
	srv, err := serverSetup(loginResp, checkaccessResp, http.StatusOK, http.StatusOK)
	if err != nil {
		t.Fatalf("Error when creating server, reason: %v", err)
	}

	client, err := clientSetup(srv.URL, "arc")
	if err != nil {
		t.Fatalf("Error when creatidng azure client. reason : %v", err)
	}

	var testOptions = data.Options{
		HardMaxCacheSize:   1,
		Shards:             1,
		LifeWindow:         1 * time.Minute,
		CleanWindow:        1 * time.Minute,
		MaxEntriesInWindow: 10,
		MaxEntrySize:       5,
		Verbose:            false,
	}
	dataStore, _ := data.NewDataStore(testOptions)

	return srv, client, dataStore
}

func TestCheck(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		var validBody = `[{"accessDecision":"Allowed",
		"actionId":"Microsoft.Kubernetes/connectedClusters/pods/delete",
		"isDataAction":true,"roleAssignment":null,"denyAssignment":null,"timeToLiveInMs":300000}]`

		srv, client, store := getServerAndClient(t, loginResp, validBody)
		defer srv.Close()
		defer store.Close()

		request := &authzv1.SubjectAccessReviewSpec{
			User: "beta@bing.com",
			ResourceAttributes: &authzv1.ResourceAttributes{Namespace: "dev", Group: "", Resource: "pods",
				Subresource: "status", Version: "v1", Name: "test", Verb: "delete"}, Extra: map[string]authzv1.ExtraValue{"oid": {"00000000-0000-0000-0000-000000000000"}}}

		resp, err := client.Check(request, store)
		assert.Nilf(t, err, "Should not have got error")
		assert.NotNil(t, resp)
		assert.Equal(t, resp.Allowed, true)
		assert.Equal(t, resp.Denied, false)
	})
}
