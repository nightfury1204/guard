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
	"fmt"

	stringz "github.com/appscode/go/strings"
	"github.com/appscode/go/types"
	v "github.com/appscode/go/version"
	"github.com/appscode/guard/auth/providers/azure"
	"github.com/appscode/guard/auth/providers/github"
	"github.com/appscode/guard/auth/providers/gitlab"
	"github.com/appscode/guard/auth/providers/google"
	"github.com/appscode/guard/auth/providers/ldap"
	"github.com/appscode/guard/auth/providers/token"
	azureauthz "github.com/appscode/guard/authz/providers/azure"
	"github.com/appscode/guard/server"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newDeployment(authopts AuthOptions, authzopts AuthzOptions) (objects []runtime.Object, err error) {
	d := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "guard",
			Namespace: authopts.Namespace,
			Labels:    labels,
		},
		Spec: apps.DeploymentSpec{
			Replicas: types.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"scheduler.alpha.kubernetes.io/critical-pod": "",
					},
				},
				Spec: core.PodSpec{
					ServiceAccountName: "guard",
					Containers: []core.Container{
						{
							Name:  "guard",
							Image: fmt.Sprintf("%s/guard:%v", authopts.PrivateRegistry, stringz.Val(v.Version.Version, "canary")),
							Args: []string{
								"run",
								"--v=3",
							},
							Ports: []core.ContainerPort{
								{
									ContainerPort: server.ServingPort,
								},
							},
							ReadinessProbe: &core.Probe{
								Handler: core.Handler{
									HTTPGet: &core.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(server.ServingPort),
										Scheme: core.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: int32(30),
							},
							LivenessProbe: &core.Probe{
								Handler: core.Handler{
									HTTPGet: &core.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(server.ServingPort),
										Scheme: core.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: int32(30),
							},
						},
					},
					Tolerations: []core.Toleration{
						{
							Key:      "CriticalAddonsOnly",
							Operator: core.TolerationOpExists,
						},
					},
				},
			},
		},
	}
	if authopts.imagePullSecret != "" {
		d.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
			{
				Name: authopts.imagePullSecret,
			},
		}
	}
	if authopts.RunOnMaster {
		d.Spec.Template.Spec.NodeSelector = map[string]string{
			"node-role.kubernetes.io/master": "",
		}
		d.Spec.Template.Spec.Tolerations = append(d.Spec.Template.Spec.Tolerations, core.Toleration{
			Key:      "node-role.kubernetes.io/master",
			Operator: core.TolerationOpExists,
			Effect:   core.TaintEffectNoSchedule,
		})
	}
	objects = append(objects, d)

	servingOpts := server.NewSecureServingOptionsFromDir(authopts.PkiDir)
	if extras, err := servingOpts.Apply(d); err != nil {
		return nil, err
	} else {
		objects = append(objects, extras...)
	}

	if extras, err := authopts.AuthProvider.Apply(d); err != nil {
		return nil, err
	} else {
		objects = append(objects, extras...)
	}

	if authopts.AuthProvider.Has(token.OrgType) {
		if extras, err := authopts.Token.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	if authopts.AuthProvider.Has(google.OrgType) {
		if extras, err := authopts.Google.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	if authopts.AuthProvider.Has(azure.OrgType) {
		if extras, err := authopts.Azure.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	if authopts.AuthProvider.Has(ldap.OrgType) {
		if extras, err := authopts.LDAP.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	if authopts.AuthProvider.Has(github.OrgType) {
		if extras, err := authopts.Github.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	if authopts.AuthProvider.Has(gitlab.OrgType) {
		if extras, err := authopts.Gitlab.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	if len(authzopts.AuthzProvider.Providers) > 0 {
		if extras, err := authzopts.AuthzProvider.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	if authzopts.AuthzProvider.Has(azureauthz.OrgType) {
		if extras, err := authzopts.Azure.Apply(d); err != nil {
			return nil, err
		} else {
			objects = append(objects, extras...)
		}
	}

	return
}
