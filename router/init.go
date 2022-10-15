package router

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/NHAS/wag/config"
	"github.com/coreos/go-iptables/iptables"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func Setup(error chan<- error, iptables bool) (err error) {

	if !config.Values().Wireguard.External {
		err = setupWireguard()
		if err != nil {
			return err
		}
	} else {
		ctrl, err = wgctrl.New()
		if err != nil {
			return fmt.Errorf("cannot start wireguard control %v", err)
		}
	}

	if iptables {
		err = setupIptables()
		if err != nil {
			return err
		}
	}

	defer func() {
		if err != nil {
			TearDown()
		}
	}()

	err = setupXDP()
	if err != nil {
		return err
	}

	go func() {
		startup := true
		var endpoints = map[wgtypes.Key]*net.UDPAddr{}

		for {

			dev, err := ctrl.Device(config.Values().Wireguard.DevName)
			if err != nil {
				error <- fmt.Errorf("endpoint watcher: %s", err)
				return
			}

			for _, p := range dev.Peers {
				previousAddress := endpoints[p.PublicKey]

				if len(p.AllowedIPs) != 1 {
					log.Println("Warning, peer ", p.PublicKey.String(), " len(p.AllowedIPs) != 1, which is not supported")
					continue
				}

				if previousAddress.String() != p.Endpoint.String() {

					endpoints[p.PublicKey] = p.Endpoint

					//Dont try and remove rules, if we've just started
					if !startup {
						ip := p.AllowedIPs[0].IP.String()
						log.Println(ip, "endpoint changed", previousAddress.String(), "->", p.Endpoint.String())
						if err := Deauthenticate(ip); err != nil {
							log.Println(ip, "unable to remove forwards for device: ", err)
						}
					}
				}

			}

			startup = false

			time.Sleep(100 * time.Millisecond)
		}
	}()

	log.Println("Started firewall management: \n",
		"\t\t\tSetting filter FORWARD policy to DROP\n",
		"\t\t\tAllowed input on tunnel port\n",
		"\t\t\tSet MASQUERADE\n",
		"\t\t\tXDP eBPF program managing firewall\n",
		"\t\t\tSet public forwards")

	return nil
}

func TearDown() {
	_, tunnelPort, _ := net.SplitHostPort(config.Values().Webserver.Tunnel.ListenAddress)

	log.Println("Removing Firewall rules...")

	ipt, err := iptables.New()
	if err != nil {
		log.Println("Unable to clean up firewall rules: ", err)
		return
	}

	err = ipt.Delete("filter", "FORWARD", "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")
	if err != nil {
		log.Println("Unable to clean up firewall rules: ", err)
	}

	//Setup the links to the new chains
	err = ipt.Delete("filter", "FORWARD", "-i", config.Values().Wireguard.DevName, "-j", "ACCEPT")
	if err != nil {
		log.Println("Unable to clean up firewall rules: ", err)
	}

	err = ipt.Delete("nat", "POSTROUTING", "-s", config.Values().Wireguard.Range.String(), "-j", "MASQUERADE")
	if err != nil {
		log.Println("Unable to clean up firewall rules: ", err)
	}

	//Allow input to authorize web server on the tunnel
	err = ipt.Delete("filter", "INPUT", "-m", "tcp", "-p", "tcp", "-i", config.Values().Wireguard.DevName, "--dport", tunnelPort, "-j", "ACCEPT")
	if err != nil {
		log.Println("Unable to clean up firewall rules: ", err)
	}

	err = ipt.Delete("filter", "INPUT", "-p", "icmp", "--icmp-type", "8", "-i", config.Values().Wireguard.DevName, "-m", "state", "--state", "NEW,ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		log.Println("Unable to clean up firewall rules: ", err)
	}

	err = ipt.Delete("filter", "INPUT", "-i", config.Values().Wireguard.DevName, "-j", "DROP")
	if err != nil {
		log.Println("Unable to clean up firewall rules: ", err)
	}

	conn, err := netlink.Dial(unix.NETLINK_ROUTE, nil)
	if err != nil {
		log.Println("Unable to remove wireguard device, netlink connection failed: ", err.Error())
		return
	}
	defer conn.Close()

	err = delWg(conn, config.Values().Wireguard.DevName)
	if err != nil {
		log.Println("Unable to remove wireguard device, delete failed: ", err.Error())
		return
	}

}
