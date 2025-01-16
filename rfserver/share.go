package rfserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/nchc-ai/rfstack/model"
	"github.com/nchc-ai/rfstack/util"
)

func (server *RFServer) ListShare(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	var l model.SharesListResponse

	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	listopts := shares.ListOpts{}
	pager := shares.ListDetail(shareclient, listopts)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		shareList, err := shares.ExtractShares(page)

		if err != nil {
			log.Errorf("Failed to get shares information from OpenStack: %s", err.Error())
			util.RespondWithError(c, http.StatusInternalServerError, "Failed to get shares information from OpenStack: %s", err.Error())
			return false, err
		}

		for _, i := range shareList {
			l.Shares = append(l.Shares, model.LabelValue{i.Name, i.ID})

		}
		return true, nil
	})

	if err != nil {
		errStr := fmt.Sprintf("Failed to get share page: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.SharesListResponse{
		Error:  false,
		Shares: l.Shares,
	})

}

func (server *RFServer) createShare(c *gin.Context, sharesize int, sharename string) {
	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	sharenetwork := server.config.GetString("stackvar.sharenetid")
	//shrenetwork:="e8814fd9-a0c0-40c6-aba3-f3488b1649bb"

	options := &shares.CreateOpts{Size: sharesize, Name: sharename, ShareNetworkID: sharenetwork, ShareProto: "NFS"}
	result, err := shares.Create(shareclient, options).Extract()

	if err != nil {
		log.Errorf("Failed to create shares from OpenStack: %s", err.Error())
		util.RespondWithError(c, http.StatusInternalServerError, "Failed to create shares from OpenStack: %s", err.Error())
		return
	}

	err = waitForStatus(shareclient, result.ID, "available", 600)
	if err != nil {
		log.Errorf("Failed to wait share volume available: %s", err.Error())
		return
	}

	// return result.ID
}

func (server *RFServer) deleteCourseShares(share_Name string) {

	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		return
	}

	pager := shares.ListDetail(shareclient, &shares.ListOpts{})
	pager.EachPage(func(page pagination.Page) (bool, error) {
		shareList, err := shares.ExtractShares(page)
		if err != nil {
			log.Errorf("Failed to get shares information from OpenStack: %s", err.Error())
			return false, err
		}
		for _, s := range shareList {
			if s.Name == share_Name {

				result := shares.Delete(shareclient, s.ID)
				if result.Err != nil {
					log.Errorf("Failed to delete share from OpenStack: %s", result.Err)
					return false, result.Err
				}

			}
		}
		return true, nil
	})
}

func (server *RFServer) getShareID(share_Name string) string {

	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		return "ERROR"
	}

	var shareid string
	pager := shares.ListDetail(shareclient, &shares.ListOpts{Name: share_Name})
	pager.EachPage(func(page pagination.Page) (bool, error) {
		shareList, err := shares.ExtractShares(page)
		if err != nil {
			log.Errorf("Failed to get shares information from OpenStack: %s", err.Error())
			return false, err
		}
		for _, s := range shareList {
			if s.Name == share_Name {
				shareid = s.ID
			}
		}
		return true, nil
	})

	return shareid

}

func (server *RFServer) getSharePath(share_ID string) string {

	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		return "ERROR"
	}

	shareclient.Microversion = "2.14"

	var result string

	s, err := shares.GetExportLocations(shareclient, share_ID).Extract()

	if err == nil {
		result = s[0].Path
	}

	return result
}

func (server *RFServer) setGrantAccess(share_ID string, grantAccessReq shares.GrantAccessOpts) {

	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		return
	}

	shareclient.Microversion = "2.7"

	_, err = shares.GrantAccess(shareclient, share_ID, grantAccessReq).Extract()

	if err != nil {
		log.Errorf("Failed to grant share access from OpenStack: %s", err)
		return
	}
}

func (server *RFServer) setRevokeAccess(share_ID string, server_IP string) {

	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		return
	}

	shareclient.Microversion = "2.7"

	accesslist, err := shares.ListAccessRights(shareclient, share_ID).Extract()

	for _, access := range accesslist {
		if access.AccessTo == server_IP {
			options := &shares.RevokeAccessOpts{AccessID: access.ID}
			err = shares.RevokeAccess(shareclient, share_ID, options).ExtractErr()

			if err != nil {
				log.Errorf("Failed to revoke share access from OpenStack: %s", err)
				return
			}
		}
	}

}

func waitForStatus(c *gophercloud.ServiceClient, id, status string, secs int) error {
	return gophercloud.WaitFor(secs, func() (bool, error) {
		current, err := shares.Get(c, id).Extract()
		if err != nil {
			return false, err
		}

		if current.Status == "error" {
			return true, fmt.Errorf("An error occurred")
		}

		if current.Status == status {
			return true, nil
		}

		return false, nil
	})
}
