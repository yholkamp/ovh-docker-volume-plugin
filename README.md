# OVH Docker Volume Plugin

Docker volume plugin that supports the 'Additional Disks' / volumes offered by OVH as part of their public cloud offerings.
While OVH uses OpenStack and their additional disks are essentially OpenStack Cinder block storage volumes, their volumes are not accessible over iSCSI.
 
To get around this, I've modified [j-griffith's Cinder docker driver](https://github.com/j-griffith/cinder-docker-driver) 
to use the OVH API, rather than their OpenStack API, to natively mount these volumes without iSCSI usage. 

## Limitations:

* Volumes have to be at least 10GB in size when created using the OVH API, while the minimum when using the OpenStack API is 1GB.

# Install

* Generate a new token [using this link](https://api.ovh.com/createToken/?GET=/cloud/project/*/volume*&POST=/cloud/project/*/volume*&GET=/cloud/project/*/instance&DELETE=/cloud/project/*/volume/*), this will:
    * grant GET & POST access to the volume APIs, allowing us to create volumes & attach them to servers,
    * grant DELETE access to the volume API, allowing us to delete volumes,
    * grant GET access to the instances in your project, allowing the plugin to determine the id of the server. 
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
    
    $ docker run -v myVolume:/Data --volume-driver=ovh -i -t ubuntu /bin/bash

Attach a volume to a Docker Swarm mode Service:

    $ docker service create --name redis --mount type=volume,src=redis,dst=/data,volume-driver=ovh redis:alpine redis-server --appendonly yes

# TODO

* Find the server ID using the ip address & OVH API, rather than manually specifying
* Implement the v2 plugin API, which returns a relative path rather than an absolute one
