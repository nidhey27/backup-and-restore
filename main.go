package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	snapshots "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	cleinetset "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	ctx := context.Background()
	kubeConfig := flag.String("kubeconfig", "/home/nidhey/.kube/config", "location to your kubeconfig file")

	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)

	if err != nil {
		// handel error
		// panic(err)
		fmt.Printf("Error %s buidling configfile from flag\n", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("Error %s getting configfile in Clutser Config\n", err.Error())
		}
	}

	namespace := "default"
	pvcName := "csi-pvc"

	fmt.Printf("Enter Namespace:\n")
	fmt.Scan(&namespace)

	fmt.Printf("Enter PVC Name from %s to take Snapshot of:\n", namespace)
	fmt.Scan(&pvcName)

	coreClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("getting CORE CLIENT SET  %s\n", err.Error())
	}

	_, err = coreClientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})

	if err != nil {
		log.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}

	clientset, err := cleinetset.NewForConfig(config)
	if err != nil {
		log.Printf("getting CR CLIENT SET  %s\n", err.Error())
	}

	current_time := time.Now()
	timeStamp := fmt.Sprintf("%d-%02d-%02dt%02d-%02d-%02dt", current_time.Year(), current_time.Month(), current_time.Day(),
		current_time.Hour(), current_time.Minute(), current_time.Second())

	fmt.Println(pvcName + timeStamp)
	volumeSnapshotClassName := "csi-hostpath-snapclass"
	volumeShnapshot := &snapshots.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName + "-" + timeStamp,
			Namespace: namespace,
		},
		Spec: snapshots.VolumeSnapshotSpec{
			Source: snapshots.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvcName,
			},
			VolumeSnapshotClassName: &volumeSnapshotClassName,
		},
	}

	vs, err := clientset.SnapshotV1().VolumeSnapshots(namespace).Create(ctx, volumeShnapshot, metav1.CreateOptions{})

	if err != nil {
		log.Printf("ERROR:  %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("VS Created: %s\n", vs.Name)
}
