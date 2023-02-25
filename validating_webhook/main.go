package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	backupnrestorev1alpha1 "github.com/nidhey27/backup-and-restore/pkg/apis/nyctonid.dev/v1alpha1"
	"github.com/spf13/pflag"
	admv1beta1 "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/component-base/cli/globalflag"
)

const (
	valCon = "val-controller"
)

type Options struct {
	SecureServingOptions options.SecureServingOptions
}

// Method on Option struct to add flags
func (o *Options) AddFlagSet(fs *pflag.FlagSet) {
	o.SecureServingOptions.AddFlags(fs)
}

// Method to return config that will eventually run server

type Config struct {
	SecureServingInfo *server.SecureServingInfo
}

func (o *Options) Config() *Config {
	if err := o.SecureServingOptions.MaybeDefaultWithSelfSignedCerts("0.0.0.0", nil, nil); err != nil {
		panic(err)
	}
	c := &Config{}

	o.SecureServingOptions.ApplyTo(&c.SecureServingInfo)
	return c
}

// Func to return default options
func NewDefaultOptions() *Options {
	o := &Options{
		SecureServingOptions: *options.NewSecureServingOptions(),
	}

	o.SecureServingOptions.BindPort = 8443
	// If we will not provide crt and key file then this will generate those files for us
	o.SecureServingOptions.ServerCert.PairName = valCon

	return o
}

func main() {
	options := NewDefaultOptions()
	fs := pflag.NewFlagSet(valCon, pflag.ExitOnError)

	// Add -help flag for our app
	globalflag.AddGlobalFlags(fs, valCon)

	// Adding
	options.AddFlagSet(fs)

	// Parse flag set
	err := fs.Parse(os.Args)

	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	}

	c := options.Config()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(ServeCRValidation))

	stopCh := server.SetupSignalHandler()

	_, ch, err := c.SecureServingInfo.Serve(mux, 30*time.Second, stopCh)

	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	} else {
		<-ch
	}

}

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func ServeCRValidation(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ServeCRValidation was called")

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Printf("ERROR[Getting Body] %s\n", err.Error())
	}

	// Read body and get instance of admissionReview object
	decoder := codecs.UniversalDeserializer()

	// get GVk for admissionReview Object
	gvk := admv1beta1.SchemeGroupVersion.WithKind("AdmisisonReview")

	// Var of type admission review
	var admissionReview admv1beta1.AdmissionReview
	_, _, err = decoder.Decode(body, &gvk, &admissionReview)

	if err != nil {
		log.Printf("ERROR[Converting Req Body to Admission Type] %s\n", err.Error())
	}

	// convert cr spec from admission review object
	gvk_cr := admv1beta1.SchemeGroupVersion.WithKind("BackupNRestore")
	var br backupnrestorev1alpha1.BackupNRestore
	_, _, err = decoder.Decode(admissionReview.Request.Object.Raw, &gvk_cr, &br)

	if err != nil {
		log.Printf("ERROR[Converting Admission Req Raw Obj to BackupNRestore Type] %s\n", err.Error())
	}

	fmt.Printf("BR taht we have is %+v\n", br)
	allow, err := validateBackupNRestore(br)
	var resp admv1beta1.AdmissionResponse

	if !allow || err != nil {
		resp = admv1beta1.AdmissionResponse{
			UID:     admissionReview.Request.UID,
			Allowed: allow,
			Result: &v1.Status{
				Message: err.Error(),
			},
		}
	} else {
		resp = admv1beta1.AdmissionResponse{
			UID:     admissionReview.Request.UID,
			Allowed: allow,
			Result: &v1.Status{
				Message: "",
			},
		}
	}
	admissionReview.Response = &resp
	// fmt.Println(resp)
	log.Printf("respoknse that we are trying to return is %+v\n", resp)
	res, err := json.Marshal(admissionReview)

	if err != nil {
		log.Printf("error %s, while converting response to byte slice", err.Error())
	}

	_, err = w.Write(res)

	if err != nil {
		log.Printf("error %s, writing respnse to responsewriter", err.Error())
	}

}

func validateBackupNRestore(br backupnrestorev1alpha1.BackupNRestore) (bool, error) {

	if br.Spec.Backup {
		err := validateBackup(br.Spec.Namespace, br.Spec.PVCName, br.Spec.SnapshotName)
		if err != nil {
			return false, err
		}
	} else if br.Spec.Restore {
		err := validateRestore(br.Spec.Namespace, br.Spec.Resource, br.Spec.ResourceName, br.Spec.PVCName, br.Spec.SnapshotName)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}
