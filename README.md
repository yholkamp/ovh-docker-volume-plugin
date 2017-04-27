# OVH Docker Volume Plugin

Docker volume plugin that supports the 'Additional Disks' / volumes offered by OVH as part of their public cloud offerings.
While OVH uses OpenStack and their additional disks are essentially OpenStack Cinder block storage volumes, their volumes are not accessible over iSCSI.
 
To get around this, I've modified [j-griffith's Cinder docker driver](https://github.com/j-griffith/cinder-docker-driver) 
to use the OVH API, rather than their OpenStack API, to 'natively' mount these volumes. 

Limitations:

* Volumes have to be at least 10GB in size when created using the OVH API, while the minimum when using the OpenStack API is 1GB.

# Usage

* Generate a new token [using this link](https://api.ovh.com/createToken/?GET=/cloud/project/*/volume*&POST=/cloud/project/*/volume*&GET=/cloud/project/*/instance)
* Create a config file: `config.json` with the following contents:

    {
    TODO
    }



# Thanks

