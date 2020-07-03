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

package installer

import (
	"github.com/appscode/guard/auth"
	"github.com/appscode/guard/auth/providers"
	"github.com/appscode/guard/auth/providers/azure"
	"github.com/appscode/guard/auth/providers/github"
	"github.com/appscode/guard/auth/providers/gitlab"
	"github.com/appscode/guard/auth/providers/google"
	"github.com/appscode/guard/auth/providers/ldap"
	"github.com/appscode/guard/auth/providers/token"
	authz "github.com/appscode/guard/authz/providers"
	azureauthz "github.com/appscode/guard/authz/providers/azure"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthOptions struct {
	PkiDir          string
	Namespace       string
	Addr            string
	RunOnMaster     bool
	PrivateRegistry string
	imagePullSecret string

	AuthProvider providers.AuthProviders
	Token        token.Options
	Google       google.Options
	Azure        azure.Options
	LDAP         ldap.Options
	Github       github.Options
	Gitlab       gitlab.Options
}

type AuthzOptions struct {
	AuthzProvider authz.AuthzProviders
	Azure         azureauthz.Options
}

func NewAuthOptions() AuthOptions {
	return AuthOptions{
		PkiDir:          auth.DefaultDataDir,
		Namespace:       metav1.NamespaceSystem,
		Addr:            "10.96.10.96:443",
		PrivateRegistry: "appscode",
		RunOnMaster:     true,
		Token:           token.NewOptions(),
		Google:          google.NewOptions(),
		Azure:           azure.NewOptions(),
		LDAP:            ldap.NewOptions(),
		Github:          github.NewOptions(),
		Gitlab:          gitlab.NewOptions(),
	}
}

func NewAuthzOptions() AuthzOptions {
	return AuthzOptions{
		Azure: azureauthz.NewOptions(),
	}
}

func (o *AuthOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.PkiDir, "pki-dir", o.PkiDir, "Path to directory where pki files are stored.")
	fs.StringVarP(&o.Namespace, "namespace", "n", o.Namespace, "Name of Kubernetes namespace used to run guard server.")
	fs.StringVar(&o.Addr, "addr", o.Addr, "Address (host:port) of guard server.")
	fs.BoolVar(&o.RunOnMaster, "run-on-master", o.RunOnMaster, "If true, runs Guard server on master instances")
	fs.StringVar(&o.PrivateRegistry, "private-registry", o.PrivateRegistry, "Private Docker registry")
	fs.StringVar(&o.imagePullSecret, "image-pull-secret", o.imagePullSecret, "Name of image pull secret")
	o.AuthProvider.AddFlags(fs)
	o.Token.AddFlags(fs)
	o.Google.AddFlags(fs)
	o.Azure.AddFlags(fs)
	o.LDAP.AddFlags(fs)
	o.Github.AddFlags(fs)
	o.Gitlab.AddFlags(fs)
}

func (o *AuthzOptions) AddFlags(fs *pflag.FlagSet) {
	o.AuthzProvider.AddFlags(fs)
	o.Azure.AddFlags(fs)
}
func (o *AuthOptions) Validate() []error {
	var errs []error
	errs = append(errs, o.AuthProvider.Validate()...)

	if o.AuthProvider.Has(token.OrgType) {
		errs = append(errs, o.Token.Validate()...)
	}
	if o.AuthProvider.Has(google.OrgType) {
		errs = append(errs, o.Google.Validate()...)
	}
	if o.AuthProvider.Has(azure.OrgType) {
		errs = append(errs, o.Azure.Validate()...)
	}
	if o.AuthProvider.Has(ldap.OrgType) {
		errs = append(errs, o.LDAP.Validate()...)
	}
	if o.AuthProvider.Has(github.OrgType) {
		errs = append(errs, o.Github.Validate()...)
	}
	if o.AuthProvider.Has(gitlab.OrgType) {
		errs = append(errs, o.Gitlab.Validate()...)
	}

	return errs
}

func (o *AuthzOptions) Validate(opt *AuthOptions) []error {
	var errs []error
	errs = append(errs, o.AuthzProvider.Validate()...)

	if o.AuthzProvider.Has(azureauthz.OrgType) {
		errs = append(errs, o.Azure.Validate(opt.Azure)...)
	}

	return errs
}
