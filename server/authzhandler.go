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
package server

import (
	"net/http"
	"strings"

	"github.com/appscode/guard/authz"
	"github.com/appscode/guard/authz/providers/azure"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
)

type Authzhandler struct {
	AuthRecommendedOptions  *AuthRecommendedOptions
	AuthzRecommendedOptions *AuthzRecommendedOptions
	Store                   authz.Store
}

func (s *Authzhandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
		writeAuthzResponse(w, nil, nil, WithCode(errors.New("Missing client certificate"), http.StatusBadRequest))
		return
	}
	crt := req.TLS.PeerCertificates[0]
	if len(crt.Subject.Organization) == 0 {
		writeAuthzResponse(w, nil, nil, WithCode(errors.New("Client certificate is missing organization"), http.StatusBadRequest))
		return
	}
	org := crt.Subject.Organization[0]
	glog.Infof("Received subject access review request for %s/%s", org, crt.Subject.CommonName)

	data := authzv1.SubjectAccessReview{}
	err := json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		writeAuthzResponse(w, nil, nil, WithCode(errors.Wrap(err, "Failed to parse request"), http.StatusBadRequest))
		return
	}

	binaryData, _ := json.MarshalIndent(&data, "", "    ")
	glog.V(10).Infof("Authz req:%s", binaryData)

	if !s.AuthzRecommendedOptions.AuthzProvider.Has(org) {
		writeAuthzResponse(w, &data.Spec, nil, WithCode(errors.Errorf("guard does not provide service for %v", org), http.StatusBadRequest))
		return
	}

	client, err := s.getAuthzProviderClient(org, crt.Subject.CommonName)
	if err != nil {
		writeAuthzResponse(w, &data.Spec, nil, err)
		return
	}

	resp, err := client.Check(&data.Spec)
	writeAuthzResponse(w, &data.Spec, resp, err)
}

func (s *Authzhandler) getAuthzProviderClient(org, commonName string) (authz.Interface, error) {
	switch strings.ToLower(org) {
	case azure.OrgType:
		return azure.New(s.AuthzRecommendedOptions.Azure, s.AuthRecommendedOptions.Azure, s.Store)
	}

	return nil, errors.Errorf("Client is using unknown organization %s", org)
}
