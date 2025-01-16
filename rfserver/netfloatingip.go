package rfserver

import (
        "github.com/gophercloud/gophercloud"
        "github.com/gophercloud/gophercloud/openstack"
        "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
        "github.com/gophercloud/gophercloud/pagination"
        "fmt"
        log "github.com/golang/glog"
)

func (server *RFServer) netListFloatingIP() (string) {

        networkclient, err := openstack.NewNetworkV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Netwwork Client: %s", err.Error())
                log.Errorf(errStr)
                return "ERROR"
        }

        availIP := ""

        floatingips.List(networkclient,floatingips.ListOpts{Status:"DOWN"}).EachPage(func(page pagination.Page) (bool, error) {
                resultlist, err := floatingips.ExtractFloatingIPs(page)
                if err != nil {
                        log.Errorf("Failed to get floatingip information from OpenStack: %s", err.Error())
                        return false, err
                }

                for _, result := range resultlist {

			check, err := server.queryCourse(server.mysqldb, "associate LIKE ?", "%"+result.FloatingIP+"%")
        		if err != nil {
                		errStr := fmt.Sprintf("Search course on condition FIP like % %s % fail: %s", result.FloatingIP, err.Error())
                		log.Errorf(errStr)
                		return false, err
        		}

			if len(check) == 0 {
                        	availIP = result.FloatingIP
				break
			}

                }
                return true, nil
        })
        return availIP

}

