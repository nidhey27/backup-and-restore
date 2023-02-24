package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ClusterID",type=string,JSONPath=`.status.respaldoID`
// +kubebuilder:printcolumn:name="Progress",type=string,JSONPath=`.status.progress`
type BackupNRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupNRestoreSpec   `json:"spec,omitempty"`
	Status            BackupNRestoreStatus `json:"status,omitempty"`
}

type BackupNRestoreSpec struct {
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
	// +kubebuilder:validation:Required
	PVCName string `json:"pvcname"`
	// +kubebuilder:validation:Required
	Resource string `json:"resource"`
	// +kubebuilder:validation:Required
	ResourceName string `json:"resourcename"`
	// +kubebuilder:validation:Required
	SnapshotName string `json:"snapshotname"`
	// +kubebuilder:validation:Required
	Backup bool `json:"backup"`
	// +kubebuilder:validation:Required
	Restore bool `json:"restore"`
}

type BackupNRestoreStatus struct {
	BackupNRestoreID string `json:"BackupNRestoreID,omitempty"`
	Progress         string `json:"progress,omitempty"`
	KubeConfig       string `json:"kubeConfig,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BackupNRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BackupNRestore `json:"items,omitempty"`
}
