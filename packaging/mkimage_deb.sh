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

. /etc/os-release
if [[ $ID != "debian" ]]; then
  err "Not a debian host"
fi

mkdir chroot_dir
mount /dev/sdb1 chroot_dir

gceobject="google-compute-engine*.deb"
agentobject="google-guest-agent*.deb"
gsutil cp "${GCS_DIR}/${agentobject}" ./chroot_dir/
gsutil cp "${GCS_DIR}/${gceobject}" ./chroot_dir/

cat >./chroot_dir/setup.sh <<"EOF"
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


rm -f /etc/sudoers.d/google*
rm -rf /var/lib/google
rm -f /etc/boto.cfg
rm -f /etc/instance_id
rm -f /etc/ssh/ssh_host_*key*
rm -f /etc/default/instance_configs.cfg
DEBIAN_FRONTEND=noninteractive apt install -y /*.deb
try_command passwd -d root
for f in /usr/share/initramfs-tools/scripts/local-{premount/expand_rootfs,bottom/xfs_growfs}; do
[[ -f $f ]] && sed -i"" 's/log_failure_message/log_failure_msg/' "$f"
done
f=/usr/share/initramfs-tools/scripts/expandfs-lib.sh
[[ -f $f ]] && sed -i"" '33d' "$f"
/usr/sbin/update-initramfs -u
EOF

chmod +x ./chroot_dir/setup.sh
chroot ./chroot_dir /setup.sh | tee ./chroot_dir/install.log

sync
umount chroot_dir
sync

sleep 30
echo "Image build success"

echo o > /proc/sysrq-trigger
