#!/bin/sh
set -e
#
# This script provides a mechanism for easy installation of the
# cinder-docker-driver, use with curl or wget:
#  'curl -sSl https://raw.githubusercontent.com/yholkamp/ovh-docker-volume-plugin/master/install.sh | sh''
# or
#  'wget -qO- https://raw.githubusercontent.com/yholkamp/ovh-docker-volume-plugin/master/install.sh | sh'

BIN_NAME=ovh-docker-volume-plugin
DRIVER_URL="https://github.com/yholkamp/ovh-docker-volume-plugin/releases/download/v0.10/$BIN_NAME"
BIN_DIR="/home/core" # Set to /usr/bin for most distributions
CONFIG_LOCATION="/etc/ovh-docker-config.json"

do_install() {
touch $CONFIG_LOCATION
mkdir -p /var/lib/ovh-volume-plugin/mount
rm $BIN_DIR/$BIN_NAME || true
cp $BIN_NAME $BIN_DIR/$BIN_NAME
curl -sSL -o $BIN_DIR/$BIN_NAME $DRIVER_URL
chmod +x $BIN_DIR/$BIN_NAME
echo "
[Unit]
Description=\"OVH Docker Volume Plugin daemon\"
Before=docker.service
Requires=ovh-docker-volume-plugin.service

[Service]
TimeoutStartSec=0
ExecStart=$BIN_DIR/$BIN_NAME -config $CONFIG_LOCATION &

[Install]
WantedBy=docker.service" >/etc/systemd/system/ovh-docker-volume-plugin.service

chmod 644 /etc/systemd/system/ovh-docker-volume-plugin.service
systemctl daemon-reload
systemctl enable ovh-docker-volume-plugin
}

do_install

echo "OVH docker volume plugin installed, your configuration file is located at:"
echo "/etc/ovh-docker-config.json"