package rfserver

import (
	"net"
	"strconv"
	"time"

	"github.com/cenkalti/backoff"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/rfstack/model"
)

func findCourse(DB *gorm.DB, job model.Job) (*model.Course, error) {

	course := model.Course{
		Model: model.Model{
			ID: job.CourseID,
		},
	}
	err := DB.Where(&course).Find(&course).Error
	if err != nil {
		log.Errorf("Query courses table fail: %s", err.Error())
		return nil, err
	}
	return &model.Course{
		Model: model.Model{
			ID:        course.ID,
			CreatedAt: course.CreatedAt,
		},
		Name:         course.Name,
		Introduction: course.Introduction,
		Image:        course.Image,
		Level:        course.Level,
		Flavor:       course.Flavor,
		Vmname:       course.Vmname,
		Associate:    course.Associate,
		Extraports:   course.Extraports,
		Sshkey:       course.Sshkey,
		Mount:        course.Mount,
		Volume:       course.Volume,
	}, nil
}

func updateTable(DB *gorm.DB, vm *servers.Server, course *model.Course, user string, provider string, privateip string, floatingip string, volumeid string) error {
	newJob := model.Job{
		Model: model.Model{
			ID: vm.ID,
		},
		OauthUser: model.OauthUser{
			User:     user,
			Provider: provider,
		},
		CourseID:   course.ID,
		Service:    "",
		Status:     JoBStatusCreated,
		PrivateIP:  privateip,
		FloatingIP: floatingip,
		VolumeID:   volumeid,
	}

	err := DB.Create(&newJob).Error
	if err != nil {
		log.Errorf("Create jod id %s into database fail: %s", vm.ID, err.Error())
		return nil
	}

	return nil
}

func getCourseObject(DB *gorm.DB, id string) *model.Course {
	course := model.Course{
		Model: model.Model{
			ID: id,
		},
	}

	err := DB.First(&course).Error
	if err != nil {
		log.Errorf("Query course id %s fail: %s", id, err.Error())
		return nil
	}

	return &course
}

// check VM ssh
func (server *RFServer) checkJobStatus(vm *servers.Server, floatingip string, sshport int) {

	jobObj := model.Job{
		Model: model.Model{
			ID: vm.ID,
		},
	}

	//ping public IP?
	//svcIP := fmt.Sprintf("%s:%d", floatingip, ssshport)
	sshport_string := strconv.Itoa(sshport)
	svcIP := floatingip + ":" + sshport_string

	for {
		time.Sleep(20 * time.Second)

		timeout := time.Duration(2 * time.Second)
		conn, err := net.DialTimeout("tcp", svcIP, timeout)
		if err != nil {
			log.Infof("%s [ssh] is not reachable", svcIP)
			continue
		}
		conn.Close()

		log.Infof("%s [ssh] is reachable", svcIP)
		break
	}
	//how to check privateip?

	if err := server.mysqldb.Model(&jobObj).Update("status", JoBStatusReady).Error; err != nil {
		log.Errorf("update job {%s} status to %s fail: %s", vm.ID, JoBStatusReady, err.Error())
	}

}

// use backoff to check VM ssh
func (server *RFServer) checkJobStatus_backoff(vm *servers.Server, floatingip string, sshport int) {

	jobObj := model.Job{
		Model: model.Model{
			ID: vm.ID,
		},
	}

	//ping public IP?
	//svcIP := fmt.Sprintf("%s:%d", floatingip, ssshport)

	sshport_string := strconv.Itoa(sshport)
	svcIP := floatingip + ":" + sshport_string

	operation := func() error {
		time.Sleep(15 * time.Second)
		timeout := time.Duration(2 * time.Second)
		conn, err := net.DialTimeout("tcp", svcIP, timeout)
		if err != nil {
			log.Infof("%s [ssh] is not reachable", svcIP)
			return err
		}
		conn.Close()
		log.Infof("%s [ssh] is reachable", svcIP)
		return nil
	}

	err := backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		log.Warningf("check Job-{%s} Accessible Retry timeout: %s", vm.ID, err.Error())
		return
	}

	if err := server.mysqldb.Model(&jobObj).Update("status", JoBStatusReady).Error; err != nil {
		log.Errorf("update job {%s} status to %s fail: %s", vm.ID, JoBStatusReady, err.Error())
	}

}

func bool2String(b bool) string {
	if b {
		return "true"
	} else {
		return "false"
	}
}

func string2Bool(s string) bool {
	b, err := strconv.ParseBool(s)

	if err != nil {
		log.Warningf("Parse string {%s} to bool fail: %s", s, err.Error())
		return false
	}
	return b
}
