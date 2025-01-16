package stackclient

import (
	"fmt"
	"time"

	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/nchc-ai/rfstack/model"
	"github.com/spf13/viper"
)

type ImageList struct {
	ID      uint `gorm:"primary_key"`
	Name    string
	ImageID string
}

type JobList struct {
	ID       uint `gorm:"primary_key"`
	Datetime time.Time
	Name     string
	ImageID  string
}

func NewDBClient(config *viper.Viper) (*gorm.DB, error) {

	dbArgs := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True",
		config.GetString("database.username"),
		config.GetString("database.password"),
		config.GetString("database.host"),
		config.GetInt("database.port"),
		config.GetString("database.database"),
	)

	db, err := gorm.Open("mysql", dbArgs)

	if err != nil {
		log.Fatalf("Create database client fail: %s", err.Error())
		return nil, err
	}

	// create tables
	vmCourse := &model.Course{}
	vmJob := &model.Job{}
	vmPort := &model.Port{}
	courseid := &model.CourseID{}
	classroomCourse := &model.ClassroomCourse{}

	// create mysql tables
	db.AutoMigrate(vmCourse, vmJob, vmPort, courseid, classroomCourse)

	// add foreign key
	db.Model(vmCourse).AddForeignKey("id", "courseid(id)", "CASCADE", "RESTRICT")
	db.Model(vmJob).AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT")
	db.Model(vmPort).AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT")
	db.Model(classroomCourse).AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT")

	return db, nil
}
