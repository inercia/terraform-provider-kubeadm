#cloud-config

# set locale
locale: en_GB.UTF-8

# set timezone
timezone: Etc/UTC

# we must be careful with repos/packages updates: could abort
# any concurrent installation we could do on command line...
repo_update: true

# set root password
chpasswd:
  list: |
    root:linux
  expire: False

ssh_authorized_keys:
  - ${public_key}

runcmd:
  - [modprobe, br_netfilter]
  - [sh, -c, 'echo 1 > /proc/sys/net/bridge/bridge-nf-call-iptables']
  - [sysctl, -w, net.ipv4.ip_forward=1]

final_message: "The system is finally up, after $UPTIME seconds"


