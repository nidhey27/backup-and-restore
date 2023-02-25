package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath" 

	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	cmdutils "k8s.io/kubectl/pkg/cmd/util"
)

type Controller struct {
	clientset *dynamic.DynamicClient
}

func getClientSetConfig() (*rest.Config, error) {
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("Building config from flags failed, %s, trying to build inclusterconfig", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			// log.Printf("ERROR[Building config]: %s\n", err.Error())
			return nil, err
		}
	}

	return config, nil
}

func getClientSet(config *rest.Config) (*dynamic.DynamicClient, error) {
	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		// log.Printf("ERROR[ClientSet]: %s\n", err.Error())
		return nil, err
	}

	return clientset, nil
}

func getCRClientSet(config *rest.Config) (*snapshotclient.Clientset, error) {
	clientset, err := snapshotclient.NewForConfig(config)
	if err != nil {
		// log.Printf("ERROR[SnapshotClientSet]: %s\n", err.Error())
		return nil, err
	}

	return clientset, nil
}

func newController(clientset *dynamic.DynamicClient, snapshotclient *snapshotclient.Clientset) *Controller {
	c := &Controller{
		clientset: clientset,
	}
	return c
}

func getController() *Controller {
	// Client Set - Structured & CR
	config, err := getClientSetConfig()

	if err != nil {
		log.Printf("ERROR[]: %s", err.Error())
	}
	clientset, err := getClientSet(config)

	if err != nil {
		log.Printf("ERROR[]: %s", err.Error())
	}

	snapshotclient, err := getCRClientSet(config)
	if err != nil {
		log.Printf("ERROR[]: %s", err.Error())
	}

	con := newController(clientset, snapshotclient)

	return con
}

func validateBackup(namespace string, PVCName string, snapshotName string) error {
	ctx := context.Background()
	con := getController()

	gvr_pvc, err := getGVR("persistentvolumeclaims")
	if err != nil {
		return err
	}

	_, err = con.clientset.Resource(schema.GroupVersionResource{
		Group:    gvr_pvc.Group,
		Version:  gvr_pvc.Version,
		Resource: gvr_pvc.Resource,
	}).Namespace(namespace).Get(ctx, PVCName, metav1.GetOptions{})

	if err != nil {
		return err
	}

	gvr_vs, err := getGVR("volumesnapshots")
	if err != nil {
		return err
	}

	_, err = con.clientset.Resource(schema.GroupVersionResource{
		Group:    gvr_vs.Group,
		Version:  gvr_vs.Version,
		Resource: gvr_vs.Resource,
	}).Namespace(namespace).Get(ctx, snapshotName, metav1.GetOptions{})

	if err != nil {
		return errors.Errorf("%s VolumeSnapshot alreay exists.")
	}

	return nil
}

func validateRestore(namespace string, resource string, resourceName string, PVCName string, snapshotName string) error {
	ctx := context.Background()
	con := getController()

	// ******** 1 **********
	// Check if resource provided is a valid resource or not
	gvr_res, err := getGVR(resource)
	if err != nil {
		return err
	}

	// check if the resource exits or not
	_, err = con.clientset.Resource(schema.GroupVersionResource{
		Group:    gvr_res.Group,
		Version:  gvr_res.Version,
		Resource: gvr_res.Resource,
	}).Namespace(namespace).Get(ctx, resourceName, metav1.GetOptions{})

	if err != nil {
		return err
	}
	// ******** 1 **********

	// ******** 2 **********
	// Check if PVC provided already exists 
	gvr_pvc, err := getGVR("persistentvolumeclaims")
	if err != nil {
		return err
	}

	_, err = con.clientset.Resource(schema.GroupVersionResource{
		Group:    gvr_pvc.Group,
		Version:  gvr_pvc.Version,
		Resource: gvr_pvc.Resource,
	}).Namespace(namespace).Get(ctx, PVCName, metav1.GetOptions{})

	if err != nil {
		return errors.Errorf("%s PVC alreay exists.")
	}

	// ******** 2 **********

	// ******** 3 **********
	// Check if VS provided exists or not 
	gvr_vs, err := getGVR("volumesnapshots")
	if err != nil {
		return err
	}

	_, err = con.clientset.Resource(schema.GroupVersionResource{
		Group:    gvr_vs.Group,
		Version:  gvr_vs.Version,
		Resource: gvr_vs.Resource,
	}).Namespace(namespace).Get(ctx, snapshotName, metav1.GetOptions{})

	if err != nil {
		return err
	}
	// ******** 3 **********

	return nil
}

// check if given resource is a k8s resource
func getGVR(resource string) (schema.GroupVersionResource, error) {

	configFlag := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	macthedVersion := cmdutils.NewMatchVersionFlags(configFlag)
	m, err := cmdutils.NewFactory(macthedVersion).ToRESTMapper()
	var gvr schema.GroupVersionResource
	if err != nil {
		fmt.Printf("gettign rest mapper from newfactory %s", err.Error())
		return gvr, err
	}
	gvr, err = m.ResourceFor(schema.GroupVersionResource{
		Resource: resource,
	})

	if err != nil {
		return gvr, err
	}

	return gvr, nil
}
