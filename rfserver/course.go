package rfserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/rfstack/model"
	"github.com/nchc-ai/rfstack/util"
)

// @Summary List all course information
// @Description get all course information
// @Tags Course
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.ListCourseResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /v1/course/list [get]
func (server *RFServer) ListAllCourse(c *gin.Context) {
	course := model.Course{}
	results, err := server.queryCourse(server.mysqldb, course)
	if err != nil {
		errStr := fmt.Sprintf("query all course fail: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}
	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: results,
	})
}

func (server *RFServer) queryCourse(DB *gorm.DB, query interface{}, args ...interface{}) ([]model.Course, error) {
	// query course based on course condition
	results := []model.Course{}

	if err := DB.Where(query, args).Find(&results).Error; err != nil {
		log.Errorf("Query courses table fail: %s", err.Error())
		return nil, err
	}

	resp := []model.Course{}

	for _, result := range results {

		imagename := server.getImageName(result.Image)
		flavorname := server.getFlavorName(result.Flavor)

		// query port table
		port := model.Port{
			CourseID: result.ID,
		}
		portResult := []model.Port{}
		if err := DB.Where(&port).Find(&portResult).Error; err != nil {
			log.Errorf("Query course {%s} ports fail: %s", result.ID, err.Error())
			return nil, err
		}

		resp = append(resp, model.Course{
			Model: model.Model{
				ID:        result.ID,
				CreatedAt: result.CreatedAt,
			},
			Name:         result.Name,
			Introduction: result.Introduction,
			Level:        result.Level,
			Vmname:       result.Vmname,

			Image:  result.Image,
			Flavor: result.Flavor,
			Sshkey: result.Sshkey,
			Volume: result.Volume,

			AssociateBool: model.Sqlbool2Bool(result.Associate), //req.Associate,
			MountBool:     model.Sqlbool2Bool(result.Mount),     //req.Mount,//bool2String(req.MountJSON),

			Associate: result.Associate,
			CourseFIP: result.CourseFIP,
			Mount:     result.Mount,

			ExtraportsLV: &portResult,

			ImageLV: model.LabelValue{
				Label: imagename,
				Value: result.Image,
			},
			FlavorLV: model.LabelValue{
				Label: flavorname,
				Value: result.Flavor,
			},
			SshkeyLV: model.LabelValue{
				Label: result.Sshkey,
				Value: result.Sshkey,
			},
			VolumeLV: model.LabelValue{
				Label: *result.Volume,
				Value: *result.Volume,
			},
		})

	}
	return resp, nil
}

// @Summary List someone's all courses information
// @Description List someone's all courses information
// @Tags Course
// @Accept  json
// @Produce  json
// @Param list_user body docs.OauthUser true "search user course"
// @Success 200 {object} docs.ListCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/course/list [post]
func (server *RFServer) ListUserCourse(c *gin.Context) {
	provider, exist := c.Get("Provider")
	if exist == false {
		provider = ""
	}

	req := model.Course{}
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

	course := model.Course{
		OauthUser: model.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
	}

	resp, err := server.queryCourse(server.mysqldb, course)

	if err != nil {
		errStr := fmt.Sprintf("query user {%s} course fail: %s", req.User, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: resp,
	})

}

