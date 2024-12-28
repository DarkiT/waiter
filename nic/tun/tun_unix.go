//go:build !windows

package tun

import (
	"os"

	nic "github.com/darkit/waiter"
	"github.com/darkit/wireguard/tun"
	"golang.org/x/sys/unix"
)

func CreateFD(tunFD int, cfg nic.Config) (*TUNIC, error) {
	err := unix.SetNonblock(tunFD, true)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(tunFD), "/dev/net/tun")
	device, err := tun.CreateTUNFromFile(file, 0)
	if err != nil {
		return nil, err
	}
	return &TUNIC{dev: device, mtu: cfg.MTU}, nil
}
