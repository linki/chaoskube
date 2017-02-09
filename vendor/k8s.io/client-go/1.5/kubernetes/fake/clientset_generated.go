/*
Copyright 2016 The Kubernetes Authors.

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

package fake

import (
	"k8s.io/client-go/1.5/discovery"
	fakediscovery "k8s.io/client-go/1.5/discovery/fake"
	clientset "k8s.io/client-go/1.5/kubernetes"
	v1alpha1apps "k8s.io/client-go/1.5/kubernetes/typed/apps/v1alpha1"
	fakev1alpha1apps "k8s.io/client-go/1.5/kubernetes/typed/apps/v1alpha1/fake"
	v1beta1authentication "k8s.io/client-go/1.5/kubernetes/typed/authentication/v1beta1"
	fakev1beta1authentication "k8s.io/client-go/1.5/kubernetes/typed/authentication/v1beta1/fake"
	v1beta1authorization "k8s.io/client-go/1.5/kubernetes/typed/authorization/v1beta1"
	fakev1beta1authorization "k8s.io/client-go/1.5/kubernetes/typed/authorization/v1beta1/fake"
	v1autoscaling "k8s.io/client-go/1.5/kubernetes/typed/autoscaling/v1"
	fakev1autoscaling "k8s.io/client-go/1.5/kubernetes/typed/autoscaling/v1/fake"
	v1batch "k8s.io/client-go/1.5/kubernetes/typed/batch/v1"
	fakev1batch "k8s.io/client-go/1.5/kubernetes/typed/batch/v1/fake"
	v1alpha1certificates "k8s.io/client-go/1.5/kubernetes/typed/certificates/v1alpha1"
	fakev1alpha1certificates "k8s.io/client-go/1.5/kubernetes/typed/certificates/v1alpha1/fake"
	v1core "k8s.io/client-go/1.5/kubernetes/typed/core/v1"
	fakev1core "k8s.io/client-go/1.5/kubernetes/typed/core/v1/fake"
	v1beta1extensions "k8s.io/client-go/1.5/kubernetes/typed/extensions/v1beta1"
	fakev1beta1extensions "k8s.io/client-go/1.5/kubernetes/typed/extensions/v1beta1/fake"
	v1alpha1policy "k8s.io/client-go/1.5/kubernetes/typed/policy/v1alpha1"
	fakev1alpha1policy "k8s.io/client-go/1.5/kubernetes/typed/policy/v1alpha1/fake"
	v1alpha1rbac "k8s.io/client-go/1.5/kubernetes/typed/rbac/v1alpha1"
	fakev1alpha1rbac "k8s.io/client-go/1.5/kubernetes/typed/rbac/v1alpha1/fake"
	v1beta1storage "k8s.io/client-go/1.5/kubernetes/typed/storage/v1beta1"
	fakev1beta1storage "k8s.io/client-go/1.5/kubernetes/typed/storage/v1beta1/fake"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/apimachinery/registered"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(api.Scheme, api.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o, registered.RESTMapper()))

	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &Clientset{fakePtr}
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return &fakediscovery.FakeDiscovery{Fake: &c.Fake}
}

var _ clientset.Interface = &Clientset{}

// Core retrieves the CoreClient
func (c *Clientset) Core() v1core.CoreInterface {
	return &fakev1core.FakeCore{Fake: &c.Fake}
}

// Apps retrieves the AppsClient
func (c *Clientset) Apps() v1alpha1apps.AppsInterface {
	return &fakev1alpha1apps.FakeApps{Fake: &c.Fake}
}

// Authentication retrieves the AuthenticationClient
func (c *Clientset) Authentication() v1beta1authentication.AuthenticationInterface {
	return &fakev1beta1authentication.FakeAuthentication{Fake: &c.Fake}
}

// Authorization retrieves the AuthorizationClient
func (c *Clientset) Authorization() v1beta1authorization.AuthorizationInterface {
	return &fakev1beta1authorization.FakeAuthorization{Fake: &c.Fake}
}

// Autoscaling retrieves the AutoscalingClient
func (c *Clientset) Autoscaling() v1autoscaling.AutoscalingInterface {
	return &fakev1autoscaling.FakeAutoscaling{Fake: &c.Fake}
}

// Batch retrieves the BatchClient
func (c *Clientset) Batch() v1batch.BatchInterface {
	return &fakev1batch.FakeBatch{Fake: &c.Fake}
}

// Certificates retrieves the CertificatesClient
func (c *Clientset) Certificates() v1alpha1certificates.CertificatesInterface {
	return &fakev1alpha1certificates.FakeCertificates{Fake: &c.Fake}
}

// Extensions retrieves the ExtensionsClient
func (c *Clientset) Extensions() v1beta1extensions.ExtensionsInterface {
	return &fakev1beta1extensions.FakeExtensions{Fake: &c.Fake}
}

// Policy retrieves the PolicyClient
func (c *Clientset) Policy() v1alpha1policy.PolicyInterface {
	return &fakev1alpha1policy.FakePolicy{Fake: &c.Fake}
}

// Rbac retrieves the RbacClient
func (c *Clientset) Rbac() v1alpha1rbac.RbacInterface {
	return &fakev1alpha1rbac.FakeRbac{Fake: &c.Fake}
}

// Storage retrieves the StorageClient
func (c *Clientset) Storage() v1beta1storage.StorageInterface {
	return &fakev1beta1storage.FakeStorage{Fake: &c.Fake}
}
