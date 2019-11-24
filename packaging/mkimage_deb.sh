#!/bin/bash
# Copyright 2019 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

function try_command() {
  n=0
  while ! "$@"; do
    echo "try $n to run $@"
    if [[ n -gt 3 ]]; then
      return 1
    fi
    ((n++))
    sleep 5
  done
}

function err() {
  echo "Image build failed: $@"
  exit 1
}

function exit_err() {
  err "$0:$1 $BASH_COMMAND"
}

trap 'exit_err $LINENO' ERR

URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_DIR=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-dir)

. /etc/os-release
if [[ $ID != "debian" ]]; then
  err "Not a debian host"
fi

for service in network accounts clock-skew; do
  if [[ -d /var/run/systemd ]]; then
    systemctl disable google-${service}-daemon.service
  elif [[ -f google-shutdown-scripts.conf ]]; then
    initctl stop google-${service}-daemon
    rm /etc/init/google-${service}-daemon.conf
  fi
done
rm -f /etc/default/instance_configs.cfg

gceobject="google-compute-engine*.deb"
agentobject="google-guest-agent*.deb"
gsutil cp "${GCS_DIR}/${agentobject}" ./
gsutil cp "${GCS_DIR}/${gceobject}" ./
DEBIAN_FRONTEND=noninteractive apt install -y ./${gceobject} ./${agentobject}
DEBIAN_FRONTEND=noninteractive apt purge -y python*google-compute-engine
systemctl stop google-guest-agent

rm -f /etc/sudoers.d/google*
rm -rf /var/lib/google
rm -f /etc/boto.cfg
rm -f /etc/instance_id
userdel -rf liamh || :
try_command passwd -d root

sync
sleep 30
echo "Image build success"
echo o > /proc/sysrq-trigger
