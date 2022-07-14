#!/usr/bin/env bash

# Copyright 2020 The OpenYurt Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -x


YURT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
YURT_OUTPUT_DIR=${YURT_ROOT}/_output

if [ -z $1 ]; then
    echo "Must input image name"
	exit 1
fi

if [ -z $2 ]; then
    echo "Must input docker user name"
	exit 1
fi

if [ -z $3 ]; then
    echo "Must input docker passwd"
	exit 1
fi


IMAGE="$1"
DOCKER_USERNAME="$2"
DOCKER_PASSWD="$3"

function create_manifest() {

    mkdir -p ${YURT_OUTPUT_DIR}

    local output_file=${YURT_OUTPUT_DIR}/manifest.yaml
	
    local imageSecretName="benchmark"
    local server_addr=$(kubectl get configmaps -n kube-public cluster-info -o yaml  |grep "server:" |grep "https://" |awk -F ' ' '{printf $2}')

    local docker_server=$(echo ${IMAGE} |awk -F '/' '{printf $1}')
	docker login --username=${DOCKER_USERNAME}  --password=${DOCKER_PASSWD} ${docker_server}
    if [ "$?" != "0" ]; then
        echo "docker login failure"
        exit 1
    fi
    
    kubectl get secret ${imageSecretName}
    if [ "$?" != "0" ]; then
        echo "create secret ${imageSecretName}"
	    kubectl create secret docker-registry ${imageSecretName} --docker-server=${docker_server} --docker-username=${DOCKER_USERNAME} --docker-password="${DOCKER_PASSWD}" --docker-email=benchmark@alibaba-inc.com
    fi 

cat << EOF > ${output_file}

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: benchmark
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: benchmark
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: benchmark 
subjects:
- kind: ServiceAccount
  name: benchmark
  namespace: default

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: benchmark
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  - secrets
  - pods
  - nodes
  - configmaps
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
  - deletecollection
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    cloud-native-challenge: benchmark
  name: benchmark 
spec:
  replicas: 1 
  selector:
    matchLabels:
      cloud-native-challenge: benchmark
  template:
    metadata:
      labels:
        cloud-native-challenge: benchmark
    spec:
      containers:
      - args:
        - --v=2
        - --server-addr=${server_addr}
        - --log_file_max_size=1800
        - --logtostderr=false
        - --stderrthreshold=1
        - --alsologtostderr=true
        - --enable-sample-handler=true
        - --log_file=/var/log/edge-proxy/edge-proxy.log
        command:
        - /usr/local/bin/edge-proxy
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
        image: ${IMAGE}
        imagePullPolicy: Always
        name: edge-proxy
        resources:
          limits:
            cpu: "4"
            memory: 1G
          requests:
            cpu: 100m
            memory: 100M
        volumeMounts:
        - mountPath: /etc/localtime
          name: volume-localtime
        - mountPath: /etc/kubernetes/
          name: cache
        - mountPath: /var/log/edge-proxy/
          name: logdir
      - args:
        - --timeout=3600
        command:
        - /usr/local/bin/benchmark
        image: ${IMAGE}
        imagePullPolicy: Always
        name: bench-mark
        ports:
        - containerPort: 9080
          protocol: TCP
        resources: {}
        securityContext:
          privileged: true
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /etc/localtime
          name: volume-localtime
        - mountPath: /etc/kubernetes/
          name: cache
        - mountPath: /var/log/edge-proxy/
          name: logdir
        - mountPath: /var/log/logtar/
          name: logtar
      dnsPolicy: ClusterFirst
      enableServiceLinks: true
      imagePullSecrets:
      - name: ${imageSecretName} 
      priorityClassName: system-node-critical
      restartPolicy: Always
      serviceAccountName: benchmark
      tolerations:
      - operator: Exists
      volumes:
      - hostPath:
          path: /etc/localtime
          type: ""
        name: volume-localtime
      - emptyDir: {}
        name: cache
      - emptyDir: {}
        name: logdir
      - hostPath:
          path: /data/benchmark/
          type: ""
        name: logtar

EOF

}

create_manifest
