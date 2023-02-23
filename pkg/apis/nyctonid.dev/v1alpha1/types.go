package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ClusterID",type=string,JSONPath=`.status.respaldoID`
// +kubebuilder:printcolumn:name="Progress",type=string,JSONPath=`.status.progress`
type Respaldo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RespaldoSpec   `json:"spec,omitempty"`
	Status            RespaldoStatus `json:"status,omitempty"`
}

type RespaldoSpec struct {
	Namespace    string `json:"namespace,omitempty" binding:"required"`
	PVCName      string `json:"pvcname,omitempty"`
	ResourceName string `json:"resourcename,omitempty"`
	SnapshotName string `json:"snapshotname,omitempty"`
	Backup       bool   `json:"backup,omitempty"`
	Restore      bool   `json:"restore,omitempty"`
}

type RespaldoStatus struct {
	RespaldoID string `json:"respaldoID,omitempty"`
	Progress   string `json:"progress,omitempty"`
	KubeConfig string `json:"kubeConfig,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RespaldoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Respaldo `json:"items,omitempty"`
}
