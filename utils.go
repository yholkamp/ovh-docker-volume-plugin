package main

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func waitForPathToExist(fileName string, numTries int) string {
	log.Infof("Waiting for path %s", fileName)
	for i := 0; i < numTries; i++ {
		matches, err := filepath.Glob(fileName)

		if err != nil {
			log.Errorf("Received error from os.Glob: %s", err)
			return ""
		}

		if matches != nil {
			return matches[0]
		}
		time.Sleep(time.Second)
	}
	return ""
}

func GetFSType(device string) string {
	log.Debugf("Begin utils.GetFSType: %s", device)
	fsType := ""
	out, err := exec.Command("blkid", device).CombinedOutput()
	if err != nil {
		return fsType
	}

	if strings.Contains(string(out), "TYPE=") {
		for _, v := range strings.Split(string(out), " ") {
			if strings.Contains(v, "TYPE=") {
				fsType = strings.Split(v, "=")[1]
				fsType = strings.Replace(fsType, "\"", "", -1)
			}
		}
	}
	return fsType
}

func FormatVolume(device, fsType string) error {
	log.Debugf("Begin utils.FormatVolume: %s, %s", device, fsType)
	cmd := "mkfs.ext4"
	if fsType == "xfs" {
		cmd = "mkfs.xfs"
	}
	log.Debug("Perform ", cmd, " on device: ", device)
	out, err := exec.Command(cmd, "-F", device).CombinedOutput()
	log.Debug("Result of mkfs cmd: ", string(out))
	return err
}

func Mount(device, mountpoint string) error {
	log.Debugf("Begin utils.Mount device: %s on: %s", device, mountpoint)
	out, err := exec.Command("mkdir", mountpoint).CombinedOutput()
	out, err = exec.Command("mount", device, mountpoint).CombinedOutput()
	log.Debug("Response from mount ", device, " at ", mountpoint, ": ", string(out))
	if err != nil {
		log.Error("Error in mount: ", err)
	}
	return err
}

func Umount(mountpoint string) error {
	log.Debugf("Begin utils.Umount: %s", mountpoint)
	out, err := exec.Command("umount", mountpoint).CombinedOutput()
	if err != nil {
		log.Warningf("Unmount call returned error: %s (%s)", err, out)
		if strings.Contains(string(out), "not mounted") {
			log.Debug("Ignore request for unmount on unmounted volume")
			err = errors.New("Volume is not mounted")
		}
	}
	return err
}

func getIpAddresses() (ips []string, error error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ips = append(ips, v.IP.String())
			case *net.IPAddr:
				ips = append(ips, v.IP.String())
			}
		}
	}

	return ips, nil
}
