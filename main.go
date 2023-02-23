package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	clientset "github.com/nidhey27/backup-and-restore/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Printf("getting klientset set %s\n", err.Error())
	}

	respaldo, err := clientset.NyctonidV1alpha1().BackupNRestores("").List(context.Background(), metav1.ListOptions{})

	if err != nil {
		log.Printf("respaldo lisiting  %s\n", err)
		// log.Panicln(err)
	}

	log.Println(len(respaldo.Items))
}