// @Summary Add new course information
// @Description Add new course information into database
// @Tags Course
// @Accept  json
// @Produce  json
// @Param course body docs.AddCourse true "course information"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/course/create [post]
func (server *RFServer) AddCourse(c *gin.Context) {

	var req model.Course
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		util.RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.User == "" {
		log.Errorf("user field in request cannot be empty")
		util.RespondWithError(c, http.StatusBadRequest, "user field in request cannot be empty")
		return
	}

	// add course information in DB
	courseID := uuid.New().String()

	provider, exist := c.Get("Provider")
	if !exist {
		log.Warning("Provider is not found in request context, set empty")
		provider = ""
	}

	//use transaction avoid partial update
	tx := server.mysqldb.Begin()

	newCourseId := model.CourseID{
		Model: model.Model{
			ID: courseID,
		},
	}
	err = tx.Create(&newCourseId).Error

	if err != nil {
		tx.Rollback()
		log.Errorf("Failed to register new course id: %s", err.Error())
		util.RespondWithError(c, http.StatusInternalServerError, "Failed to register new course id: %s", err.Error())
		return
	}

	availIP := "" //server.netListFloatingIP()
	floatingip := server.CreateFloatingIP(availIP)

	newCourse := model.Course{
		Model: model.Model{
			ID: courseID,
		},
		OauthUser: model.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
		Name:         req.Name,
		Introduction: req.Introduction,
		Level:        req.Level,
		Vmname:       courseID, //req.Vmname,
		//Extraports:   req.Extraports,

		Image:  req.ImageLV.Value,
		Flavor: req.FlavorLV.Value,
		Sshkey: req.SshkeyLV.Value,
		Volume: &req.VolumeLV.Value,

		Associate: model.Bool2Sqlbool(req.AssociateBool), //req.Associate,
		CourseFIP: floatingip,
		Mount:     model.Bool2Sqlbool(req.MountBool), //req.Mount,//bool2String(req.MountJSON),
	}

	// add course required port number in DB
	newCourse.Extraports = "4200"

	if newCourse.Associate == 1 {
		ports := *req.ExtraportsLV
		if len(ports) != 0 {
			for _, port := range ports {

				if port.Name == "" {
					tx.Rollback()
					log.Errorf("Empty Port name is not allowed")
					util.RespondWithError(c, http.StatusInternalServerError, "Empty Port name is not allowed")
					return
				}

				newPort := model.Port{
					CourseID: courseID,
					Name:     strings.TrimSpace(port.Name),
					Port:     port.Port,
				}

				newCourse.Extraports = fmt.Sprintf("%s#%s", newCourse.Extraports, strconv.Itoa(int(port.Port)))

				err = tx.Create(&newPort).Error
				if err != nil {
					tx.Rollback()
					log.Errorf("Failed to create course information: %s", err.Error())
					util.RespondWithError(c, http.StatusInternalServerError, "Failed to create course information: %s", err.Error())
					return
				}
			} //for
		} //if len
	} //if new

	group_ID := server.getsecgroupID(courseID)
	if group_ID == "" {
		server.createCourseSecgroup(courseID, req.User, newCourse.Extraports)
	} else {
		server.createRule(group_ID, newCourse.Extraports)
	}

	//security group, add port
	/*        if req.Extraports != "" {
	                  server.createCourseSecgroup(courseID, req.User, req.Extraports)
	          }
	*/

	//create share for course
	if newCourse.Mount == 1 {
		if *newCourse.Volume != "0" {
			size, _ := strconv.Atoi(*newCourse.Volume)
			server.createShare(c, size, courseID)
		}
	} else {
		*newCourse.Volume = "0"
	}

	err = tx.Create(&newCourse).Error
	if err != nil {
		tx.Rollback()
		log.Errorf("Failed to create course information: %s", err.Error())
		util.RespondWithError(c, http.StatusInternalServerError, "Failed to create course information: %s", err.Error())

		//delete OpenStack related resources
		server.deleteFloatingIP(floatingip)
		if req.Extraports != "" {
			server.deleteCourseSecgroup(courseID)
		}
		if newCourse.Mount == 1 {
			server.deleteCourseShares(courseID)
		}

		return
	}

	tx.Commit()
	util.RespondWithOk(c, "Course %s created successfully", req.Name)
}

