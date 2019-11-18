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

trap 'err $LINENO' ERR

URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_DIR=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-dir)

if [[ ! -f /etc/redhat-release ]]; then
  err "Not a Enterprise Linux host"
fi

if [[ ! -f /etc/os-release ]]; then
  VERSION_ID=6
else
  . /etc/os-release
fi

for service in network accounts clock-skew; do
  if [[ -d /var/run/systemd ]]; then
    systemctl disable google-${service}-daemon.service
  elif [[ -f google-shutdown-scripts.conf ]]; then
    initctl stop google-${service}-daemon
    rm /etc/init/google-${service}-daemon.conf
  fi
done

rm -f /etc/sudoers.d/google*
rm -rf /var/lib/google
userdel -rf liamh || :
try_command passwd -d root

# Upgrade GCE to break dependency.
object="google-compute-engine*.deb"
gsutil cp "${GCS_DIR}/${object}" ./
rpm -Uvh ./$object

# Remove python altogether.
python=$(rpmquery -a|grep -iE 'python.?-google-compute-engine')
[[ -n "$python" ]] && rpm -e "$python"

object="google-guest-agent*el${VERSION_ID/.*}*.rpm"
gsutil cp "${GCS_DIR}/${object}" ./

rpm -Uvh ./$object

echo "Image build success"
sleep 30
echo o > /proc/sysrq-trigger
