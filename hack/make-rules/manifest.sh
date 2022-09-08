#!/usr/bin/env bash

# Copyright 2020 The OpenYurt Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0 #
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -x

YURT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
YURT_OUTPUT_DIR=${YURT_ROOT}/_output
LOCAL_KUBECONFIG="$HOME/.kube/cloudnative-challenge/config"

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

if [ ! -f ${LOCAL_KUBECONFIG} ]; then
	echo "Must support kubeconfig file: ${LOCAL_KUBECONFIG}"
	exit 1
fi

which kubectl
if [ "$?" != "0" ]; then
    echo "Must install kubectl command"
    exit 1
fi


IMAGE="$1"
DOCKER_USERNAME="$2"
DOCKER_PASSWD="$3"

KUBECTL="kubectl --kubeconfig=${LOCAL_KUBECONFIG}"


function create_manifest() {
    
    mkdir -p ${YURT_OUTPUT_DIR}

    local output_file=${YURT_OUTPUT_DIR}/manifest.yaml
    local docker_server=$(echo ${IMAGE} |awk -F '/' '{printf $1}')
    # machine already login
#    docker login --username=${DOCKER_USERNAME}  --password=${DOCKER_PASSWD} ${docker_server}
#    if [ "$?" != "0" ]; then
#        echo "docker login failure"
#        exit 1
#    fi

    local imageSecretName="benchmark-image"
    local server_addr=$(${KUBECTL} config view --minify -o=jsonpath="{.clusters[*].cluster.server}")

    local ns=$(${KUBECTL} get cm kube-root-ca.crt -o=jsonpath="{.metadata.namespace}")
    
    ${KUBECTL} get secret ${imageSecretName} -n ${ns}
    if [ "$?" != "0" ]; then
        echo "create secret ${imageSecretName}"
        ${KUBECTL} create secret docker-registry -n ${ns} ${imageSecretName} --docker-server=${docker_server} --docker-username=${DOCKER_USERNAME} --docker-password="${DOCKER_PASSWD}" --docker-email=benchmark@alibaba-inc.com
    fi 

cat << EOF > ${output_file}

---
apiVersion: v1
kind: Pod 
metadata:
  labels:
    localtest: benchmark-pod
  name: benchmark 
  namespace: ${ns}
spec:
  containers:
  - args:
    - --v=4
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
      requests:
        memory: "100Mi"
        cpu: "100m"
      limits:
        memory: "1000Mi"
        cpu: "1000m"
    volumeMounts:
    - mountPath: /etc/localtime
      name: volume-localtime
    - mountPath: /etc/kubernetes/
      name: cache
    - mountPath: /var/log/edge-proxy/
      name: logdir
  - args:
    - --timeout=3600
    - --namespace=${ns}
    command:
    - /usr/local/bin/benchmark
    image: ${IMAGE}
    imagePullPolicy: Always
    resources:
      requests:
        memory: "100Mi"
        cpu: "100m"
      limits:
        memory: "200Mi"
        cpu: "200m"
    name: bench-mark
    ports:
    - containerPort: 9080
      protocol: TCP
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
  restartPolicy: Never 
  serviceAccount: benchmark 
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

    echo "create manifest file ${output_file}"
}

create_manifest