// @Summary Delete course information
// @Description All associated job, vm, secgroup and vol-share in OpenStack are also deleted.
// @Tags Course
// @Accept  json
// @Produce  json
// @Param id path string true "course uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/course/delete/{id} [delete]
func (server *RFServer) DeleteCourse(c *gin.Context) {
	courseId := c.Param("id")

	if courseId == "" {
		util.RespondWithError(c, http.StatusBadRequest,
			"Course Id is not found")
		return
	}

	course := model.Course{
		Model: model.Model{
			ID: courseId,
		},
	}

	result := model.Course{}
	if err := server.mysqldb.Where(&course).First(&result).Error; err != nil {
		log.Errorf("Query courses {%s} fail: %s", courseId, err.Error())
		util.RespondWithError(c, http.StatusInternalServerError, "Query courses {%s} fail: %s", courseId, err.Error())
		return
	}

	jobs := []model.Job{}
	// Step 1: Find all associated Jobs
	if err := server.mysqldb.Model(&course).Related(&jobs).Error; err != nil {
		util.RespondWithError(c, http.StatusInternalServerError,
			"Failed to find jobs belong to course {%s} information : %s", courseId, err.Error())
		return
	}

	// Step 2-1: Delete instance in OpenStack &  job in DB
	for _, j := range jobs {
		if errStr, err := server.deleteJobInstance(j.ID); err != nil {
			util.RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}
	}

	// Step 2-2: Delete instance in OpenStack &  job in DB
	//delete security group
	server.deleteCourseSecgroup(courseId)
	//delete share
	server.deleteCourseShares(courseId)
	//release floating ip
	//server.deleteFloatingIP(result.Associate)
	server.deleteFloatingIP(result.CourseFIP)

	// Step 3: Delete course in DB.
	courseid := model.CourseID{
		Model: model.Model{
			ID: courseId,
		},
	}

	err := server.mysqldb.Unscoped().Delete(&courseid).Error
	if err != nil {
		util.RespondWithError(c, http.StatusInternalServerError,
			"Failed to delete course {%s} information : %s", courseId, err.Error())
		return
	}

	util.RespondWithOk(c, "Course %s is deleted successfully, associated jobs are also deleted", courseId)
}

// @Summary Get one courses information by course id
// @Description Get one courses information by course id
// @Tags Course
// @Accept  json
// @Produce  json
// @Param id path string true "course uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GetCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/course/get/{id} [get]
func (server *RFServer) GetCourse(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		log.Errorf("Empty course id")
		util.RespondWithError(c, http.StatusBadRequest, "Empty course id")
		return
	}

	course := model.Course{
		Model: model.Model{
			ID: id,
		},
	}

	result := model.Course{}
	if err := server.mysqldb.Where(&course).First(&result).Error; err != nil {
		log.Errorf("Query courses {%s} fail: %s", id, err.Error())
		util.RespondWithError(c, http.StatusInternalServerError, "Query courses {%s} fail: %s", id, err.Error())
		return
	}

	// get image's & flavor's name
	//	result.Image = server.getImageName(result.Image)
	//	result.Flavor = server.getFlavorName(result.Flavor)

	imagename := server.getImageName(result.Image)
	flavorname := server.getFlavorName(result.Flavor)

	// query port table
	port := model.Port{
		CourseID: result.ID,
	}
	portResult := []model.Port{}
	if err := server.mysqldb.Where(&port).Find(&portResult).Error; err != nil {
		log.Errorf("Query course {%s} ports fail: %s", result.ID, err.Error())
		return
	}

	result = model.Course{
		Model: model.Model{
			ID:        result.ID,
			CreatedAt: result.CreatedAt,
		},
		Name:         result.Name,
		Introduction: result.Introduction,
		Level:        result.Level,
		Vmname:       result.Vmname,

		Image:  result.Image,
		Flavor: result.Flavor,
		Sshkey: result.Sshkey,
		Volume: result.Volume,

		AssociateBool: model.Sqlbool2Bool(result.Associate),
		MountBool:     model.Sqlbool2Bool(result.Mount),

		Associate: result.Associate,
		Mount:     result.Mount,

		ExtraportsLV: &portResult,

		ImageLV: model.LabelValue{
			Label: imagename,
			Value: result.Image,
		},
		FlavorLV: model.LabelValue{
			Label: flavorname,
			Value: result.Flavor,
		},
		SshkeyLV: model.LabelValue{
			Label: result.Sshkey,
			Value: result.Sshkey,
		},
		VolumeLV: model.LabelValue{
			Label: *result.Volume,
			Value: *result.Volume,
		},
	}

	c.JSON(http.StatusOK, model.GetCourseResponse{
		Error:  false,
		Course: result,
	})
}

