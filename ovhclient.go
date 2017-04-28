package main

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/ovh/go-ovh/ovh"
)

type OVHClient struct {
	Client *ovh.Client
	Conf   *Config
}

type Volume struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	AttachedTo  []string `json:"attachedTo"`
	Status      string   `json:"status"`
}

// POST data used to create a new volume
type VolumePost struct {
	Region string `json:"region"` // required
	Size   int    `json:"size"`   // required, size in GBs
	Type   string `json:"type"`   // required

	Name        string `json:"name"`
	Description string `json:"description"`
	ImageId     string `json:"imageId"`
	SnapshotId  string `json:"snapshotId"`
}

// POST data used for volume attaching & detaching
type VolumeAttachmentPost struct {
	InstanceId string `json:"instanceId"`
}

type GenericApiResponse struct {
	Message string `json:"message"`
}

type Instance struct {
	Status      string       `json:"status"`
	Region      string       `json:"region"`
	Name        string       `json:"name"`
	ImageId     string       `json:"imageId"`
	Created     string       `json:"created"`
	FlavorId    string       `json:"flavorId"`
	SshKeyId    string       `json:"sshKeyId"`
	Id          string       `json:"id"`
	IpAddresses []InstanceIp `json:"ipAddresses"`
}

type InstanceIp struct {
	GatewayIp string `json:"gatewayIp"`
	NetworkId string `json:"networkId"`
	Version   int    `json:"version"`
	Ip        string `json:"ip"`
	Type      string `json:"type"`
}

func (oc OVHClient) ListVolumes() (volumes []Volume, error error) {
	volumes = []Volume{}
	url := fmt.Sprintf("/cloud/project/%s/volume", oc.Conf.ProjectId)
	log.Debugf("Retrieving %s", url)
	if err := oc.Client.Get(url, &volumes); err != nil {
		fmt.Printf("Error: %q", err)
		return volumes, errors.New(fmt.Sprintf("Could not retrieve volumes: %s", err.Error()))
	}

	return
}

func (oc OVHClient) GetVolumeByName(name string) (vol Volume, err error) {
	vol = Volume{}
	volumes, err := oc.ListVolumes()
	if err != nil {
		return vol, err
	}

	for _, element := range volumes {
		if element.Name == name {
			return element, nil
		}
	}

	return vol, nil
}

func (oc OVHClient) CreateVolume(createVolumeOptions VolumePost) (volume Volume, err error) {
	log.Debugf("Creating volume with options: %+v", createVolumeOptions)

	createUrl := fmt.Sprintf("/cloud/project/%s/volume", oc.Conf.ProjectId)
	log.Debugf("Sending POST to %s", createUrl)
	volume = Volume{}
	if err := oc.Client.Post(createUrl, createVolumeOptions, &volume); err != nil {
		fmt.Printf("Error: %q\n", err)
		return volume, errors.New(fmt.Sprintf("Error while creating volume %s, %s", createVolumeOptions.Name, err))
	}

	return volume, err
}

func (oc OVHClient) DeleteVolume(volumeId string) error {
	deleteUrl := fmt.Sprintf("/cloud/project/%s/volume/%s", oc.Conf.ProjectId, volumeId)
	deleteResponse := GenericApiResponse{}
	log.Debugf("Sending DELETE to %s", deleteUrl)
	if err := oc.Client.Delete(deleteUrl, &deleteResponse); err != nil {
		log.Errorf("Failed to delete volume %s: %s. %s", volumeId, err.Error(), deleteResponse)
		return errors.New(fmt.Sprintf("Failed to delete %s: %s", volumeId, err.Error()))
	}
	log.Debugf("Response from Delete: %+v\n", deleteResponse)

	return nil
}

func (oc OVHClient) AttachVolume(volumeId string) (volume Volume, err error) {
	//volume = Volume{}
	attachRequest := VolumeAttachmentPost{
		InstanceId: oc.Conf.ServerId,
	}
	attachUrl := fmt.Sprintf("/cloud/project/%s/volume/%s/attach", oc.Conf.ProjectId, volumeId)
	log.Debugf("Sending POST to %s", attachUrl)
	if err = oc.Client.Post(attachUrl, attachRequest, &volume); err != nil {
		fmt.Printf("Error: %q\n", err)
		return
	}
	log.Debugf("Received attach response: %s", volume)

	return
}

func (oc OVHClient) DetachVolume(volumeId string) (volume Volume, err error) {
	detachRequest := VolumeAttachmentPost{
		InstanceId: oc.Conf.ServerId,
	}
	detachUrl := fmt.Sprintf("/cloud/project/%s/volume/%s/detach", oc.Conf.ProjectId, volumeId)
	log.Debugf("Sending POST to %s", detachUrl)
	if err = oc.Client.Post(detachUrl, detachRequest, &volume); err != nil {
		fmt.Printf("Error: %q\n", err)
		return
	}
	log.Debugf("Received detach response: %s", volume)

	return
}

func (oc OVHClient) ListInstances() (instances []Instance, error error) {
	url := fmt.Sprintf("/cloud/project/%s/instance", oc.Conf.ProjectId)
	log.Debugf("GET for %s", url)
	if err := oc.Client.Get(url, &instances); err != nil {
		fmt.Printf("Error: %q", err)
		return instances, errors.New(fmt.Sprintf("Could not retrieve instances: %s", err.Error()))
	}

	return
}

func (oc OVHClient) GetInstanceByIps(ips []string) (instance Instance, error error) {
	instances, err := oc.ListInstances()
	if err != nil {
		log.Errorf("Could not get instances: %s", err)
		return instance, err
	}
	// loop over all instances, ip addresses and see if there's any overlap
	for _, i := range instances {
		log.Debugf("Checking instance %s's ips against %s", i.Name, ips)
		for _, ipAddress := range i.IpAddresses {
			if contains(ips, ipAddress.Ip) {
				return i, nil
			}
		}
	}
	return
}

// checks if s contains e
func contains(xs []string, e string) bool {
	for _, x := range xs {
		if x == e {
			return true
		}
	}
	return false
}
