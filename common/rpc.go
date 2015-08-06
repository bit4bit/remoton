package common

import (
	"net"
	"runtime"

	"github.com/bit4bit/remoton/common/p2p/nat"
)

//Capabilities for this client
type Capabilities struct {
	//XpraVersion of running client xpra
	XpraVersion string
}

type RemotonClient struct {
	Capabilities *Capabilities
	NatIF        nat.Interface
}

func (c *RemotonClient) GetCapabilities(args struct{}, reply *Capabilities) error {
	*reply = *c.Capabilities
	return nil
}

func (c *RemotonClient) GetExternalIP(args struct{}, reply *net.IP) (err error) {
	*reply, err = c.NatIF.ExternalIP()
	return
}

func (c *RemotonClient) GetExternalPort(args struct{}, reply *int) error {
	//TODO this need to be dinamic
	*reply = 9932
	return nil
}

//GetOS of running system it's the same runtime.GOOS
func (c *RemotonClient) GetOS(args struct{}, reply *string) error {
	*reply = runtime.GOOS
	return nil
}

//GetArch of running system it's the same runtime.GOARCH
func (c *RemotonClient) GetArch(args struct{}, reply *string) error {
	*reply = runtime.GOARCH
	return nil
}
