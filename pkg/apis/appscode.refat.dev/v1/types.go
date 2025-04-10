package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Apiserver is a Apiserver resource.
type Apiserver struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`

	Spec ApiServerSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApiserverList is a collection of Apiserver resources.
type ApiserverList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata"`

	Items []Apiserver `json:"items"`
}

// Apiserver is the spec of a Apiserver resource.
type ApiServerSpec struct {
	Image    string `json:"image"`    // The container image
	Replicas *int32 `json:"replicas"` // Number of replicas (pods) to run
	Port     int32  `json:"port"`     // The port for the Apiserver
}
