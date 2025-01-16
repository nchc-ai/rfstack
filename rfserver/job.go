package rfserver

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	v2 "github.com/gophercloud/gophercloud/acceptance/openstack/compute/v2"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/rfstack/model"
	"github.com/nchc-ai/rfstack/util"
)

// @Summary List all running course vm for a user
// @Description List all running course vm for a user
// @Tags Job
// @Accept  json
// @Produce  json
// @Param list_user body docs.OauthUser true "search user's job"
// @Success 200 {object} docs.JobListResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/job/list [post]
func (server *RFServer) ListJob(c *gin.Context) {

	services := []model.LabelValue{}

	provider, exist := c.Get("Provider")
	if exist == false {
		provider = ""
	}

	req := model.Job{}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		util.RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.User == "" {
		log.Errorf("Empty user name")
		util.RespondWithError(c, http.StatusBadRequest, "Empty user name")
		return
	}

	job := model.Job{
		OauthUser: model.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
	}

	resultJobs := []model.Job{}
	err = server.mysqldb.Where(&job).Find(&resultJobs).Error
	if err != nil {
		strErr := fmt.Sprintf("Query Job table for user {%s} fail: %s", req.User, err.Error())
		log.Errorf(strErr)
		util.RespondWithError(c, http.StatusInternalServerError, strErr)
		return
	}

	jobList := []model.JobInfo{}

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	for _, result := range resultJobs {

		courseInfo, err := findCourse(server.mysqldb, result)
		if err != nil {
			errStr := fmt.Sprintf("Query Course info for job {%s} fail: %s", result.ID, err.Error())
			log.Errorf(errStr)
			util.RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		vnc_url := GetVMVncUrl(computeclient, result.ID)
		vncservice := model.LabelValue{Label: "VNC", Value: vnc_url}
		servicelist := append(services, vncservice)

		sshport := server.getExposePort(result.PrivateIP, 4200)
		sshport_string := strconv.Itoa(sshport)
		if result.FloatingIP != "" {
			ssh_url := `https://` + result.FloatingIP + ":" + sshport_string
			//ssh_url := result.FloatingIP
			sshservice := model.LabelValue{Label: "SSH", Value: ssh_url}
			servicelist = append(servicelist, sshservice)
		}

		if courseInfo.Mount == 1 {

			sharepath := server.getSharePath(result.VolumeID)
			shareservice := model.LabelValue{Label: "NFS", Value: sharepath}
			servicelist = append(servicelist, shareservice)

		}

		imagename := server.getImageName(courseInfo.Image)
		flavorname := server.getFlavorName(courseInfo.Flavor)

		snapshot := false
		// user can take snapshot only when user is course owner
		c := model.Course{
			Model: model.Model{
				ID: req.CourseID,
			},
			OauthUser: model.OauthUser{
				User:     req.User,
				Provider: provider.(string),
			},
		}

		if result := server.mysqldb.Where(c).Find(&c); result.Error == nil {
			snapshot = true
		}

		// query port table
		port := model.Port{
			CourseID: courseInfo.ID,
		}
		portResult := []model.Port{}
		if err := server.mysqldb.Where(&port).Find(&portResult).Error; err != nil {
			log.Errorf("Query course {%s} ports fail: %s", result.ID, err.Error())
			return
		}

		jobInfo := model.JobInfo{
			Id:           result.ID, //serverGet.ID,
			CourseID:     courseInfo.ID,
			StartAt:      result.CreatedAt, //serverGet.Created,
			Status:       result.Status,    //serverGet.Status,
			Name:         courseInfo.Name,
			Introduction: *courseInfo.Introduction,
			Level:        courseInfo.Level,
			VMName:       courseInfo.Vmname,
			Image: model.LabelValue{
				Label: imagename,
				Value: courseInfo.Image,
			},
			Flavor: model.LabelValue{
				Label: flavorname,
				Value: courseInfo.Flavor,
			},
			SSHKey: model.LabelValue{
				Label: courseInfo.Sshkey,
				Value: courseInfo.Sshkey,
			},
			//				Image:        imagename,//courseInfo.Image,
			//				Flavor:       flavorname,//courseInfo.Flavor,
			//				SSHKey:       courseInfo.Sshkey,
			PrivateIP:   result.PrivateIP,  //privateip,
			FloatingIP:  result.FloatingIP, //floatingip,
			Volume:      result.VolumeID,   //volumes_attached,
			ExtraPorts:  &portResult,       //courseInfo.Extraports,
			Service:     servicelist,
			CanSnapshot: snapshot,
		}

		jobList = append(jobList, jobInfo)
	}
	c.JSON(http.StatusOK, model.JobListResponse{
		Error: false,
		Jobs:  jobList,
	})

}

const (
	JoBStatusCreated  = "Created"
	JoBStatusShutdown = "Shutdown"
	JoBStatusReady    = "Ready"
)

// @Summary Create a course vm in OpenStack
// @Description Create a course vm in OpenStack
// @Tags Job
// @Accept  json
// @Produce  json
// @Param launch_course body docs.LaunchCourseRequest true "course want to launch"
// @Success 200 {object} docs.LaunchCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/job/launch [post]
func (server *RFServer) LaunchJob(c *gin.Context) {

	var req model.LaunchCourseRequest
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		util.RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	user := req.User
	if user == "" {
		log.Errorf("user field in request cannot be empty")
		util.RespondWithError(c, http.StatusBadRequest, "user field in request cannot be empty")
		return
	}

	provider, exist := c.Get("Provider")
	if !exist {
		log.Warning("Provider is not found in request context, set empty")
		provider = ""
	}

	//Step 1: retrive required information
	course := getCourseObject(server.mysqldb, req.CourseId)
	if course == nil {
		log.Errorf("Query course id %s fail", req.CourseId)
		util.RespondWithError(c, http.StatusInternalServerError,
			"Query course id %s fail", req.CourseId)
		return
	}

	// Step 2: create vm
	net_uuid := server.config.GetString("stackvar.netid")

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	//password := stackclient.SetVMPW()
	//password := course.User
	password := req.User
	userdata := SetVMUserData(password)

	copts := servers.CreateOpts{
		Name:      course.ID, // date time
		ImageRef:  course.Image,
		FlavorRef: course.Flavor,
		Networks:  []servers.Network{{UUID: net_uuid}},
		UserData:  []byte(userdata),
	}

	//admin key
	keyname := course.Sshkey
	//add extra opts
	coptsext := keypairs.CreateOptsExt{
		copts,
		keyname,
	}

	vm, err := servers.Create(computeclient, coptsext).Extract()
	if err != nil {
		errStrt := fmt.Sprintf("create vm for course {id = %s} fail: %s", course.ID, err.Error())
		log.Errorf(errStrt)
		util.RespondWithError(c, http.StatusInternalServerError, errStrt)
		return
	}

	//wait VM active for serverGet
	if err := v2.WaitForComputeStatus(computeclient, vm, "ACTIVE"); err != nil {
		panic(err)
	}

	/*
	           var floatingip string
	           //associate floating ip
	           if course.Associate == "true" {

	   //	 availIP := server.ListFloatingIP()
	   //	 floatingip = server.CreateFloatingIP(availIP)
	   //	 AssociateFloatingIP(computeclient, vm.ID, floatingip)

	           } //if c

	   	// get private IP
	           serverGet, err := servers.Get(computeclient, vm.ID).Extract()
	           if err != nil {
	   		errStr := fmt.Sprintf("Faild to get vm {id = %s}: %s",vm.ID, err.Error())
	   		log.Errorf(errStr)
	   		util.RespondWithError(c, http.StatusInternalServerError, errStr)
	              	return
	           }

	   	fixedpool := server.config.GetString("stackvar.fixedpool")
	           var privateip string
	   	//addresses := serverGet.Addresses["public_private"].([]interface{})
	           addresses := serverGet.Addresses[fixedpool].([]interface{})
	           for _, ipresult := range addresses {
	              if ipresult.(map[string]interface{})["OS-EXT-IPS:type"] == "fixed" {
	                  privateip = fmt.Sprint(ipresult.(map[string]interface{})["addr"])
	              }
	           }
	*/

	// get private IP
	fixedpool := server.config.GetString("stackvar.fixedpool")
	var privateip string

	pages := 0
	err = servers.ListAddresses(computeclient, vm.ID).EachPage(func(page pagination.Page) (bool, error) {
		pages++
		actual, err := servers.ExtractAddresses(page)
		if err != nil {
			errStr := fmt.Sprintf("Faild to get vm-{id = %s}'s address: %s", vm.ID, err.Error())
			log.Errorf(errStr)
			util.RespondWithError(c, http.StatusInternalServerError, errStr)
			return false, err
		}
		privateip = actual[fixedpool][0].Address
		return true, nil
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to list vm's address: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	floatingip := course.CourseFIP //course.Associate

	//create portforwarding
	//server.createPortForwarding(floatingip,privateip)
	server.createPortForwarding(floatingip, privateip, course.Extraports)

	//manila share
	var shareid string

	// set nfs ro or rw
	if course.Mount == 1 {

		shareid = server.getShareID(course.ID)
		if shareid == "" {
			size, _ := strconv.Atoi(*course.Volume)
			server.createShare(c, size, course.ID)
			shareid = server.getShareID(course.ID)
		}
		var grantAccessReq shares.GrantAccessOpts
		grantAccessReq.AccessType = "ip"
		grantAccessReq.AccessTo = privateip

		if req.User == course.User {
			grantAccessReq.AccessLevel = "rw"
		} else {
			grantAccessReq.AccessLevel = "ro"
		}
		server.setGrantAccess(shareid, grantAccessReq)
	} // if c

	//add security group name to server
	if course.Extraports != "" {

		group_ID := server.getsecgroupID(course.ID)
		if group_ID == "" {
			server.createCourseSecgroup(course.ID, course.User, course.Extraports)
			group_ID := server.getsecgroupID(course.ID)
			server.createRule(group_ID, course.Extraports)
		}

		err = secgroups.AddServer(computeclient, vm.ID, course.ID).ExtractErr()
		if err != nil {
			errStrt := fmt.Sprintf("Failed to add security group {name = %s} to server: %s", course.ID, err.Error())
			log.Errorf(errStrt)
			util.RespondWithError(c, http.StatusBadRequest, errStrt)
			return
		}
	} // if c

	// Step 3: update Job Table
	err = updateTable(server.mysqldb, vm, course, user, provider.(string), privateip, floatingip, shareid)
	if err != nil {
		errStrt := fmt.Sprintf("Update Job Table for job {id = %s} fail: %s", vm.Name, err.Error())
		log.Errorf(errStrt)
		util.RespondWithError(c, http.StatusInternalServerError, errStrt)
		return
	}

	c.JSON(http.StatusOK, model.LaunchCourseResponse{
		Error: false,
		Job: model.JobStatus{
			JobId:  vm.ID,
			Ready:  false,
			Status: "Created",
		},
	})

	// create a go routine to check job is ready or not
	//        if course.Associate == 1 {
	sshport := server.getExposePort(privateip, 4200)
	//go server.checkJobStatus(vm,floatingip,sshport)
	go server.checkJobStatus_backoff(vm, floatingip, sshport)
	// }
}

// @Summary Delete a running job vm in OpenStack
// @Description Delete a running job vm in OpenStack
// @Tags Job
// @Accept  json
// @Produce  json
// @Param id path string true "job uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/job/delete/{id} [delete]
func (server *RFServer) DeleteJob(c *gin.Context) {

	jobId := c.Param("id")

	if jobId == "" {
		util.RespondWithError(c, http.StatusBadRequest,
			"Job Id is empty")
		return
	}

	if errStr, err := server.deleteJobInstance(jobId); err != nil {
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	util.RespondWithOk(c, "Job {%s} is deleted successfully", jobId)
}

// @Summary Delete classroom running jobs in OpenStack
// @Description Delete classroom running jobs in OpenStack
// @Tags Classroom
// @Accept  json
// @Produce  json
// @Param id path string true "job uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/classroom/delete/{id} [delete]
func (server *RFServer) DeleteClassroomJobs(c *gin.Context) {

	classroomId := c.Param("id")

	if classroomId == "" {
		util.RespondWithError(c, http.StatusBadRequest,
			"Classroom Id is empty")
		return
	}

	classroomCourse := model.ClassroomCourse{
		Classroom_ID: classroomId,
	}
	resultCourseId := []model.ClassroomCourse{}

	err := server.mysqldb.Where(&classroomCourse).Find(&resultCourseId).Error
	if err != nil {
		strErr := fmt.Sprintf("Query classroomCourse table for classroom {%s} fail: %s", classroomId, err.Error())
		log.Errorf(strErr)
		util.RespondWithError(c, http.StatusInternalServerError, strErr)
		return
	}

	for _, resultone := range resultCourseId {

		job := model.Job{CourseID: resultone.Course_ID}

		respall, err := server.queryJob(server.mysqldb, job)
		if err != nil {
			errStr := fmt.Sprintf("Query course {%s} related jobs fail: %s", resultone.Course_ID, err.Error())
			log.Errorf(errStr)
			util.RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		for _, respone := range respall {
			if errStr, err := server.deleteJobInstance(respone.ID); err != nil {
				util.RespondWithError(c, http.StatusInternalServerError, errStr)
				return
			}
		}
	}

	util.RespondWithOk(c, "Classroom {%s} jobs are deleted successfully", classroomId)
}

func (server *RFServer) deleteJobInstance(jobId string) (string, error) {

	job := model.Job{
		Model: model.Model{
			ID: jobId,
		},
	}

	if err := server.mysqldb.First(&job).Error; err != nil {
		return fmt.Sprintf("Failed to find job {%s} information : %s", jobId, err.Error()), err
	}

	computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
		return "", err
	}

	secgroups.ListByServer(computeclient, jobId).EachPage(func(page pagination.Page) (bool, error) {
		groupList, err := secgroups.ExtractSecurityGroups(page)
		if err != nil {
			log.Errorf("Failed to get Security Group information from OpenStack: %s", err.Error())
			return false, err
		}

		//check if "security group for course" exist or not
		for _, g := range groupList {
			if g.Name == job.CourseID {
				secgroups.RemoveServer(computeclient, jobId, job.CourseID)
			}
		}
		return true, nil
	})

	if job.VolumeID != "" {
		server.setRevokeAccess(job.VolumeID, job.PrivateIP)
	}

	//delete portforwarding
	server.deletePortForwarding(job.FloatingIP, job.PrivateIP)

	serverdelete := servers.Delete(computeclient, jobId)
	if serverdelete.Err != nil {
		return fmt.Sprintf("Failed to delete instance {%s} information : %s", jobId, serverdelete.Err), nil
	}

	if err := server.mysqldb.Unscoped().Delete(&job).Error; err != nil {
		return fmt.Sprintf("Failed to delete job {%s} information : %s", jobId, err.Error()), err
	}

	return "", nil
}

func (server *RFServer) queryJob(DB *gorm.DB, query interface{}, args ...interface{}) ([]model.Job, error) {
	// query job based on job condition
	results := []model.Job{}

	if err := DB.Where(query, args).Find(&results).Error; err != nil {
		log.Errorf("Query jobs table fail: %s", err.Error())
		return nil, err
	}

	resp := []model.Job{}

	for _, result := range results {
		resp = append(resp, model.Job{
			Model: model.Model{
				ID: result.ID,
			},
			CourseID: result.CourseID,
		})
	}
	return resp, nil
}
