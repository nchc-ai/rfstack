package rfserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/nchc-ai/rfstack/model"
	"github.com/nchc-ai/rfstack/util"
)

// @Summary List ssh key
// @Description List ssh key
// @Tags Key
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.KeysListResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/key/list [get]
func (server *RFServer) ListKey(c *gin.Context) {

	var l model.KeysListResponse

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	pager := keypairs.List(computeclient)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		keyList, err := keypairs.ExtractKeyPairs(page)
		if err != nil {
			log.Errorf("Failed to get key information from OpenStack: %s", err.Error())
			util.RespondWithError(c, http.StatusInternalServerError, "Failed to get key information from OpenStack: %s", err.Error())
			return false, err
		}
		for _, k := range keyList {
			// hard code, need to modify future
			if k.Name == "mykey" {
				l.Keys = append(l.Keys, model.LabelValue{k.Name, k.Name})
			}
		}
		return true, nil
	})

	if err != nil {
		errStr := fmt.Sprintf("Failed to get key page: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.KeysListResponse{
		Error: false,
		Keys:  l.Keys,
	})

}
