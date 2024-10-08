// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeServices implements ServiceInterface
type FakeServices struct {
	Fake *FakeCoreV1
	ns   string
}

var servicesResource = v1.SchemeGroupVersion.WithResource("services")

var servicesKind = v1.SchemeGroupVersion.WithKind("Service")

// Get takes name of the service, and returns the corresponding service object, and an error if there is any.
func (c *FakeServices) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Service, err error) {
	emptyResult := &v1.Service{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(servicesResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Service), err
}

// List takes label and field selectors, and returns the list of Services that match those selectors.
func (c *FakeServices) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ServiceList, err error) {
	emptyResult := &v1.ServiceList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(servicesResource, servicesKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.ServiceList{ListMeta: obj.(*v1.ServiceList).ListMeta}
	for _, item := range obj.(*v1.ServiceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested services.
func (c *FakeServices) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(servicesResource, c.ns, opts))

}

// Create takes the representation of a service and creates it.  Returns the server's representation of the service, and an error, if there is any.
func (c *FakeServices) Create(ctx context.Context, service *v1.Service, opts metav1.CreateOptions) (result *v1.Service, err error) {
	emptyResult := &v1.Service{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(servicesResource, c.ns, service, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Service), err
}

// Update takes the representation of a service and updates it. Returns the server's representation of the service, and an error, if there is any.
func (c *FakeServices) Update(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (result *v1.Service, err error) {
	emptyResult := &v1.Service{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(servicesResource, c.ns, service, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Service), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeServices) UpdateStatus(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (result *v1.Service, err error) {
	emptyResult := &v1.Service{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(servicesResource, "status", c.ns, service, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Service), err
}

// Delete takes name of the service and deletes it. Returns an error if one occurs.
func (c *FakeServices) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(servicesResource, c.ns, name, opts), &v1.Service{})

	return err
}

// Patch applies the patch and returns the patched service.
func (c *FakeServices) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Service, err error) {
	emptyResult := &v1.Service{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(servicesResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Service), err
}