// @Summary Update course information
// @Description Update course information
// @Tags Course
// @Accept  json
// @Produce  json
// @Param course body docs.UpdateCourse true "new course information"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /v1/course/update/ [put]
func (server *RFServer) UpdateCourse(c *gin.Context) {
	var req model.Course

	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request: %s", err.Error())
		util.RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request: %s", err.Error())
		return
	}

	if req.ID == "" {
		log.Errorf("Course id is empty")
		util.RespondWithError(c, http.StatusBadRequest, "Course id is empty")
		return
	}

	findCourse := model.Course{
		Model: model.Model{
			ID: req.ID,
		},
	}

	if err = server.mysqldb.First(&findCourse).Error; err != nil {
		errStr := fmt.Sprintf("find course {%s} fail: %s", req.ID, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	tx := server.mysqldb.Begin()

	req.Associate = model.Bool2Sqlbool(req.AssociateBool)
	req.Mount = model.Bool2Sqlbool(req.MountBool)

	// update ports required by course
	// Step 1: delete ports used by course
	if err = tx.Where("course_id = ?", req.ID).Delete(model.Port{}).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("Failed to delete course {%s} port information in DB: %s", req.ID, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
	}
	// Step 2: create new ports
	if req.Associate == 1 {
		req.Extraports = "4200"
		ports := *req.ExtraportsLV
		if len(ports) != 0 {
			for _, port := range ports {

				if port.Name == "" {
					log.Errorf("Empty Port name is not allowed")
					util.RespondWithError(c, http.StatusInternalServerError, "Empty Port name is not allowed")
					return
				}

				newPort := model.Port{
					CourseID: req.ID,
					Name:     strings.TrimSpace(port.Name),
					Port:     port.Port,
				}

				req.Extraports = fmt.Sprintf("%s#%s", req.Extraports, strconv.Itoa(int(port.Port)))

				if err = tx.Create(&newPort).Error; err != nil {
					tx.Rollback()
					log.Errorf("Failed to create course-port information in DB: %s", err.Error())
					util.RespondWithError(c, http.StatusInternalServerError, "Failed to create course-port information in DB: %s", err.Error())
					return
				}
			}
		}

		if req.Extraports != findCourse.Extraports {
			if req.Extraports != "" && findCourse.Extraports == "" {
				//group exist or not: 1.create or 2.add new rule
				group_ID := server.getsecgroupID(req.ID)
				if group_ID == "" {
					server.createCourseSecgroup(req.ID, findCourse.User, req.Extraports)
				} else {
					server.createRule(group_ID, req.Extraports)
				}
			} else if req.Extraports == "" && findCourse.Extraports != "" {
				//remove rule
				group_ID := server.getsecgroupID(req.ID)
				server.deleteRule(group_ID, req.Extraports)
			} else {
				//reset rule
				server.resetCourseSecgroup(req.ID, findCourse.User, req.Extraports)
			}
		}
	}

	//Share client
	shareclient, err := openstack.NewSharedFileSystemV2(server.client, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Share Client: %s", err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}
	shareclient.Microversion = "2.7"

	if req.Mount != findCourse.Mount {
		if req.Mount == 1 {
			if req.VolumeLV.Value != "0" {
				//if share not exist,create new share
				var shareid string
				shareid = server.getShareID(req.ID)
				if shareid == "" {
					size, _ := strconv.Atoi(req.VolumeLV.Value)
					server.createShare(c, size, req.ID)
					*findCourse.Volume = strconv.Itoa(size)
				} else {
					s, err := shares.Get(shareclient, shareid).Extract()
					if err != nil {
						log.Errorf("Failed to get share size: %s", err)
						return
					}
					*findCourse.Volume = strconv.Itoa(s.Size)
				}
			}
		} else {
			req.VolumeLV.Value = "0"
		}
	}

	//process share extend or shrink
	if req.VolumeLV.Value != "0" && req.VolumeLV.Value != *findCourse.Volume {
		if *findCourse.Volume != "" {

			var shareid string
			shareid = server.getShareID(req.ID)

			req_size, _ := strconv.Atoi(req.VolumeLV.Value)
			course_size, _ := strconv.Atoi(*findCourse.Volume)

			if req_size > course_size {
				err := shares.Extend(shareclient, shareid, &shares.ExtendOpts{NewSize: req_size}).ExtractErr()
				if err != nil {
					log.Errorf("Failed to extend share: %s", err)
					return
				}

				err = waitForStatus(shareclient, shareid, "available", 600)
				if err != nil {
					log.Errorf("Failed to wait share available: %s", err.Error())
					return
				}
			}
			if 0 < req_size && req_size < course_size {
				err := shares.Shrink(shareclient, shareid, &shares.ShrinkOpts{NewSize: req_size}).ExtractErr()
				if err != nil {
					log.Errorf("Failed to shrink share: %s", err)
					return
				}

				err = waitForStatus(shareclient, shareid, "available", 600)
				if err != nil {
					log.Errorf("Failed to wait share available: %s", err.Error())
					return
				}
			}

		}
	}

	if err := tx.Model(&findCourse).
		UpdateColumn("associate", req.Associate).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update course {%s} associate field fail: %s", req.ID, err.Error())
		log.Error(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if err := tx.Model(&findCourse).
		UpdateColumn("mount", req.Mount).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update course {%s} mount field fail: %s", req.ID, err.Error())
		log.Error(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	// update Course DB
	if err := tx.Model(&findCourse).Updates(
		model.Course{
			Name:         req.Name,
			Introduction: req.Introduction,
			Level:        req.Level,
			//Vmname: req.Vmname,
			Extraports: req.Extraports,
			//			Associate: req.Associate,

			Image:  req.ImageLV.Value,
			Flavor: req.FlavorLV.Value,
			Sshkey: req.SshkeyLV.Value,
			Volume: &req.VolumeLV.Value,
			//			Mount:  req.Mount,
		}).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update course {%s} information fail: %s", req.ID, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	tx.Commit()
	util.RespondWithOk(c, "Course {%s} update successfully", req.ID)
}

// @Summary Search course name
// @Description Search course name
// @Tags Course
// @Accept  json
// @Produce  json
// @Param search body docs.Search true "search keyword"
// @Success 200 {object} docs.ListCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /v1/course/search [post]
func (server *RFServer) SearchCourse(c *gin.Context) {

	req := model.Search{}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		util.RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.Query == "" {
		log.Errorf("Empty query condition")
		util.RespondWithError(c, http.StatusBadRequest, "Empty query condition")
		return
	}
	results, err := server.queryCourse(server.mysqldb, "name LIKE ?", "%"+req.Query+"%")

	if err != nil {
		errStr := fmt.Sprintf("Search course on condition Name like % %s % fail: %s", req.Query, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: results,
	})
}

// @Summary List basic or advance courses information
// @Description List basic or advance courses information
// @Tags Course
// @Accept  json
// @Produce  json
// @Param level path string true "basic or advance"
// @Success 200 {object} docs.ListCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /v1/course/level/{level} [get]
func (server *RFServer) ListLevelCourse(c *gin.Context) {
	level := c.Param("level")

	if level == "" {
		log.Errorf("empty level string")
		util.RespondWithError(c, http.StatusBadRequest, "empty level string")
		return
	}

	course := model.Course{
		Level: level,
	}

	results, err := server.queryCourse(server.mysqldb, course)

	if err != nil {
		errStr := fmt.Sprintf("Query %s level course fail: %s", level, err.Error())
		log.Errorf(errStr)
		util.RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: results,
	})

}
