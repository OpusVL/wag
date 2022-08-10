package router

import (
	"errors"
	"fmt"
	"net"
	"wag/config"
	"wag/utils"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var (
	ctrl *wgctrl.Client
)

func ServerDetails() (key wgtypes.Key, port int, err error) {
	ctr, err := wgctrl.New()
	if err != nil {
		return key, port, fmt.Errorf("cannot start wireguard control %v", err)
	}
	defer ctr.Close()

	dev, err := ctr.Device(config.Values().WgDevName)
	if err != nil {
		return key, port, fmt.Errorf("unable to start wireguard-ctrl on device with name %s: %v", config.Values().WgDevName, err)
	}

	return dev.PublicKey, dev.ListenPort, nil
}

func RemovePeer(public wgtypes.Key) error {

	var c wgtypes.Config
	c.Peers = append(c.Peers, wgtypes.PeerConfig{
		PublicKey: public,
		Remove:    true,
	})

	return ctrl.ConfigureDevice(config.Values().WgDevName, c)
}

// AddPeer the device to wireguard
func AddPeer(public wgtypes.Key) (string, error) {

	dev, err := ctrl.Device(config.Values().WgDevName)
	if err != nil {
		return "", err
	}

	//Poor selection algorithm

	//If we dont have any peers take the server tun address and increment that
	newAddress := config.Values().VPNServerAddress.String()
	if len(dev.Peers) > 0 {
		addresses := []string{}
		for _, peer := range dev.Peers {
			addresses = append(addresses, utils.GetIP(peer.AllowedIPs[0].IP.String()))
		}

		newAddress = addresses[len(addresses)-1]
	}

	newAddress, err = incrementIP(newAddress, config.Values().VPNRange.String())
	if err != nil {
		return "", err
	}

	_, network, err := net.ParseCIDR(newAddress + "/32")
	if err != nil {
		return "", err
	}

	var c wgtypes.Config
	c.Peers = append(c.Peers, wgtypes.PeerConfig{
		PublicKey:         public,
		ReplaceAllowedIPs: true,
		AllowedIPs:        []net.IPNet{*network},
	})

	return network.IP.String(), ctrl.ConfigureDevice(config.Values().WgDevName, c)
}

func GetPeerRealIp(address string) (string, error) {
	dev, err := ctrl.Device(config.Values().WgDevName)
	if err != nil {
		return "", err
	}

	for _, peer := range dev.Peers {
		if len(peer.AllowedIPs) == 1 && peer.AllowedIPs[0].IP.String() == address {
			return peer.Endpoint.String(), nil
		}
	}

	return "", errors.New("not found")
}

func incrementIP(origIP, cidr string) (string, error) {
	ip := net.ParseIP(origIP)
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return origIP, err
	}
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	if !ipNet.Contains(ip) {
		return origIP, errors.New("overflowed CIDR while incrementing IP")
	}
	return ip.String(), nil
}