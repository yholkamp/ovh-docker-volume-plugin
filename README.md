# OVH Docker Volume Plugin

Docker volume plugin that supports the 'Additional Disks' / volumes offered by OVH as part of their public cloud offerings.
Depending on the operating system used on OVH, the volumes may be available over iSCSI or RBD but can always be mounted using the OVH API.
 
To provide a cross-distribution plugin for OVH, I've created this driver after working with [j-griffith's Cinder docker driver](https://github.com/j-griffith/cinder-docker-driver), which relies on iSCSI access.

### Limitations:

* Volumes have to be at least 10GB in size when created using the OVH API, while the minimum when using the OpenStack API is 1GB.

# Install

* Generate a new token [using this link](https://api.ovh.com/createToken/?GET=/cloud/project/*/volume*&POST=/cloud/project/*/volume*&GET=/cloud/project/*/instance&DELETE=/cloud/project/*/volume/*), this will:
    * grant GET & POST access to the volume APIs, allowing us to create volumes & attach them to servers,
    * Optional: grant GET access to the instances in your project, allowing the plugin to determine the id of the server.
    * Optional: grant DELETE access to the volume API, allowing us to delete volumes,
    * Note that when your token expires, you will have to update your configuration file, so pick a suitable expiration period.
* Create your own `ovh-docker-config.json` file using `config.example.json` as template.

## Pre-built installation

* Copy the install script to your server: `curl -sSl https://raw.githubusercontent.com/yholkamp/ovh-docker-volume-plugin/master/install.sh`
* Verify the install script is what you expected, optionally modify the paths, and run `sudo sh install.sh`
* Upload your config file to `/etc/ovh-docker-config.json` on your server.

## Installation from source

    git clone https://github.com/yholkamp/ovh-docker-volume-plugin
    cd ovh-docker-volume-plugin
    go build
    sudo ./install.sh

# Using the plugin

Create a new volume:

    $ docker volume create -d ovh --name myVolume -o size=10
    
Interactively connect with a volume:
    
    $ docker run -v myVolume:/Data --volume-driver=ovh -i -t bash

Attach a volume to a Docker Swarm mode Service:

    $ docker service create --name redis --mount type=volume,src=redis,dst=/data,volume-driver=ovh redis:alpine redis-server --appendonly yes

# TODO

* Find the server ID using the ip address & OVH API, rather than manually specifying
* Implement the v2 plugin API, which returns a relative path rather than an absolute one
