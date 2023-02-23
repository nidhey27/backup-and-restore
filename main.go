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
			log.Printf("error %s building inclusterconfig", err.Error())
		}
	}
	crclientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Printf("getting klientset set %s\n", err.Error())
	}

	// respaldo, err := clientset.NyctonidV1alpha1().BackupNRestores("").List(context.Background(), metav1.ListOptions{})

	// if err != nil {
	// 	log.Printf("respaldo lisiting  %s\n", err)
	// 	// log.Panicln(err)
	// }

	// log.Println(len(respaldo.Items))
	coreclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("getting std client %s\n", err.Error())
	}
	snapshotclient, err := snapshotclient.NewForConfig(config)
	if err != nil {
		log.Printf("getting snapshot client %s\n", err.Error())
	}
	infoFactory := infFac.NewSharedInformerFactoryWithOptions(crclientset, 1*time.Minute)
	ch := make(chan struct{})

	c := controller.NewController(coreclientset, snapshotclient, crclientset, infoFactory.Nyctonid().V1alpha1().BackupNRestores())
	infoFactory.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("error running controller %s\n", err.Error())
	}
}
