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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DynDeploySpec defines the desired state of DynDeploy
type DynDeploySpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// replicas is the number of replicas to maintain
	Replicas int32 `json:"replicas,omitempty"`

	// image is the container image to run.  Image must have a tag.
	// +kubebuilder:validation:Pattern=.+(:.+)?
	Image string `json:"image,omitempty"`

	// keys are the prefixes use to do several deployments based on each key
	Keys []string `json:"keys,omitempty"`
}

// DynDeployStatus defines the observed state of DynDeploy
type DynDeployStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	HealthyReplicas int32 `json:"healthyReplicas,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynDeploy is the Schema for the dyndeploys API
// +k8s:openapi-gen=true
type DynDeploy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynDeploySpec   `json:"spec,omitempty"`
	Status DynDeployStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynDeployList contains a list of DynDeploy
type DynDeployList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynDeploy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DynDeploy{}, &DynDeployList{})
}
