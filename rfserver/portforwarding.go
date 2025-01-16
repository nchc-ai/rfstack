package rfserver

import (
        "github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	log "github.com/golang/glog"
	"fmt"
	"strconv"
	"strings"
)

//get fixed ip (port_ip=internal_ip_address) - port id 
func (server *RFServer) getPortID(port_ip string) (string) {

        networkclient, err := openstack.NewNetworkV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Netwwork Client: %s", err.Error())
                log.Errorf(errStr)
                return "ERROR"
        }

	var portID string
	ports.List(networkclient, ports.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		portlist, err := ports.ExtractPorts(page)
		if err != nil {
			log.Errorf("Failed to extract subnets: %v", err.Error)
			return false, nil
		}

		for _, l := range portlist {
		    if len(l.FixedIPs) != 0 {
			if l.FixedIPs[0].IPAddress == port_ip {
				portID=l.ID
			}
		    }
		}
	return true, nil
	})

        return portID
}

//get expose port 
func (server *RFServer) getExposePort(internal_ip_address string, port int) (int) {

        stringSlice := strings.Split(internal_ip_address, ".")

	//var c int
	var d int

            for i, slice := range stringSlice {
 	      	//if i==2 {
			//x.x.c.x	
              	//	c,_ = strconv.Atoi(slice)
		//}
                if i==3 {
			//x.x.x.d
                        d,_ = strconv.Atoi(slice)
                }

	    }

	//exposeport := 30000+thousand*1000+hundred
	exposeport := 30000+d*10+port
	return exposeport
}

//get fip id
func (server *RFServer) getFipID(fip string) (fip_id string) {

        networkclient, err := openstack.NewNetworkV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Netwwork Client: %s", err.Error())
                log.Errorf(errStr)
                return "ERROR"
        }

	floatingips.List(networkclient, floatingips.ListOpts{FloatingIP: fip}).EachPage(func(page pagination.Page) (bool, error) {
		resultlist, err := floatingips.ExtractFloatingIPs(page)
                if err != nil {
                        errStr := fmt.Sprintf("Failed to extract information: %s", err.Error())
                        log.Errorf(errStr)
                        return false, nil
                }

	 	for _, result := range resultlist {
			fip_id = result.ID
		}

	return true, nil
	})
	return fip_id
}

func (server *RFServer) getPortForwardingID(fip string, internal_ip_address string) (pf_id string) {

	pflist := server.listPortForwarding(fip)

	if pflist == nil {
	        errStr := "No port farwarding for this FIP"
	        log.Errorf(errStr)
                return ""
	}else{
		for _,pf := range *pflist {
			if pf.Internal_IP_Address == internal_ip_address {
				pf_id = pf.ID
			}
		}
		return pf_id
	}
}

//list portforward
func (server *RFServer) listPortForwarding(fip string) (*Port_Forwardings) {

        networkclient, err := openstack.NewNetworkV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Netwwork Client: %s", err.Error())
                log.Errorf(errStr)
                return nil
        }

	fip_id := server.getFipID(fip)
	l, err := server.ListPF(networkclient, fip_id).Extract()

        if err != nil {
                errStr := fmt.Sprintf("Failed to list port forwardings: %s", err.Error())
                log.Errorf(errStr)
                return nil
        }

	return l
	//fmt.Println(l) 
	//need to modify

}

//create forward
func (server *RFServer) createPortForwarding(fip string, internal_ip_address string, extraports string) {

        networkclient, err := openstack.NewNetworkV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Netwwork Client: %s", err.Error())
                log.Errorf(errStr)
                return
        }

	fip_id := server.getFipID(fip)
	internal_port_id := server.getPortID(internal_ip_address)

	var inport int	
	stringSlice := strings.Split(extraports, "#")
	for _, slice := range stringSlice {

		inport,_ = strconv.Atoi(slice)
		external_port := server.getExposePort(internal_ip_address,inport)

		opts := CreateOpts{
			Protocol:"tcp",
			Internal_IP_Address:internal_ip_address,
			Internal_Port:inport,
			Internal_Port_ID:internal_port_id,
			External_Port:external_port,
		}

	        _,err = server.CreatePF(networkclient, fip_id, opts).ExtractCreate()

        	if err != nil {
                	errStr := fmt.Sprintf("Failed to create port forwardings: %s", err.Error())
                	log.Errorf(errStr)
                	return
        	}
	}
	//fmt.Println(output)
}

