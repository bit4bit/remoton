package common

type RemotonClient struct {
	Capabilities *Capabilities
}

//Capabilities for this client
type Capabilities struct {
	XpraVersion string
}

func (c *RemotonClient) GetCapabilities(args struct{}, reply *Capabilities) error {
	*reply = *c.Capabilities
	return nil
}
