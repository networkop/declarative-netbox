/*
Copyright 2021.

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

package v1

import (
	_ "github.com/netbox-community/go-netbox/netbox/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceState string

const DeviceKind = "Device"
const DeviceFinalizer = "finalizers.netbox.networkop.co.uk"
const DeviceReadyState DeviceState = "Ready"

// DeviceSpec defines the desired state of Netbox Device
type DeviceSpec struct {
	// Name of an existing Netbox Site
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +required
	Site string `json:"site,omitempty"`

	// Name of an existing Netbox Device Type
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +required
	DeviceType string `json:"device_type,omitempty"`

	// Name of an existing Netbox Device Role
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +required
	Role string `json:"role,omitempty"`
}

// DeviceStatus defines the observed state of Device
type DeviceStatus struct {
	ID    *int64      `json:"id,omitempty"`
	State DeviceState `json:"state,omitempty"`
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.site`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.device_type`
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.role`
// Device is the Schema for the devices API
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceSpec   `json:"spec,omitempty"`
	Status DeviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceList contains a list of Device
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Device{}, &DeviceList{})
}
