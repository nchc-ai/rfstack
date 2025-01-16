package rfserver

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"fmt"
        log "github.com/golang/glog"
)

func (server *RFServer) ListFloatingIP() (string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return "ERROR"
        }

	allPages, err := floatingips.List(computeclient).AllPages()
	if err != nil {
		panic(err)
	}

	allFloatingIPs, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		panic(err)
	}

	availIP := ""
	fippool := server.config.GetString("stackvar.fippool")

	for _, afip := range allFloatingIPs {
		if (afip.Pool == fippool) && (afip.FixedIP == "") && (afip.InstanceID == "") {
			availIP = afip.IP
		}
	}
	return availIP

}

func (server *RFServer) CreateFloatingIP(availIP string) (string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return "ERROR"
        }

	fippool := server.config.GetString("stackvar.fippool")
	floating_ip := ""

	if (availIP == "") {
		fcopts := floatingips.CreateOpts{
			Pool: fippool,
		}
		fip, err := floatingips.Create(computeclient, fcopts).Extract()
		if err != nil {
			panic(err)
		}
		floating_ip = fip.IP
		//fmt.Println("Floating IP:", fip.IP)
	} else {
		floating_ip = availIP
		//fmt.Println("Floating IP:", availIP)
	}
	return floating_ip

}

func (server *RFServer) deleteFloatingIP(fip string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return 
        }

	fip_id := server.getFipID(fip)
	err = floatingips.Delete(computeclient, fip_id).ExtractErr()

        if err != nil {
                errStr := fmt.Sprintf("Failed to delete(relese) OpenStack floating ip: %s", err.Error())
                log.Errorf(errStr)
                return 
        }

}

func AssociateFloatingIP(client *gophercloud.ServiceClient, server_id string, floating_ip string) {
	associateOpts := floatingips.AssociateOpts{
		FloatingIP: floating_ip,
	}

	err := floatingips.AssociateInstance(client, server_id, associateOpts).ExtractErr()
	if err != nil {
		panic(err)
	}

}

