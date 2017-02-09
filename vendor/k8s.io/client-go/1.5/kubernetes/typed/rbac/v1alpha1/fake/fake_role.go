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
	api "k8s.io/client-go/1.5/pkg/api"
	unversioned "k8s.io/client-go/1.5/pkg/api/unversioned"
	v1alpha1 "k8s.io/client-go/1.5/pkg/apis/rbac/v1alpha1"
	labels "k8s.io/client-go/1.5/pkg/labels"
	watch "k8s.io/client-go/1.5/pkg/watch"
	testing "k8s.io/client-go/1.5/testing"
)

// FakeRoles implements RoleInterface
type FakeRoles struct {
	Fake *FakeRbac
	ns   string
}

var rolesResource = unversioned.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1alpha1", Resource: "roles"}

func (c *FakeRoles) Create(role *v1alpha1.Role) (result *v1alpha1.Role, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(rolesResource, c.ns, role), &v1alpha1.Role{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Role), err
}

func (c *FakeRoles) Update(role *v1alpha1.Role) (result *v1alpha1.Role, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(rolesResource, c.ns, role), &v1alpha1.Role{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Role), err
}

func (c *FakeRoles) Delete(name string, options *api.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(rolesResource, c.ns, name), &v1alpha1.Role{})

	return err
}

func (c *FakeRoles) DeleteCollection(options *api.DeleteOptions, listOptions api.ListOptions) error {
	action := testing.NewDeleteCollectionAction(rolesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.RoleList{})
	return err
}

func (c *FakeRoles) Get(name string) (result *v1alpha1.Role, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(rolesResource, c.ns, name), &v1alpha1.Role{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Role), err
}

func (c *FakeRoles) List(opts api.ListOptions) (result *v1alpha1.RoleList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(rolesResource, c.ns, opts), &v1alpha1.RoleList{})

	if obj == nil {
		return nil, err
	}

	label := opts.LabelSelector
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.RoleList{}
	for _, item := range obj.(*v1alpha1.RoleList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested roles.
func (c *FakeRoles) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(rolesResource, c.ns, opts))

}

// Patch applies the patch and returns the patched role.
func (c *FakeRoles) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1alpha1.Role, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(rolesResource, c.ns, name, data, subresources...), &v1alpha1.Role{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Role), err
}
