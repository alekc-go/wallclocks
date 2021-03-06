/*


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

// TimezoneSpec defines the desired state of Timezone
type TimezoneSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Timezones is a list of valid timezones
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Required
	Timezones []string `json:"timezones"`
}

// TimezoneStatus defines the observed state of Timezone
type TimezoneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true

// Timezone is the Schema for the timezones API
type Timezone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TimezoneSpec   `json:"spec,omitempty"`
	Status TimezoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TimezoneList contains a list of Timezone
type TimezoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Timezone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Timezone{}, &TimezoneList{})
}
