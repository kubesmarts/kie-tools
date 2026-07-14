#!/bin/sh
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.


set -o errexit

container_engine="$1"
reg_name='kind-registry'
reg_port='5001'

# 0. Raise inotify limits so the registry container and Kind pods don't hit
#    "too many open files". The default max_user_instances (128) is too low
#    for a multi-container cluster running a local registry.
current_instances=$(cat /proc/sys/fs/inotify/max_user_instances 2>/dev/null || echo 0)
if [ "${current_instances}" -lt 1024 ]; then
  echo "Raising fs.inotify.max_user_instances from ${current_instances} to 8192"
  sysctl -w fs.inotify.max_user_instances=8192 || \
    echo "WARNING: could not raise fs.inotify.max_user_instances – registry may crash with 'too many open files'"
fi
current_watches=$(cat /proc/sys/fs/inotify/max_user_watches 2>/dev/null || echo 0)
if [ "${current_watches}" -lt 524288 ]; then
  echo "Raising fs.inotify.max_user_watches from ${current_watches} to 524288"
  sysctl -w fs.inotify.max_user_watches=524288 || true
fi

# 1. Create kind cluster with containerd registry config dir enabled
# TODO: kind will eventually enable this by default and this patch will
# be unnecessary.
#
# See:
# https://github.com/kubernetes-sigs/kind/issues/2875
# https://github.com/containerd/containerd/blob/main/docs/cri/config.md#registry-configuration
# See: https://github.com/containerd/containerd/blob/main/docs/hosts.md
cat <<EOF | kind create cluster -n kind --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        system-reserved: memory=2Gi
- role: worker
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        system-reserved: memory=4Gi
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
EOF

# 2. Wait for kube system pods to reach running state
if ! kubectl wait -n kube-system --for=condition=ready pods --all --timeout=120s ; then
  echo "some pods in the system are not running"
  kubectl get pods -A -o wide || true
  exit 1
fi

# 3. Create registry container directly on the kind network so its IP is
#    stable across Docker-managed restarts (--restart=always re-attaches to
#    the same network, preserving the IP assignment).
${container_engine} run \
  -d --restart=always -p "127.0.0.1:${reg_port}:5000" --network kind --name "${reg_name}" \
  registry:2

# 4. Add the registry config to the nodes
#
# This is necessary because localhost resolves to loopback addresses that are
# network-namespace local.
# In other words: localhost in the container is not localhost on the host.
#
# We want a consistent name that works from both ends, so we tell containerd to
# alias localhost:${reg_port} to the registry container when pulling images

# Retrieve IP address of the container connected to the cluster network
IP_ADDRESS=$("${container_engine}" inspect --format='{{(index (index .NetworkSettings.Networks "kind") ).IPAddress}}' ${reg_name})

REGISTRY_DIR="/etc/containerd/certs.d/${IP_ADDRESS}:5000"
for node in $(kind get nodes); do
  ${container_engine} exec "${node}" mkdir -p "${REGISTRY_DIR}"
  cat <<EOF | "${container_engine}" exec -i "${node}" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://${IP_ADDRESS}:5000"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF
done

# 5. Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    hostFromClusterNetwork: "${IP_ADDRESS}:5000"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
