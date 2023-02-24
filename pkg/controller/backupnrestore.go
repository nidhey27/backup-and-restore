package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	volumesnapshot "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"github.com/nidhey27/backup-and-restore/pkg/apis/nyctonid.dev/v1alpha1"
	nytonidv1aphla1 "github.com/nidhey27/backup-and-restore/pkg/apis/nyctonid.dev/v1alpha1"
	clientset "github.com/nidhey27/backup-and-restore/pkg/client/clientset/versioned"
	"github.com/nidhey27/backup-and-restore/pkg/client/clientset/versioned/scheme"
	kinf "github.com/nidhey27/backup-and-restore/pkg/client/informers/externalversions/nyctonid.dev/v1alpha1"
	clister "github.com/nidhey27/backup-and-restore/pkg/client/listers/nyctonid.dev/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	genericclioptions "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	cmdutils "k8s.io/kubectl/pkg/cmd/util"
)

type Controller struct {
	client               kubernetes.Interface
	snapshotterclientset snapshotclient.Interface
	klient               clientset.Interface
	dynamicclientset     dynamic.Interface
	klusterSynced        cache.InformerSynced
	kLister              clister.BackupNRestoreLister
	wq                   workqueue.RateLimitingInterface
	recorder             record.EventRecorder
}

func NewController(client kubernetes.Interface, snapshotterclientset snapshotclient.Interface, klient clientset.Interface, dynamicclientset dynamic.Interface, klusterInformer kinf.BackupNRestoreInformer) *Controller {
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
		dynamicclientset:     dynamicclientset,
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
		return false
	}
	defer c.wq.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		log.Printf("ERROR[Calling Namespace key func on cache for item]: %s \n", err.Error())
		return false
	}
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("ERROR[Splitting key into namespace and name]: %s\n", err.Error())
		return false
	}
	backupNRestore, err := c.kLister.BackupNRestores(ns).Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false
		}

		log.Printf("ERROR[Getting the BackupNRestores resource from lister]: %s\n", err.Error())
		return false
	}
	if backupNRestore.Spec.Backup {
		vs, err := c.createSnapshot(backupNRestore)
		if err != nil {
			log.Printf("ERROR[Creating the Snapshot]: %s", err.Error())
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			c.updateStatus("SnapshotError", backupNRestore)
			return false
		}
		c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", "createSnapshot() was called to create snapshot")
		err = c.updateStatus("Creating", backupNRestore)
		if err != nil {
			log.Printf("ERROR[updating status of the backupNRestore]: %s, [CR NAME]: %s\n", err.Error(), backupNRestore.Name)
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			return false
		}
		err = c.waitForSnapshot(*vs)
		if err != nil {
			log.Printf("ERROR[Waiting for snapshot]: %s\n", err.Error())
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			c.updateStatus("SnapshotError", backupNRestore)
			return false
		}
		err = c.updateStatus("Created", backupNRestore)
		if err != nil {
			log.Printf("ERROR[]: %s ", err.Error())
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			return false
		}
		c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCompleted", "VolumeSnapshot was completed")

	} else if backupNRestore.Spec.Restore {
		c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", "restoreSnapsShot() was called to restore snapshot")
		pvc, err := c.restoreSnapsShot(backupNRestore)
		if err != nil {
			log.Printf("ERROR[RESTORING SNAPSHOT]: %s, [CR NAME] %s\n", err.Error(), backupNRestore.Name)
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			c.updateStatus("PVCError", backupNRestore)
			return false
		}
		err = c.updateStatus("Restoring", backupNRestore)
		if err != nil {
			log.Printf("ERROR[updating status of the backupNRestore]: %s, [CR NAME] %s\n", err.Error(), backupNRestore.Name)
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			c.updateStatus("PVCError", backupNRestore)
			return false
		}
		err = c.waitForPVC(pvc)
		if err != nil {
			log.Printf("ERROR[Waiting for snapshot]: %s\n", err.Error())
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			c.updateStatus("PVCError", backupNRestore)
			return false
		}

		err = c.updateStatus("Restored", backupNRestore)
		if err != nil {
			log.Printf("ERROR[]: %s ", err.Error())
			c.recorder.Event(backupNRestore, corev1.EventTypeNormal, "BackupNRestoreCreation", err.Error())
			return false
		}
	}

	return true
}

func (c *Controller) waitForPVC(pvc *corev1.PersistentVolumeClaim) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	res, err := c.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	phase := res.Status.Phase

	switch phase {
	case corev1.ClaimPending:
		err = errors.Errorf("PVC is pending")
	case corev1.ClaimBound:
		err = nil
	case corev1.ClaimLost:
		err = errors.Errorf("PVC binding is lost")
	}
	return err
}

