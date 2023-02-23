package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   "nyctonid.dev",
	Version: "v1alpha1",
}

var (
	SchemeBuilder runtime.SchemeBuilder
	AddToScheme   = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(addKnowTypes)
}

func addKnowTypes(scheme *runtime.Scheme) error {

	scheme.AddKnownTypes(SchemeGroupVersion, &Respaldo{}, &RespaldoList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}

// ~/go/src/generate-groups.sh  all github.com/nidhey27/respaldo/pkg/client github.com/nidhey27/respaldo/pkg/apis nyctonid.dev:v1alpha1 --go-header-file ~/go/src/hack/boilerplate.go.txt -o ./
// controller-gen rbac:roleName=respaldo-role crd paths=./pkg/apis/nyctonid.dev/v1alpha1 output:crd:dir=./manifests output:stdout
