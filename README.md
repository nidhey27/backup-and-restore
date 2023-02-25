# backup-and-restore

A kubernetes operator which takes backup of given PVC and restore it at any given point of time


How to run

- Setup Admission Webhook
```
cd validating_webhook/manifests
kubectl apply -f deployment.yaml
kubectl apply -f crb.yaml  
kubectl apply -f cr.yaml  
kubectl apply -f deployment.yaml  
kubectl apply -f sa.yaml  
kubectl apply -f service.yaml  
kubectl apply -f val-webhook.yaml
```

- Setup BackupNRestore Operatore
```
kubectl apply -f ./manifests/nyctonid.dev_backupnrestores.yaml
```

- Take a Snapshot
  - Required fields: 
      - namespace 
      - backup - flag must be true
      - pvcname - name of PVC which needs to be backuped
      - snapshotname - name of the snapshot 
      

#### Example:
```
---
apiVersion: nyctonid.dev/v1alpha1
kind: BackupNRestore
metadata:
  name: backup-n-restore-1
spec:
  backup: true
  namespace: default
  pvcname: pvc-claim-5
  resource: "deployments"
  resourcename:  "mysql-deployment"
  restore: false
  snapshotname: "snapshot-one"
```

- Restore a Snapshot
  - Required fields: 
    - namespace 
    - restore - flag must be true
    - snapshotname - name of vs which needs to be restored
    - pvcname

#### Example:
```
---
apiVersion: nyctonid.dev/v1alpha1
kind: BackupNRestore
metadata:
  name: backup-n-restore-2
spec:
  backup: false
  namespace: default
  pvcname: pvc-claim-6
  resource: "deployments"
  resourcename: "mysql-deployment"
  restore: true
  snapshotname: "snapshot-one"
```
