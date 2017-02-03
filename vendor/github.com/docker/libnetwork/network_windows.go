// +build windows

package libnetwork

import (
	"runtime"
	"time"
	"fmt"

	"github.com/Microsoft/hcsshim"
	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/drivers/windows"
	"github.com/docker/libnetwork/ipamapi"
	"github.com/docker/libnetwork/ipams/windowsipam"
)

func executeInCompartment(compartmentID uint32, x func()) {
	runtime.LockOSThread()

	if err := hcsshim.SetCurrentThreadCompartmentId(compartmentID); err != nil {
		logrus.Error(err)
	}
	defer func() {
		hcsshim.SetCurrentThreadCompartmentId(0)
		runtime.UnlockOSThread()
	}()

	x()
}

func (n *network) startResolver() {
	n.resolverOnce.Do(func() {
		logrus.Debugf("Launching DNS server for network", n.Name())

		hnsallnetworks, err := hcsshim.HNSListNetworkRequest("GET", "", "")
		fmt.Println(windows.HNSID)
		hnsid := hnsallnetworks[0].Id
		fmt.Println(hnsid)

		hnsresponse := hnsallnetworks[0];
		for _, subnet := range hnsresponse.Subnets {
			if subnet.GatewayAddress != "" {
				for i := 0; i < 3; i++ {
					resolver := NewResolver(subnet.GatewayAddress, false, "", n)
					logrus.Debugf("Binding a resolver on network %s gateway %s", n.Name(), subnet.GatewayAddress)
					executeInCompartment(hnsresponse.DNSServerCompartment, resolver.SetupFunc(53))

					if err = resolver.Start(); err != nil {
						logrus.Errorf("Resolver Setup/Start failed for container %s, %q", n.Name(), err)
						time.Sleep(1 * time.Second)
					} else {
						logrus.Debugf("Resolver bound successfully for network %s", n.Name())
						n.resolver = append(n.resolver, resolver)
						break
					}
				}
			}
		}
	})
}

func defaultIpamForNetworkType(networkType string) string {
	if windows.IsBuiltinLocalDriver(networkType) {
		return windowsipam.DefaultIPAM
	}
	return ipamapi.DefaultIPAM
}
