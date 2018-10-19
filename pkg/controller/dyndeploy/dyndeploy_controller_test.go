/*
Copyright 2018 Travix International.

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

package dyndeploy

import (
	"testing"
	"time"

	dyndeployv1beta1 "github.com/Travix-International/dynamic-deploy/pkg/apis/dyndeploy/v1beta1"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	instance := &dyndeployv1beta1.DynDeploy{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Spec:       dyndeployv1beta1.DynDeploySpec{Replicas: 1, Image: "nginx", Keys: []string{"ct.nl", "bua.com"}}}

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())
	defer close(StartTestManager(mgr, g))

	// Create the DynDeploy object and expect the Reconcile and Deployment to be created
	err = c.Create(context.TODO(), instance)
	// The instance object may not be a valid object because it might be missing some required fields.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	// Deploys
	deployCtnl := &appsv1.Deployment{}
	deployBuacom := &appsv1.Deployment{}
	var depKeyCtnl = types.NamespacedName{Name: "foo-ctnl", Namespace: "default"}
	var depKeyBuacom = types.NamespacedName{Name: "foo-buacom", Namespace: "default"}
	g.Eventually(func() error { return c.Get(context.TODO(), depKeyCtnl, deployCtnl) }, timeout).
		Should(gomega.Succeed())
	g.Eventually(func() error { return c.Get(context.TODO(), depKeyBuacom, deployBuacom) }, timeout).
		Should(gomega.Succeed())

	// Delete the Deployments and expect Reconcile to be called for Deployment deletion
	g.Expect(c.Delete(context.TODO(), deployCtnl)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Delete(context.TODO(), deployBuacom)).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	g.Eventually(func() error { return c.Get(context.TODO(), depKeyCtnl, deployCtnl) }, timeout).
		Should(gomega.Succeed())
	g.Eventually(func() error { return c.Get(context.TODO(), depKeyBuacom, deployBuacom) }, timeout).
		Should(gomega.Succeed())

	// Manually delete Deployments since GC isn't enabled in the test control plane
	g.Expect(c.Delete(context.TODO(), deployCtnl)).To(gomega.Succeed())
	g.Expect(c.Delete(context.TODO(), deployBuacom)).To(gomega.Succeed())
}
