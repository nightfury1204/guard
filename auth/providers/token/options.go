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
package token

import (
	"io/ioutil"

	"github.com/appscode/go/types"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Options struct {
	AuthFile string
}

func NewOptions() Options {
	return Options{}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.AuthFile, "token-auth-file", "", "To enable static token authentication")
}

func (o *Options) Validate() []error {
	var errs []error
	if o.AuthFile == "" {
		errs = append(errs, errors.New("token-auth-file must be non-empty"))
	}
	return errs
}

func (o Options) Apply(d *apps.Deployment) (extraObjs []runtime.Object, err error) {
	container := d.Spec.Template.Spec.Containers[0]

	// create auth secret
	_, err = LoadTokenFile(o.AuthFile)
	if err != nil {
		return nil, err
	}
	tokens, err := ioutil.ReadFile(o.AuthFile)
	if err != nil {
		return nil, err
	}
	authSecret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "guard-token-auth",
			Namespace: d.Namespace,
			Labels:    d.Labels,
		},
		Data: map[string][]byte{
			"token.csv": tokens,
		},
	}
	extraObjs = append(extraObjs, authSecret)

	// mount auth secret into deployment
	volMount := core.VolumeMount{
		Name:      authSecret.Name,
		MountPath: "/etc/guard/auth/token",
	}
	container.VolumeMounts = append(container.VolumeMounts, volMount)

	vol := core.Volume{
		Name: authSecret.Name,
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName:  authSecret.Name,
				DefaultMode: types.Int32P(0555),
			},
		},
	}
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, vol)

	// use auth secret in container[0] args
	if o.AuthFile != "" {
		container.Args = append(container.Args, "--token-auth-file=/etc/guard/auth/token/token.csv")
	}
	d.Spec.Template.Spec.Containers[0] = container

	return extraObjs, nil
}
