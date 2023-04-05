// Package nordlynx provides nordlynx vpn technology.
package nordlynx

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"syscall"

	"github.com/NordSecurity/nordvpn-linux/internal"

	"golang.org/x/sys/unix"
)

const (
	// InterfaceName for various NordLynx implementations
	InterfaceName = "nordlynx"
	defaultPort   = 51820
	defaultMTU    = 1500
)

var (
	errNoKernelModule   = errors.New("interface of type wireguard not supported")
	errNoDefaultGateway = errors.New("default gateway not found")
)

// nordlynx client ipv6 address interface id (second portion of the address)
// nordlynx requires interface id to end with 2
// firewall rules depend on it
// agree with infra before changing it
func interfaceID() [8]byte {
	return [8]byte{0x0, 0x0, 0x0, 0x11, 0x0, 0x5, 0x0, 0x2}
}
func getDefaultGateway() (net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return net.Interface{}, err
		}
		for _, addr := range addrs {
			if net.ParseIP(addr.String()).Equal(net.IPv4zero) || net.ParseIP(addr.String()).Equal(net.IPv6zero) {
				return iface, nil
			}
		}

	}
	return net.Interface{}, errNoDefaultGateway
}

// SetMTU for an interface.
func SetMTU(iface net.Interface) error {
	var err error

	defaultGateway, err := getDefaultGateway()
	if err != nil {
		return nil
	}
	// wireguard-quick does this
	mtu := defaultGateway.MTU - 80

	fd, err := unix.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_IP)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	req, err := unix.NewIfreq(iface.Name)
	if err != nil {
		return err
	}
	req.SetUint32(uint32(mtu))

	return unix.IoctlIfreq(fd, unix.SIOCSIFMTU, req)
}

func upWGInterface(iface string) error {
	debug("ip", "link", "add", iface, "type", "wireguard")
	err := addDevice(iface, "wireguard")
	// there are only 2 cases when this can fail:
	// 1. Either kernel module is not found or the kernel was
	// recently updated, but the system is yet to be rebooted.
	// 2. wg command not found in path. (valid while we still rely on wg-tools)
	if err != nil {
		if internal.IsCommandAvailable("wg") {
			return err
		}
		return errNoKernelModule
	}
	return nil
}

func deleteInterface(iface net.Interface) error {
	debug("ip", "link", "delete", iface.Name)
	out, err := removeDevice(iface.Name)
	if err != nil {
		return errors.New(strings.Trim(string(out), "\n"))
	}
	return nil
}

// addDevice creates a new device with a given
// name and specified device type.
func addDevice(device, devType string) error {
	_, err := exec.Command("ip", "link", "add", device, "type", devType).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add device %w", err)
	}

	return nil
}

// removeDevice deletes the specified device.
func removeDevice(device string) ([]byte, error) {
	out, err := exec.Command("ip", "link", "delete", device).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("failed to remove device %w", err)
	}
	return out, nil
}

func debug(data ...string) {
	log.Println("[nordlynx]", strings.Join(data, " "))
}
