/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProfileSpec defines the desired state of Profile
type ProfileSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	NatsAddress        string             `json:"nats_address"`
	RedisAddress       string             `json:"redis_address"`
	FrontendService    FrontendService    `json:"frontend_service"`
	AccumulatorService AccumulatorService `json:"accumulator_service"`
	DispenserService   DispenserService   `json:"dispenser_service"`
	MatchProfiles      []*MatchProfile    `json:"match_profiles"`
}

type FrontendService struct {
	Image            string `json:"image"`
	ValidationTarget string `json:"validation_target,omitempty"`
	DataTarget       string `json:"data_target,omitempty"`
}

type AccumulatorService struct {
	Image string `json:"image"`
}

type DispenserService struct {
	Image            string `json:"image"`
	AssignmentTarget string `json:"assignment_target"`
}

type MatchProfile struct {
	Name             string `json:"name"`
	MatchmakerTarget string `json:"matchmaker_target"`
	Pools            []Pool `json:"pools"`
}

type Pool struct {
	Name                string               `json:"name"`
	DoubleRangeFilters  []DoubleRangeFilter  `json:"double_range_filters,omitempty"`
	StringEqualsFilters []StringEqualsFilter `json:"string_equals_filters,omitempty"`
	TagPresentFilters   []TagPresentFilter   `json:"tag_present_filters,omitempty"`
}

type DoubleRangeFilter struct {
	Arg string  `json:"arg"`
	Min float64 `json:"min,string"`
	Max float64 `json:"max,string"`
}

type StringEqualsFilter struct {
	Arg   string `json:"arg"`
	Value string `json:"value"`
}

type TagPresentFilter struct {
	Tag string `json:"tag"`
}

// ProfileStatus defines the observed state of Profile
type ProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Profile is the Schema for the profiles API
type Profile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProfileSpec   `json:"spec,omitempty"`
	Status ProfileStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProfileList contains a list of Profile
type ProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Profile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Profile{}, &ProfileList{})
}
