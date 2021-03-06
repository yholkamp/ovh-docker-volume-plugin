package main

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/ovh/go-ovh/ovh"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

const (
	VOLUME_TYPE_CLASSIC    = "classic"
	VOLUME_TYPE_HIGH_SPEED = "high-speed"
)

type Config struct {
	SocketGroup    string //User group to use for the plugin socket
	DefaultVolSz   int
	DefaultVolType string
	DefaultRegion  string

	MountPoint string
	ProjectId  string
	ServerId   string

	// OVH API settings
	ApplicationKey    string
	ApplicationSecret string
	ConsumerKey       string
	OVHEndpoint       string
}

type OVHPlugin struct {
	Mutex  *sync.Mutex
	Conf   *Config
	Client *OVHClient
}

func processConfig(cfg string) (Config, error) {
	var conf Config
	content, err := ioutil.ReadFile(cfg)
	if err != nil {
		log.Fatal("Error reading config file: ", err)
	}
	err = json5.Unmarshal(content, &conf)
	if err != nil {
		log.Fatal("Error parsing json config file: ", err)
	}
	if conf.MountPoint == "" {
		conf.MountPoint = "/var/lib/ovh-volume-plugin/mount"
	}
	if conf.OVHEndpoint == "" {
		conf.OVHEndpoint = "ovh-eu"
	}
	// set the default SocketGroup to root, which should work on most Linuxes
	if conf.SocketGroup == "" {
		conf.SocketGroup = "root"
	}
	if conf.DefaultVolSz < 10 {
		conf.DefaultVolSz = 10
	}
	if conf.DefaultVolType == "" {
		conf.DefaultVolType = VOLUME_TYPE_CLASSIC
	}
	log.Infof("Using config file: %s", cfg)
	log.Infof("Set DefaultVolSz to: %d GiB", conf.DefaultVolSz)
	log.Infof("Set DefaultVolType to: %s", conf.DefaultVolType)
	log.Infof("Set OVHEndpoint to: %s", conf.OVHEndpoint)
	log.Infof("Set SocketGroup to: %s", conf.SocketGroup)
	return conf, nil
}

func New(cfgFile string) OVHPlugin {
	conf, err := processConfig(cfgFile)
	if err != nil {
		log.Fatal("Error processing OVH docker volume plugin config file: ", err)
	}

	_, err = os.Lstat(conf.MountPoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(conf.MountPoint, 0755); err != nil {
			log.Fatalf("Failed to create mount directory during driver init: %v", err)
		}
	}

	ovhClient, err := ovh.NewClient(conf.OVHEndpoint, conf.ApplicationKey, conf.ApplicationSecret, conf.ConsumerKey)
	if err != nil {
		log.Fatalf("Error: %q\n", err)
	}
	ovhWrapper := OVHClient{Conf: &conf, Client: ovhClient}

	if conf.ServerId == "" {
		log.Debug("No ServerId configured")
		ips, err := getIpAddresses()
		if err != nil {
			log.Fatalf("No server id defined and could not find ip addresses for this server: %s", err)
		} else {
			instance, err := ovhWrapper.GetInstanceByIps(ips)
			if err != nil {
				log.Fatalf("Could not find the instance matching this server's IP: %s", err.Error())
			} else {
				log.Infof("Set ServerId to %s based on server ip", instance.Id)
				conf.ServerId = instance.Id
			}
		}
	}

	d := OVHPlugin{
		Conf:   &conf,
		Mutex:  &sync.Mutex{},
		Client: &ovhWrapper,
	}
	log.Debug("Finished driver initialization")
	return d
}

// Parses the user provided volume creation options and creates an OVH API object
func (d OVHPlugin) parseOpts(r volume.Request) VolumePost {
	opts := VolumePost{
		Type:        d.Conf.DefaultVolType,
		Size:        d.Conf.DefaultVolSz,
		Region:      d.Conf.DefaultRegion,
		Name:        r.Name,
		Description: "Docker volume.",
	}
	for k, v := range r.Options {
		log.Debugf("Option: %s = %s", k, v)
		switch k {
		case "size":
			vSize, err := strconv.Atoi(v)
			if err == nil {
				opts.Size = vSize
			}
		case "type":
			if r.Options["type"] != "" {
				opts.Type = v
			}
		}
	}
	return opts
}

