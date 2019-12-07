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
GCS_DIR="$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-dir|sed 's#/\+$##')"

if [[ ! -f /etc/redhat-release ]]; then
  err "Not a Enterprise Linux host"
fi

if [[ ! -f /etc/os-release ]]; then
  VERSION_ID=6
else
  . /etc/os-release
  VERSION_ID="${VERSION_ID/.*}"
fi

if [[ "$VERSION_ID" -gt 6 ]]; then
  opts="rw,nouuid"
else
  opts="rw"
fi
mkdir chroot_dir
mount -o $opts /dev/sdb1 chroot_dir


# Upgrade GCE to break dependency, adds dep on guest-agent.
object="google-compute-engine*el${VERSION_ID}*.rpm"
gsutil cp "${GCS_DIR}/${object}" ./chroot_dir/
object="google-guest-agent*el${VERSION_ID}*.rpm"
gsutil cp "${GCS_DIR}/${object}" ./chroot_dir/


cat > ./chroot_dir/setup.sh <<"EOF"
#!/bin/bash

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

mknod -m 666 /dev/random c 1 8
mknod -m 666 /dev/urandom c 1 9
mount -t proc none /proc
rpm -Uvh /*rpm
rm -f /etc/boto.cfg
rm -f /etc/sudoers.d/google*
rm -rf /var/lib/google
rm -f /etc/instance_id
rm -f /etc/ssh/ssh_host_*key*
sed -i"" 's/SELINUX=enforcing/SELINUX=permissive/' /etc/selinux/config
try_command passwd -d root
umount /proc
EOF

chmod +x ./chroot_dir/setup.sh
chroot ./chroot_dir /setup.sh | tee ./chroot_dir/install.log

sync
umount chroot_dir
sync

sleep 30
echo "Image build success"
sleep 30

echo o > /proc/sysrq-trigger