func (c *Controller) waitForSnapshot(vs volumesnapshot.VolumeSnapshot) error {
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
	res := newObj.(*nytonidv1aphla1.BackupNRestore)
	log.Printf("%s updated", res.Name)
}
func (c *Controller) handleDel(obj interface{}) {
	res := obj.(*nytonidv1aphla1.BackupNRestore)
	log.Printf("%s deleted", res.Name)
	c.wq.Add(obj)
}
func (c *Controller) handleAdd(obj interface{}) {
	res := obj.(*nytonidv1aphla1.BackupNRestore)
	log.Printf("%s created", res.Name)
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

	volumeSnapshotClassName := "csi-hostpath-snapclass"
	pvcName := obj.Spec.PVCName
	volumeShnapshot := volumesnapshot.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.Spec.SnapshotName,
			Namespace: obj.Spec.Namespace,
		},
		Spec: volumesnapshot.VolumeSnapshotSpec{
			Source: volumesnapshot.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvcName,
			},
			VolumeSnapshotClassName: &volumeSnapshotClassName,
		},
	}

	vs, err := c.snapshotterclientset.SnapshotV1().VolumeSnapshots(obj.Spec.Namespace).Create(ctx, &volumeShnapshot, metav1.CreateOptions{})
	if err != nil {
		log.Printf("ERROR:  %s\n", err.Error())
		return nil, err
	}
	fmt.Printf("Volume Snaphot Created: %s\n", vs.Name)
	return vs, nil
}

func (c *Controller) restoreSnapsShot(obj *nytonidv1aphla1.BackupNRestore) (*corev1.PersistentVolumeClaim, error) {
	ctx := context.Background()
	configFlag := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	macthedVersion := cmdutils.NewMatchVersionFlags(configFlag)
	m, err := cmdutils.NewFactory(macthedVersion).ToRESTMapper()
	if err != nil {
		fmt.Printf("gettign rest mapper from newfactory %s", err.Error())
		return nil, err
	}
	gvr, err := m.ResourceFor(schema.GroupVersionResource{
		Resource: obj.Spec.Resource,
	})
	if err != nil {
		fmt.Printf("ERROR GVR: %s\n", err.Error())
		return nil, err
	}
	log.Println(gvr)
	resource, err := c.dynamicclientset.Resource(schema.GroupVersionResource{
		Group:    gvr.Group,
		Version:  gvr.Version,
		Resource: gvr.Resource,
	}).Namespace(obj.Spec.Namespace).Get(ctx, obj.Spec.ResourceName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return nil, err
	}

	pvc, err := c.createPVC(obj)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return nil, err
	}
	err = c.updateDeploymentVolume(gvr, resource, obj.Spec.Namespace, pvc.Name)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return nil, err
	}

	return pvc, nil
}

func (c *Controller) createPVC(obj *nytonidv1aphla1.BackupNRestore) (*corev1.PersistentVolumeClaim, error) {
	ctx := context.Background()
	storageClassName := "csi-hostpath-sc"
	pvcName := obj.Spec.PVCName
	snapshotName := obj.Spec.SnapshotName
	apiGroup := "snapshot.storage.k8s.io"
	pvObj := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: obj.Spec.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			DataSource: &corev1.TypedLocalObjectReference{
				Name:     snapshotName,
				Kind:     "VolumeSnapshot",
				APIGroup: &apiGroup,
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}

	pv, err := c.client.CoreV1().PersistentVolumeClaims(obj.Spec.Namespace).Create(ctx, pvObj, metav1.CreateOptions{})
	if err != nil {
		log.Printf("ERROR:  %s\n", err.Error())
		return nil, err
	}
	fmt.Printf("PVC Created: %s\n", pv.Name)
	return pv, nil
}

func (c *Controller) updateDeploymentVolume(gvr schema.GroupVersionResource, resource *unstructured.Unstructured, namespace string, pvcName string) error {
	ctx := context.Background()
	volInterface, _, err := unstructured.NestedFieldNoCopy(resource.Object, "spec", "template", "spec", "volumes")
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return err
	}

	volumes, ok := volInterface.([]interface{})

	if !ok {
		return errors.Errorf("expected of type %T but got %T", []interface{}{}, volInterface)
	}

	name, _, err := unstructured.NestedString(volumes[0].(map[string]interface{}), "name")
	if err != nil {
		return errors.Wrapf(err, "failed to get name present in volumes")
	}

	newVols := []interface{}{
		map[string]interface{}{
			"name": name,
			"persistentVolumeClaim": map[string]interface{}{
				"claimName": pvcName,
			},
		},
	}

	volInterface = newVols

	resource.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"] = newVols
	_, err = c.dynamicclientset.Resource(schema.GroupVersionResource{
		Group:    gvr.Group,
		Version:  gvr.Version,
		Resource: gvr.Resource,
	}).Namespace(namespace).Update(ctx, resource, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("ERROR updating resource: %s\n", err.Error())
		return err
	}
	return nil
}
