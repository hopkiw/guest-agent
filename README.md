# Guest Agent for Google Compute Engine

This repo contains the guest agent and metadata script runner components
installed on Google supported Google Compute Engine
[images](https://cloud.google.com/compute/docs/images).

**Table of Contents**

*  [Overview](#overview)
    * [Logging](#logging)
    * [Linux account management](#linux-account-management)
    * [Clock Skew](#clock-skew)
    * [Network](#network)
*  [Instance Setup Actions](#instance-setup-actions)
*  [Metadata Scripts](#metadata-scripts)
*  [Configuration](#configuration)

## Overview


The **guest agent** takes on-guest actions needed to support GCE functionality.

By default the agent manages the following on Linux:

* On-boot system configuration
* SSH-key-in-metadata based automatic user and group management.
* OSLogin based user and group management.
* Network interface configuration and IP forwarding support.
* System clock syncing.

And manages the following on Windows:

* Network interface configuration and IP forwarding support.
* Password reset and account creation.
* Provide a health check agent for Windows Server Failover Clustering

https://cloud.google.com/compute/docs/tutorials/running-windows-server-failover-clustering

On both operating systems, the metadata script runner implements support for
running user provided startup scripts and shutdown scripts.

Packaging configuration files are provided for Debian, RPM and Googet packages.
On Linux, these packages also contain systemd (or upstart, on EL6 systems)
configurations for running the agent and the metadata scripts.

The Linux guest environment is written in Go. The design of the agent is
detailed in the sections below.

## Technical details

### Packages

The packaging configuration in this repo produces the following packages:

* google-guest-agent-$VERSION.el$DIST.rpm, where DIST can be 6, 7, or 8
* google-guest-agent_$VERSION_amd64.deb
* google-compute-engine-windows.x86_64.$VERSION.goo
* google-compute-engine-metadata-scripts.x86_64.$VERSION.goo

### Logging

The guest agent and metadata script runner use the guest-logging-go library to
log to the serial port, the relevant system logger (syslog or windows event
log), and google cloud logging. On systems running systemd, the syslog output will be
captured in the systemd journal and can be viewed with `journalctl -u
google-guest-agent`.

## Features

### Linux account management

The guest agent is responsible for provisioning and deprovisioning user
accounts. For users with SSH keys in metadata, the agent creates a local user
account and generates an authorized keys file to permit SSH login. User account
creation is based on 
[adding and removing SSH Keys](https://cloud.google.com/compute/docs/instances/adding-removing-ssh-keys)
stored in metadata. The authorized keys file for a Google managed user is
deleted when all SSH keys for the user are removed from metadata.

All users provisioned by the account daemon are added to the `google-sudoers`
group and the group is configured to provide root access via sudo.

User accounts not managed by Google are not modified by the accounts daemon.

### Clock Skew

The guest agent is responsible for syncing the software clock with the hypervisor
clock after a stop/start event or after a migration. Preventing clock skew may
result in `system time has changed` messages in VM logs.

### Network

The guest agent uses network interface metadata to manage the network interfaces
in the guest. This involves the following:

*   Enabling all network interfaces on boot.
*   Uses IP forwarding metadata to setup or remove IP routes in the guest.
    *   Supports creating routes for forwarded IPs, target IPs and alias IPs.
    *   Only IPv4 IP addresses are currently supported.
    *   Routes are set on the primary network interface.
    *   Google routes are configured, by default, with the routing protocol ID
        `66`. This ID is a namespace for daemon configured IP addresses.

Links for these three types of IPs?

### Instance Setup Actions

Instance setup actions run during VM boot. The script configures the Linux guest
environment by performing the following tasks.

*   Optimize for local SSD.
*   Enable multi-queue on all the virtionet devices.
*   Wait for network availability.
*   Set SSH host keys the first time the instance is booted.
*   Set the `boto` config for using Google Cloud Storage.
*   Create the defaults configuration file.

### Metadata Scripts

Metadata scripts implement support for running user provided
[startup scripts](https://cloud.google.com/compute/docs/startupscript) and
[shutdown scripts](https://cloud.google.com/compute/docs/shutdownscript). The
guest support for metadata scripts is implemented in Python with the following
design details.

*   Metadata scripts are executed in a shell.
*   If multiple metadata keys are specified (e.g. `startup-script` and
    `startup-script-url`) both are executed.
*   If multiple metadata keys are specified (e.g. `startup-script` and
    `startup-script-url`) a URL is executed first.
*   The exit status of a metadata script is logged after completed execution.

### OS Login

Although the binary components which enable OS Login to function are stored in
their own repository, the guest agent is currently responsible for making the
necessary configuration changes to the system such as modifying the NSS, SSHD
and PAM configurations.

https://github.com/GoogleCloudPlatform/guest-oslogin/

https://cloud.google.com/compute/docs/oslogin/

## Configuration

Users of Google provided images may configure the guest environment behaviors
using a configuration file. To make configuration changes, add or change
settings in `/etc/default/instance_configs.cfg`. If you are attempting to change
the behavior of a running instance, restart the guest agent after creating or
editing the file.

Previously users were directed to edit the file
`/etc/default/instance_configs.cfg.template`. For this reason, this file is
still supported, though its use is discouraged.

The following are valid user configuration options. The configuration file we
ship with packages is here.

Section           | Option                 | Value
----------------- | ---------------------- | -----
Accounts          | deprovision\_remove    | `true` makes deprovisioning a user destructive.
Accounts          | groups                 | Comma separated list of groups for newly provisioned users.
Accounts          | useradd\_cmd           | Command string to create a new user.
Accounts          | userdel\_cmd           | Command string to delete a user.
Accounts          | gpasswd\_add\_cmd      | Command string to add a user to a group.
Accounts          | gpasswd\_remove\_cmd   | Command string to remove a user from a group.
Accounts          | groupadd\_cmd          | Command string to create a new group.
Daemons           | accounts\_daemon       | `false` disables the accounts daemon.
Daemons           | clock\_skew\_daemon    | `false` disables the clock skew daemon.
Daemons           | network\_daemon        | `false` disables the network daemon.
InstanceSetup     | host\_key\_types       | Comma separated list of host key types to generate.
InstanceSetup     | optimize\_local\_ssd   | `false` prevents optimizing for local SSD.
InstanceSetup     | network\_enabled       | `false` skips instance setup functions that require metadata.
InstanceSetup     | set\_boto\_config      | `false` skips setting up a `boto` config.
InstanceSetup     | set\_host\_keys        | `false` skips generating host keys on first boot.
InstanceSetup     | set\_multiqueue        | `false` skips multiqueue driver support.
IpForwarding      | ethernet\_proto\_id    | Protocol ID string for daemon added routes.
IpForwarding      | ip\_aliases            | `false` disables setting up alias IP routes.
IpForwarding      | target\_instance\_ips  | `false` disables internal IP address load balancing.
MetadataScripts   | default\_shell         | String with the default shell to execute scripts.
MetadataScripts   | run\_dir               | String base directory where metadata scripts are executed.
MetadataScripts   | startup                | `false` disables startup script execution.
MetadataScripts   | shutdown               | `false` disables shutdown script execution.
NetworkInterfaces | setup                  | `false` skips network interface setup.
NetworkInterfaces | ip\_forwarding         | `false` skips IP forwarding.
NetworkInterfaces | dhcp\_command          | String path for alternate dhcp executable used to enable network interfaces.

Keys in the 'Daemons' section retain their names from an earlier edition of
software that existed as multiple independent daemons, but function as
described.

### Interaction of settings:
Setting `network_enabled` to `false` will skip setting up host keys and the
`boto` config in the guest. As a result, the `set_boto_config` and
`set_host_keys` keys have no meaning in this case. The setting will also prevent
startup and shutdown script execution, rendering the `default_shell`, `run_dir`
`startup` and `shutdown` keys meaningless.
