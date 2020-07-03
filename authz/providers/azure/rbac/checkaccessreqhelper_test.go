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

package rbac

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
)

func Test_getScope(t *testing.T) {
	type args struct {
		resourceId string
		attr       *authzv1.ResourceAttributes
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"nilAttr", args{"resourceId", nil}, "resourceId"},
		{"bothnil", args{"", nil}, ""},
		{"emptyRes", args{"", &authzv1.ResourceAttributes{Namespace: ""}}, ""},
		{"emptyNS", args{"resourceId", &authzv1.ResourceAttributes{Namespace: ""}}, "resourceId"},
		{"bothPresent", args{"resourceId", &authzv1.ResourceAttributes{Namespace: "test"}}, "resourceId/namespaces/test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getScope(tt.args.resourceId, tt.args.attr); got != tt.want {
				t.Errorf("getScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getValidSecurityGroups(t *testing.T) {
	type args struct {
		groups []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"nilGroup", args{nil}, nil},
		{"emptyGroup", args{[]string{}}, nil},
		{"noGuidGroup", args{[]string{"abc", "def", "system:ghi"}}, nil},
		{"someGroup",
			args{[]string{"abc", "1cffe3ae-93c0-4a87-9484-2e90e682aae9", "sys:admin", "", "0ab7f20f-8e9a-43ba-b5ac-1811c91b3d40"}},
			[]string{"1cffe3ae-93c0-4a87-9484-2e90e682aae9", "0ab7f20f-8e9a-43ba-b5ac-1811c91b3d40"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getValidSecurityGroups(tt.args.groups); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getValidSecurityGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDataAction(t *testing.T) {
	type args struct {
		subRevReq   *authzv1.SubjectAccessReviewSpec
		clusterType string
	}
	tests := []struct {
		name string
		args args
		want AuthorizationActionInfo
	}{
		{"aks", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				NonResourceAttributes: &authzv1.NonResourceAttributes{Path: "/apis", Verb: "list"}}, clusterType: "aks"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "aks/apis/read"}, IsDataAction: true}},

		{"aks2", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				NonResourceAttributes: &authzv1.NonResourceAttributes{Path: "/logs", Verb: "get"}}, clusterType: "aks"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "aks/logs/read"}, IsDataAction: true}},

		{"arc", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "", Resource: "pods", Subresource: "status", Version: "v1", Name: "test", Verb: "delete"}}, clusterType: "arc"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "arc/pods/delete"}, IsDataAction: true}},

		{"arc2", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "apps", Resource: "deployments", Subresource: "status", Version: "v1", Name: "test", Verb: "create"}}, clusterType: "arc"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "arc/apps/deployments/write"}, IsDataAction: true}},

		{"arc3", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "policy", Resource: "podsecuritypolicies", Subresource: "status", Version: "v1", Name: "test", Verb: "use"}}, clusterType: "arc"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "arc/policy/podsecuritypolicies/use/action"}, IsDataAction: true}},

		{"aks3", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "authentication.k8s.io", Resource: "userextras", Subresource: "scopes", Version: "v1", Name: "test", Verb: "impersonate"}}, clusterType: "aks"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "aks/authentication.k8s.io/userextras/impersonate/action"}, IsDataAction: true}},

		{"arc4", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "rbac.authorization.k8s.io", Resource: "clusterroles", Subresource: "status", Version: "v1", Name: "test", Verb: "bind"}}, clusterType: "arc"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "arc/rbac.authorization.k8s.io/clusterroles/bind/action"}, IsDataAction: true}},

		{"aks4", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "rbac.authorization.k8s.io", Resource: "clusterroles", Subresource: "status", Version: "v1", Name: "test", Verb: "escalate"}}, clusterType: "aks"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "aks/rbac.authorization.k8s.io/clusterroles/escalate/action"}, IsDataAction: true}},

		{"arc5", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "scheduling.k8s.io", Resource: "priorityclasses", Subresource: "status", Version: "v1", Name: "test", Verb: "update"}}, clusterType: "arc"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "arc/scheduling.k8s.io/priorityclasses/write"}, IsDataAction: true}},

		{"aks5", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "events.k8s.io", Resource: "events", Subresource: "status", Version: "v1", Name: "test", Verb: "watch"}}, clusterType: "aks"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "aks/events.k8s.io/events/read"}, IsDataAction: true}},

		{"arc6", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "batch", Resource: "cronjobs", Subresource: "status", Version: "v1", Name: "test", Verb: "patch"}}, clusterType: "arc"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "arc/batch/cronjobs/write"}, IsDataAction: true}},

		{"aks6", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authzv1.ResourceAttributes{Group: "certificates.k8s.io", Resource: "certificatesigningrequests", Subresource: "approvals", Version: "v1", Name: "test", Verb: "deletecollection"}}, clusterType: "aks"},
			AuthorizationActionInfo{AuthorizationEntity: AuthorizationEntity{Id: "aks/certificates.k8s.io/certificatesigningrequests/delete"}, IsDataAction: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDataAction(tt.args.subRevReq, tt.args.clusterType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDataAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNameSpaceScope(t *testing.T) {
	req := authzv1.SubjectAccessReviewSpec{ResourceAttributes: nil}
	want := false
	got, str := getNameSpaceScope(&req)
	if got || str != "" {
		t.Errorf("Want:%v, got:%v", want, got)
	}

	req = authzv1.SubjectAccessReviewSpec{
		ResourceAttributes: &authzv1.ResourceAttributes{Namespace: ""}}
	want = false
	got, str = getNameSpaceScope(&req)
	if got || str != "" {
		t.Errorf("Want:%v, got:%v", want, got)
	}

	req = authzv1.SubjectAccessReviewSpec{
		ResourceAttributes: &authzv1.ResourceAttributes{Namespace: "dev"}}
	outputstring := "namespaces/dev"
	want = true
	got, str = getNameSpaceScope(&req)
	if !got || str != outputstring {
		t.Errorf("Want:%v - %s, got: %v - %s", want, outputstring, got, str)
	}
}

func Test_prepareCheckAccessRequestBody(t *testing.T) {
	req := &authzv1.SubjectAccessReviewSpec{Extra: nil}
	resouceId := "resourceId"
	clusterType := "aks"
	var want *CheckAccessRequest = nil
	wantErr := errors.New("oid info not sent from authenticatoin module")

	got, gotErr := prepareCheckAccessRequestBody(req, clusterType, resouceId, true)

	if got != want && gotErr != wantErr {
		t.Errorf("Want:%v WantErr:%v, got:%v, gotErr:%v", want, wantErr, got, gotErr)
	}

	req = &authzv1.SubjectAccessReviewSpec{Extra: map[string]authzv1.ExtraValue{"oid": {"test"}}}
	resouceId = "resourceId"
	clusterType = "arc"
	want = nil
	wantErr = errors.New("oid info sent from authenticatoin module is not valid")

	got, gotErr = prepareCheckAccessRequestBody(req, clusterType, resouceId, true)

	if got != want && gotErr != wantErr {
		t.Errorf("Want:%v WantErr:%v, got:%v, gotErr:%v", want, wantErr, got, gotErr)
	}
}

func Test_getResultCacheKey(t *testing.T) {
	type args struct {
		subRevReq *authzv1.SubjectAccessReviewSpec
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"aks", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				User:                  "charlie@yahoo.com",
				NonResourceAttributes: &authzv1.NonResourceAttributes{Path: "/apis/v1", Verb: "list"}}},
			"charlie@yahoo.com/apis/v1/read"},

		{"aks", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				User:                  "echo@outlook.com",
				NonResourceAttributes: &authzv1.NonResourceAttributes{Path: "/logs", Verb: "get"}}},
			"echo@outlook.com/logs/read"},

		{"aks", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				User: "alpha@bing.com",
				ResourceAttributes: &authzv1.ResourceAttributes{Namespace: "dev", Group: "", Resource: "pods",
					Subresource: "status", Version: "v1", Name: "test", Verb: "delete"}}},
			"alpha@bing.com/dev/pods/delete"},

		{"arc", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				User: "beta@msn.com",
				ResourceAttributes: &authzv1.ResourceAttributes{Namespace: "azure-arc",
					Group: "authentication.k8s.io", Resource: "userextras", Subresource: "scopes", Version: "v1",
					Name: "test", Verb: "impersonate"}}},
			"beta@msn.com/azure-arc/authentication.k8s.io/userextras/impersonate/action"},

		{"arc", args{
			subRevReq: &authzv1.SubjectAccessReviewSpec{
				User: "beta@msn.com",
				ResourceAttributes: &authzv1.ResourceAttributes{Namespace: "", Group: "", Resource: "nodes",
					Subresource: "scopes", Version: "v1", Name: "", Verb: "list"}}},
			"beta@msn.com/nodes/read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getResultCacheKey(tt.args.subRevReq); got != tt.want {
				t.Errorf("getResultCacheKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
