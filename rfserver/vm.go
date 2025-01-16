package rfserver

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/nchc-ai/rfstack/model"
	"github.com/nchc-ai/rfstack/util"
)

// @Summary Create a course vm snapshot in OpenStack
// @Description Create a course vm snapshot in OpenStack
// @Tags VM
// @Accept  json
// @Produce  json
// @Param snapshot body docs.SnapshotRequest true "snapshot request"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/vm/snapshot [post]
func (server *RFServer) SnapshotVM(c *gin.Context) {

	req := model.SnapshotRequest{}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		util.RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.Name == "" {
		log.Errorf("Empty snapshot name")
		util.RespondWithError(c, http.StatusBadRequest, "Empty snapshot name")
		return
	}

	jobid := req.ID
	snapshot_name := req.Name
	snapshotOpts := servers.CreateImageOpts{
		Name: snapshot_name,
	}

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	snapshot_id, err := servers.CreateImage(computeclient, jobid, snapshotOpts).ExtractImageID()
	if err != nil {
		log.Errorf("Failed to Snapshot VM from OpenStack: %s", err.Error())
		util.RespondWithError(c, http.StatusInternalServerError, "Failed to Snapshot VM from OpenStack: %s", err.Error())
		return
	}

	imageclient, err := openstack.NewImageServiceV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Image Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	err = waitForIMGStatus(imageclient, snapshot_id, "active", 600)
	if err != nil {
		errStr := fmt.Sprintf("Failed to wait snapshot image id %s active: %s", snapshot_id, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	err = waitForVMStatus(computeclient, jobid, "ACTIVE", 600)
	if err != nil {
		errStr := fmt.Sprintf("Failed to wait VM job id %s active: %s", jobid, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	util.RespondWithOk(c, "Snapshot VM ID-{%s} as Image Name-{%s} successfully", jobid, snapshot_name)

}

func SetVMPW() string {

	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	length := 8
	buf := make([]rune, length)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	password := string(buf)
	return password

}

func SetVMUserData(password string) string {

	script := `packages:
 - shellinabox
runcmd:
 - yum install -t -y epel-release
 - yum install -t -y shellinabox
 - sed -i 's/LOGIN/SSH/g' /etc/sysconfig/shellinaboxd
 - [systemctl, enable, shellinaboxd]
 - [systemctl, restart, shellinaboxd]
 - [systemctl, enable, shellinabox]
 - [systemctl, restart, shellinabox]`

	//set user(ex: ubuntu, centos) password
	userdata := fmt.Sprintf("#cloud-config\nssh_pwauth: True\npassword: %s\n%s\n", password, script)
	return userdata

}

func waitForIMGStatus(client *gophercloud.ServiceClient, id string, status images.ImageStatus, secs int) error {
	return gophercloud.WaitFor(secs, func() (bool, error) {
		current, err := images.Get(client, id).Extract()
		if err != nil {
			return false, err
		}

		if current.Status == status {
			return true, nil
		}

		return false, nil
	})
}

func waitForVMStatus(client *gophercloud.ServiceClient, id string, status string, secs int) error {
	return gophercloud.WaitFor(secs, func() (bool, error) {
		current, err := servers.Get(client, id).Extract()
		if err != nil {
			return false, err
		}

		if current.Status == status {
			return true, nil
		}

		if current.Status == "ERROR" {
			return false, fmt.Errorf("Instance in ERROR state")
		}

		return false, nil
	})
}

// @Summary Shutdown a running VM in OpenStack
// @Description Shutdown a running VM in OpenStack
// @Tags VM
// @Accept  json
// @Produce  json
// @Param id path string true "job uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/vm/stop/{id} [get]
func (server *RFServer) StopVM(c *gin.Context) {

	jobId := c.Param("id")

	if jobId == "" {
		util.RespondWithError(c, http.StatusBadRequest,
			"Job Id is empty")
		return
	}

	jobObj := model.Job{
		Model: model.Model{
			ID: jobId,
		},
	}

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	err = startstop.Stop(computeclient, jobId).ExtractErr()
	if err != nil {
		util.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	} else {

		if err := server.mysqldb.Model(&jobObj).Update("status", JoBStatusShutdown).Error; err != nil {
			log.Errorf("update job {%s} status to %s fail: %s", jobId, JoBStatusShutdown, err.Error())
		}

	}

	util.RespondWithOk(c, "Stop Job {%s} successfully", jobId)

}

// @Summary Start a vm from shutdown status in OpenStack
// @Description Start a vm from shutdown status in OpenStack
// @Tags VM
// @Accept  json
// @Produce  json
// @Param id path string true "job uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/vm/start/{id} [get]
func (server *RFServer) StartVM(c *gin.Context) {

	jobId := c.Param("id")

	if jobId == "" {
		util.RespondWithError(c, http.StatusBadRequest, "Job Id is empty")
		return
	}

	jobObj := model.Job{
		Model: model.Model{
			ID: jobId,
		},
	}

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	err = startstop.Start(computeclient, jobId).ExtractErr()
	if err != nil {
		util.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	} else {

		if err := server.mysqldb.Model(&jobObj).Update("status", JoBStatusReady).Error; err != nil {
			log.Errorf("Update job {%s} status to %s fail: %s", jobId, JoBStatusReady, err.Error())
		}

	}

	util.RespondWithOk(c, "Start Job {%s} successfully", jobId)

}