//delete forward
func (server *RFServer) deletePortForwarding(fip string, internal_ip_address string) {

        networkclient, err := openstack.NewNetworkV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Netwwork Client: %s", err.Error())
                log.Errorf(errStr)
                return
        }

	fip_id := server.getFipID(fip)
	pf_id := server.getPortForwardingID(fip,internal_ip_address)

	if pf_id != ""{
		output := server.DeletePF(networkclient, fip_id, pf_id)
		if output.Err != nil {
                	errStr := fmt.Sprintf("Failed to delete port forwardings: %s", output.Err)
                	log.Errorf(errStr)
                	return
        	}
	}else{
                errStr := "NO port farwarding need to be deleted"
                log.Errorf(errStr)
	}

}

type GetResult struct {
	commonResult
}

type commonResult struct {
	gophercloud.Result
}

type CreateOptsBuilder interface {
	ToPFCreateMap() (map[string]interface{}, error)
}

func (opts CreateOpts) ToPFCreateMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "port_forwarding")
}

func (server *RFServer) ListPF (client *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = client.Get(portforwardingURL(client, id), &r.Body, nil)
	return
}

func (server *RFServer) CreatePF (client *gophercloud.ServiceClient, id string, opts CreateOptsBuilder) (r GetResult) {

	pfmap, err := opts.ToPFCreateMap()
	if err != nil {
		r.Err = err
		return
	}
        _, r.Err = client.Post(portforwardingURL(client, id), pfmap, &r.Body, nil)
        return
}

func (server *RFServer) DeletePF(client *gophercloud.ServiceClient, id string, pf_id string) (r GetResult) {
	_, r.Err = client.Delete(portforwardingdeleteURL(client, id, pf_id), nil)
	return
}

func (r commonResult)Extract() (*Port_Forwardings, error) {
        var s struct {
                Port_Forwardings *Port_Forwardings `json:"port_forwardings"`
        }
        err := r.ExtractInto(&s)
        return s.Port_Forwardings, err
}

func (r commonResult)ExtractCreate() (*ListOpts, error) {
        var s struct {
                Port_Forwarding *ListOpts `json:"port_forwarding"`
        }
        err := r.ExtractInto(&s)
        return s.Port_Forwarding, err
}


type CreateOpts struct {
	Protocol	string `json:"protocol"`
	Internal_IP_Address	string `json:"internal_ip_address"`
	Internal_Port	int    `json:"internal_port"`
	Internal_Port_ID	string `json:"internal_port_id"`
	External_Port	int    `json:"external_port"`
} 

//also for Createoutput struct
type ListOpts struct {

      ID string `json:"id"`
      Protocol string `json:"protocol"`
      Internal_IP_Address string `json:"internal_ip_address"`
      Internal_Port int `json:"internal_port"`
      Internal_Port_ID string `json:"internal_port_id"`
      External_Port int `json:"external_port"`

}

type Port_Forwardings []struct {

      ID string `json:"id"`
      Protocol string `json:"protocol"`
      Internal_IP_Address string `json:"internal_ip_address"`
      Internal_Port int `json:"internal_port"`
      Internal_Port_ID string `json:"internal_port_id"`
      External_Port int `json:"external_port"`

}

const resourcePath = "floatingips"

func portforwardingURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(resourcePath, id,"port_forwardings")
}

func portforwardingdeleteURL(c *gophercloud.ServiceClient, id string, pf_id string) string {
        return c.ServiceURL(resourcePath, id,"port_forwardings",pf_id)
}

