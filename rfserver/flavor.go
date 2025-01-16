package rfserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/nchc-ai/rfstack/model"
	"github.com/nchc-ai/rfstack/util"
)

// @Summary List flavor
// @Description List flavor
// @Tags Flavor
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.FlavorsListResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/flavor/list [get]
func (server *RFServer) ListFlavor(c *gin.Context) {

	var l model.FlavorsListResponse

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	pager := flavors.ListDetail(computeclient, nil)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		flavorList, err := flavors.ExtractFlavors(page)
		if err != nil {
			log.Errorf("Failed to get flavor information from OpenStack: %s", err.Error())
			util.RespondWithError(c, http.StatusInternalServerError, "Failed to get flavor information from OpenStack: %s", err.Error())
			return false, err
		}
		for _, f := range flavorList {
			// hard code, need to modify future
			if f.Name != "manila-service-flavor" && f.Name != "kafka" {
				l.Flavors = append(l.Flavors, model.LabelValue{f.Name, f.ID})
			}
		}
		return true, nil
	})

	if err != nil {
		errStr := fmt.Sprintf("Failed to get flavor page: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.FlavorsListResponse{
		Error:   false,
		Flavors: l.Flavors,
	})

}

func (server *RFServer) getFlavorName(flavor_id string) string {

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		return "ERROR"
	}

	flavor, err := flavors.Get(computeclient, flavor_id).Extract()

	if err != nil {
		log.Errorf("Failed to get flavor information from OpenStack: %s", err.Error())
		return "ERROR"
	}

	return flavor.Name

}
