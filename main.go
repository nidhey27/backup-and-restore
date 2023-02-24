package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	clientset "github.com/nidhey27/backup-and-restore/pkg/client/clientset/versioned"
	infFac "github.com/nidhey27/backup-and-restore/pkg/client/informers/externalversions"
	"github.com/nidhey27/backup-and-restore/pkg/controller"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {

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
			log.Printf("ERROR[Building config]: %s\n", err.Error())
		}
	}
	crclientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Printf("ERROR[CR ClientSet]: %s\n", err.Error())
	}
	coreclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("ERROR[ClientSet]: %s\n", err.Error())
	}
	snapshotclient, err := snapshotclient.NewForConfig(config)
	if err != nil {
		log.Printf("ERROR[SnapshotClientSet]: %s\n", err.Error())
	}

	dynamicclientset, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Printf("ERROR[DynamicClientSet]: %s\n", err.Error())
	}
	infoFactory := infFac.NewSharedInformerFactoryWithOptions(crclientset, 1*time.Minute)
	ch := make(chan struct{})

	c := controller.NewController(coreclientset, snapshotclient, crclientset, dynamicclientset, infoFactory.Nyctonid().V1alpha1().BackupNRestores())
	infoFactory.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("ERROR[Running Controller]%s\n", err.Error())
	}
}
