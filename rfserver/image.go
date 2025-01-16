package rfserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/nchc-ai/rfstack/model"
	"github.com/nchc-ai/rfstack/util"
)

// @Summary List image
// @Description List image
// @Tags Image
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.ImagesListResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/image/list [get]
func (server *RFServer) ListImage(c *gin.Context) {
	var l model.ImagesListResponse

	imageclient, err := openstack.NewImageServiceV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Image Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	listopts := images.ListOpts{Tags: []string{"aitrain"}}
	pager := images.List(imageclient, listopts)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		imageList, err := images.ExtractImages(page)

		if err != nil {
			log.Errorf("Failed to get images information from OpenStack: %s", err.Error())
			util.RespondWithError(c, http.StatusInternalServerError, "Failed to get images information from OpenStack: %s", err.Error())
			return false, err
		}

		for _, i := range imageList {
			// hard code, need to modify future
			//	    if i.Name != "manila-service-image" && i.Name != "amphora-x64-haproxy" {
			l.Images = append(l.Images, model.LabelValue{i.Name, i.ID})
			//	    }
		}

		return true, nil
	})

	if err != nil {
		errStr := fmt.Sprintf("Failed to get image page: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.ImagesListResponse{
		Error:  false,
		Images: l.Images,
	})

}

func (server *RFServer) getImageName(image_id string) string {

	imageclient, err := openstack.NewImageServiceV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Image Client: %s", err.Error())
		log.Errorf(errStr)
		return "ERROR"
	}

	image, err := images.Get(imageclient, image_id).Extract()

	if err != nil {
		log.Errorf("Failed to get images information from OpenStack: %s", err.Error())
		return "ERROR"
	}

	return image.Name

}
