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
	"context"
	"log"
	"reflect"
	"regexp"

	dyndeployv1beta1 "github.com/Travix-International/dynamic-deploy/pkg/apis/dyndeploy/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new DynDeploy Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &DynDeployController{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("dyndeploy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to DynDeploy
	err = c.Watch(&source.Kind{Type: &dyndeployv1beta1.DynDeploy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch a Deployment created by DynDeploy
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dyndeployv1beta1.DynDeploy{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &DynDeployController{}

// DynDeployController reconciles a DynDeploy object
type DynDeployController struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a DynDeploy object and makes changes based on the state read
// and what is in the DynDeploy.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dyndeploy.travix.com,resources=dyndeploys,verbs=get;list;watch;create;update;patch;delete
func (r *DynDeployController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the DynDeploy
	dd := &dyndeployv1beta1.DynDeploy{}
	err := r.Get(context.TODO(), request.NamespacedName, dd)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	for _, k := range dd.Spec.Keys {
		err = r.createOrUpdateDeployment(request, dd, k)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *DynDeployController) createOrUpdateDeployment(request reconcile.Request, dd *dyndeployv1beta1.DynDeploy, key string) error {
	var processedKey string
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Printf("Error compiling regex, using key as is.\n")
		processedKey = key
	} else {
		processedKey = reg.ReplaceAllString(key, "")
	}

	// Calculate the expected Deployment Spec
	spec := getDeploymentSpec(request, dd, processedKey)

	// Define the desired Deployment object
	deploy := &appsv1.Deployment{Spec: spec}
	deploy.Name = request.Name + "-" + processedKey
	deploy.Namespace = request.Namespace

	// Check if the Deployment already exists
	found := &appsv1.Deployment{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating Deployment %s/%s\n", deploy.Namespace, deploy.Name)
		if err = controllerutil.SetControllerReference(dd, deploy, r.scheme); err != nil {
			return err
		}
		err = r.Create(context.TODO(), deploy)
		if err != nil {
			log.Printf("Error creating Deployment %s/%s: %s\n", deploy.Namespace, deploy.Name, err)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	// Update the found object and write the result back if there are any changes
	if !reflect.DeepEqual(deploy.Spec, found.Spec) {
		found.Spec = deploy.Spec
		log.Printf("Updating Deployment %s/%s\n", deploy.Namespace, deploy.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

func getDeploymentSpec(request reconcile.Request, dd *dyndeployv1beta1.DynDeploy, key string) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"dyndeploy": request.Name},
		},
		Replicas: &dd.Spec.Replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"dyndeploy": request.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: request.Name + "-" + key,
						Image: dd.Spec.Image},
				},
			},
		},
	}
}
