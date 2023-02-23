package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	volumesnapshot "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/nidhey27/backup-and-restore/pkg/apis/nyctonid.dev/v1alpha1"
	clientset "github.com/nidhey27/backup-and-restore/pkg/client/clientset/versioned"
	"github.com/nidhey27/backup-and-restore/pkg/client/clientset/versioned/scheme"
	kinf "github.com/nidhey27/backup-and-restore/pkg/client/informers/externalversions/nyctonid.dev/v1alpha1"
	clister "github.com/nidhey27/backup-and-restore/pkg/client/listers/nyctonid.dev/v1alpha1"

	// "github.com/viveksinghggits/kluster/pkg/apis/viveksingh.dev/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	nytonidv1aphla1 "github.com/nidhey27/backup-and-restore/pkg/apis/nyctonid.dev/v1alpha1"
)

type Controller struct {
	client               kubernetes.Interface
	snapshotterclientset snapshotclient.Interface
	klient               clientset.Interface
	klusterSynced        cache.InformerSynced
	kLister              clister.BackupNRestoreLister
	wq                   workqueue.RateLimitingInterface
	recorder             record.EventRecorder
}

func NewController(client kubernetes.Interface, snapshotterclientset snapshotclient.Interface, klient clientset.Interface, klusterInformer kinf.BackupNRestoreInformer) *Controller {
	runtime.Must(scheme.AddToScheme(scheme.Scheme))

	eveBroadCaster := record.NewBroadcaster()
	eveBroadCaster.StartStructuredLogging(0)
	eveBroadCaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: client.CoreV1().Events(""),
	})
	recorder := eveBroadCaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "BackupNRestore"})

	c := &Controller{
		client:               client,
		snapshotterclientset: snapshotterclientset,
		klient:               klient,
		klusterSynced:        klusterInformer.Informer().HasSynced,
		kLister:              klusterInformer.Lister(),
		wq:                   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "backupnrestore"),
		recorder:             recorder,
	}

	klusterInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
			UpdateFunc: c.handleUpdate,
		},
	)

	return c
}

func (c *Controller) worker() {
	for c.processNextItem() {

	}
}

func (c *Controller) processNextItem() bool {
	item, shutDown := c.wq.Get()
	if shutDown {
		// logs as well
		return false
	}
	defer c.wq.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		log.Printf("error %s calling Namespace key func on cache for item", err.Error())
		return false
	}
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("splitting key into namespace and name, error %s\n", err.Error())
		return false
	}
	// fmt.Println(ns, name)
	backupNRestore, err := c.kLister.BackupNRestores(ns).Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false
		}

		log.Printf("error %s, Getting the BackupNRestores resource from lister", err.Error())
		return false
	}
	log.Printf("BackupNRestores spec that we have is %+v\n", backupNRestore.Spec)

	vs, err := c.createSnapshot(backupNRestore)
	if err != nil {
		// do something
		log.Printf("errro %s, creating the Snapshot", err.Error())
	}
	c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", "createSnapshot func was called to create snapshot")

	err = c.updateStatus("creating", backupNRestore)
	if err != nil {
		log.Printf("error %s, updating status of the backupNRestore %s\n", err.Error(), backupNRestore.Name)
	}
	// query DO API to make sure clsuter' state is running
	err = c.waitForSnapshot(*vs, vs.Name)
	if err != nil {
		log.Printf("error %s, waiting for snapshot", err.Error())
	}
	err = c.updateStatus("Created", backupNRestore)
	if err != nil {
		log.Printf("error %s updaring cluster status after waiting for cluster", err.Error())
	}
	c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCompleted", "VolumeSnapshot was completed")
	return true
}

func (c *Controller) waitForSnapshot(vs volumesnapshot.VolumeSnapshot, snapshotName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	_, err := c.snapshotterclientset.SnapshotV1().VolumeSnapshots(vs.Namespace).Get(ctx, vs.Name, metav1.GetOptions{})
	return err
}

func (c *Controller) Run(ch chan struct{}) error {
	log.Println("Satrting controller..")
	if ok := cache.WaitForCacheSync(ch, c.klusterSynced); !ok {
		log.Println("cache was not sycned")
	}

	go wait.Until(c.worker, time.Second, ch)

	<-ch
	return nil
}

// its gong to get called, whenever the resource is updated
func (c *Controller) handleUpdate(ondObj, newObj interface{}) {
	log.Println("obj updated")
}
func (c *Controller) handleDel(obj interface{}) {
	log.Println("obj deleted")
	c.wq.Add(obj)
}
func (c *Controller) handleAdd(obj interface{}) {
	log.Println("obj created")
	c.wq.Add(obj)
}

func (c *Controller) updateStatus(progress string, kluster *v1alpha1.BackupNRestore) error {
	// get the latest version of kluster
	k, err := c.klient.NyctonidV1alpha1().BackupNRestores(kluster.Namespace).Get(context.Background(), kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	k.Status.Progress = progress
	_, err = c.klient.NyctonidV1alpha1().BackupNRestores(kluster.Namespace).UpdateStatus(context.Background(), k, metav1.UpdateOptions{})
	return err
}

func (c *Controller) createSnapshot(obj *nytonidv1aphla1.BackupNRestore) (*volumesnapshot.VolumeSnapshot, error) {
	ctx := context.Background()
	current_time := time.Now()
	timeStamp := fmt.Sprintf("%d-%02d-%02dt%02d-%02d-%02dt", current_time.Year(), current_time.Month(), current_time.Day(),
		current_time.Hour(), current_time.Minute(), current_time.Second())

	// fmt.Println(obj.Spec.PVCName + timeStamp)
	volumeSnapshotClassName := "csi-hostpath-snapclass"
	pvcName := obj.Spec.PVCName
	volumeShnapshot := volumesnapshot.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.Spec.PVCName + "-" + timeStamp,
			Namespace: obj.Spec.Namespace,
		},
		Spec: volumesnapshot.VolumeSnapshotSpec{
			Source: volumesnapshot.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvcName,
			},
			VolumeSnapshotClassName: &volumeSnapshotClassName,
		},
	}
	// fmt.Println(*volumeShnapshot)
	// fmt.Println(obj.Spec.Namespace)
	vs, err := c.snapshotterclientset.SnapshotV1().VolumeSnapshots(obj.Spec.Namespace).Create(ctx, &volumeShnapshot, metav1.CreateOptions{})
	if err != nil {
		log.Printf("ERROR:  %s\n", err.Error())
		return nil, err
	}
	fmt.Printf("Volume Snaphot Created: %s\n", vs.Name)
	return vs, nil
}