func (d OVHPlugin) Create(r volume.Request) volume.Response {
	log.Infof("Create volume %s on OVH", r.Name)
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	vol, err := d.Client.GetVolumeByName(r.Name)
	if err != nil {
		log.Errorf("Error while checking if volume %s already exists: %s", vol, err.Error())
		return volume.Response{Err: fmt.Sprintf("Error while checking if volume %s exists, %s", r.Name, err)}
	}

	// volume does not yet exist
	if vol.Id == "" {
		log.Infof("Did not find a volume with name %s, creating a new one", r.Name)
		createVolumeOptions := d.parseOpts(r)
		log.Debugf("Creating volume with options: %+v", createVolumeOptions)

		if _, err := d.Client.CreateVolume(createVolumeOptions); err != nil {
			return volume.Response{Err: fmt.Sprintf("Error while creating volume %s, %s", r.Name, err)}
		}
	} else if vol.Status != "available" && !contains(vol.AttachedTo, d.Conf.ServerId) {
		return volume.Response{Err: fmt.Sprintf("Volume %s already exists and is not available, state is %s", r.Name, vol.Status)}
	}
	log.Infof("Found an existing volume with name %s, reusing this", r.Name)

	// create a mount point so we can easily track this volume
	path := filepath.Join(d.Conf.MountPoint, r.Name)
	if err := os.Mkdir(path, os.ModeDir); err != nil {
		log.Errorf("Failed to create Mount directory: %v", err)
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{}
}

func (d OVHPlugin) Remove(r volume.Request) volume.Response {
	log.Info("Remove/Delete Volume: ", r.Name)
	vol, err := d.Client.GetVolumeByName(r.Name)
	log.Debugf("Remove/Delete Volume ID: %s", vol.Id)
	if err != nil {
		log.Errorf("Failed to retrieve volume named: ", r.Name, "during Remove operation", err)
		return volume.Response{Err: err.Error()}
	}
	if vol.Id == "" {
		return volume.Response{Err: fmt.Sprintf("Volume with name %s could not be found", r.Name)}
	}
	if vol.Status == "attaching" || vol.Status == "in-use" {
		return volume.Response{Err: fmt.Sprintf("Cannot delete %s while in %s state", r.Name, vol.Status)}
	}
	if err := d.Client.DeleteVolume(vol.Id); err != nil {
		return volume.Response{Err: fmt.Sprintf("Failed to delete %s: %s", r.Name, err.Error())}
	}

	path := filepath.Join(d.Conf.MountPoint, r.Name)
	if err := os.Remove(path); err != nil {
		log.Errorf("Failed to remove Mount directory: %v", err)
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (d OVHPlugin) Path(r volume.Request) volume.Response {
	log.Info("Retrieve path info for volume: `", r.Name, "`")
	path := filepath.Join(d.Conf.MountPoint, r.Name)
	log.Debug("Path reported as: ", path)
	return volume.Response{Mountpoint: path}
}

func (d OVHPlugin) Mount(r volume.Request) volume.Response {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	hostname, _ := os.Hostname()
	log.Infof("Mounting volume %+v on %s", r, hostname)
	vol, err := d.Client.GetVolumeByName(r.Name)
	if err != nil {
		log.Errorf("Failed to retrieve volume named: ", r.Name, "during Mount operation", err)
		return volume.Response{Err: err.Error()}
	}
	if vol.Id == "" {
		return volume.Response{Err: fmt.Sprintf("Volume with name %s could not be found", r.Name)}
	}
	if vol.Status == "creating" {
		// NOTE(jdg):  This may be a successive call after a create which from
		// the docker volume api can be quite speedy.  Take a short pause and
		// check the status again before proceeding
		time.Sleep(time.Second * 5)
		vol, err = d.Client.GetVolumeByName(r.Name)
	}

	if err != nil {
		log.Errorf("Failed to retrieve volume named: ", r.Name, "during Mount operation", err)
		return volume.Response{Err: err.Error()}
	}

	volumeIsAttachedToServer := contains(vol.AttachedTo, d.Conf.ServerId)
	if (vol.Status == "in-use" || vol.Status == "attaching") && volumeIsAttachedToServer {
		// disk is already attached, we can skip the pleasantries
		log.Infof("Disk %s is already attached to %s", vol.Id, d.Conf.ServerId)
	} else if vol.Status != "available" {
		log.Debugf("Volume info: %+v\n", vol)
		errMsg := fmt.Sprintf("Invalid volume status for mount request, volume is: %s but must be available", vol.Status)
		log.Error(errMsg)
		err := errors.New(errMsg)
		return volume.Response{Err: err.Error()}
	}

	// only if the volume is not yet attached, attach it
	if !volumeIsAttachedToServer {
		if _, err := d.Client.AttachVolume(vol.Id); err != nil {
			fmt.Printf("Error: %q\n", err)
			return volume.Response{Err: err.Error()}
		}
	}

	fileName := "/dev/disk/by-id/*" + vol.Id[0:20]
	var device string
	if device = waitForPathToExist(fileName, 60); device == "" {
		return volume.Response{Err: fmt.Sprintf("Waited 60 seconds for volume %s, as device %s, to appear but it never did", vol.Id, device)}
	}
	if GetFSType(device) == "" {
		//TODO(jdg): Enable selection of *other* fs types
		log.Debugf("Formatting device")
		err := FormatVolume(device, "ext4")
		if err != nil {
			err := errors.New("Failed to format device")
			log.Error(err)
			return volume.Response{Err: err.Error()}
		}
	}
	// check if the drive is already present
	if volumeIsAttachedToServer && waitForPathToExist(d.Conf.MountPoint+"/"+r.Name, 1) != "" {
		log.Infof("Volume already mounted")
		return volume.Response{Mountpoint: d.Conf.MountPoint + "/" + r.Name}

		// mount the disk
	} else if mountErr := Mount(device, d.Conf.MountPoint+"/"+r.Name); mountErr != nil {
		err := errors.New("Problem mounting docker volume: " + mountErr.Error())
		log.Error(err)
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: d.Conf.MountPoint + "/" + r.Name}
}

func (d OVHPlugin) Unmount(r volume.Request) volume.Response {
	log.Infof("Unmounting volume: %+v", r)
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	vol, err := d.Client.GetVolumeByName(r.Name)
	if err != nil {
		log.Errorf("Failed to retrieve volume named: `", r.Name, "` during Unmount operation", err)
		return volume.Response{Err: err.Error()}
	}
	if vol.Id == "" {
		log.Infof("Volume with name %s could not be found, we're done here", r.Name)
		return volume.Response{Err: fmt.Sprintf("Volume with name %s could not be found", r.Name)}
	}

	if umountErr := Umount(d.Conf.MountPoint + "/" + r.Name); umountErr != nil {
		if umountErr.Error() == "Volume is not mounted" {
			log.Warning("Request to unmount volume, but it's not mounted")
			return volume.Response{}
		} else {
			return volume.Response{Err: umountErr.Error()}
		}
	}

	if _, err := d.Client.DetachVolume(vol.Id); err != nil {
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{}
}

func (d OVHPlugin) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}

func (d OVHPlugin) Get(r volume.Request) volume.Response {
	log.Info("Get volume: ", r.Name)
	vol, err := d.Client.GetVolumeByName(r.Name)
	if err != nil {
		log.Errorf("Failed to retrieve volume `%s`: %s", r.Name, err.Error())
		return volume.Response{Err: err.Error()}
	}
	if vol.Id == "" {
		return volume.Response{Err: fmt.Sprintf("Volume with name %s could not be found", r.Name)}
	}

	// NOTE(jdg): Volume can exist but not necessarily be attached, this just
	// gets the volume object and where it "would" be attached, it may or may
	// not currently be attached, but we don't care here
	path := filepath.Join(d.Conf.MountPoint, r.Name)

	return volume.Response{Volume: &volume.Volume{Name: r.Name, Mountpoint: path}}
}

func (d OVHPlugin) List(r volume.Request) volume.Response {
	log.Info("List volumes: ", r.Name)
	volumes, err := d.Client.ListVolumes()
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	var vols []*volume.Volume
	for _, v := range volumes {
		vols = append(vols, &volume.Volume{Name: v.Name, Mountpoint: filepath.Join(d.Conf.MountPoint, v.Name)})
	}
	return volume.Response{Volumes: vols}
}
