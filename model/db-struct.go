package model

import (
	"time"
)

func (Course) TableName() string {
	return "vmCourses"
}

func (Job) TableName() string {
	return "vmJobs"
}

type OauthUser struct {
	User     string `gorm:"size:50;not null" json:"user,omitempty"`
	Provider string `gorm:"size:30;not null" json:"-"`
}

type Model struct {
	ID        string     `gorm:"primary_key;size:36" json:"id"`
	CreatedAt time.Time  `json:"createAt"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`
}

type Sqlbool uint8

const TRUE = 1
const FALSE = 0

func Bool2Sqlbool(input bool) Sqlbool {
	if input == true {
		return TRUE
	} else {
		return FALSE
	}
}

func Sqlbool2Bool(input Sqlbool) bool {

	if input == TRUE {
		return true
	} else {
		return false
	}

}
type Course struct {
	Model
	OauthUser

	// db & JSON
	Name         string  `gorm:"not null" json:"name"`
	Introduction *string `gorm:"size:3000" json:"introduction,omitempty"`
	Level        string  `gorm:"not null;default:'basic';size:10" json:"level"`
	Vmname       string  `gorm:"not null" json:"vmname"`
	Associate    Sqlbool    `gorm:"not null;type:tinyint" json:"-"`
	Mount        Sqlbool  `gorm:"not null;type:tinyint" json:"-"`
	CourseFIP    string  `gorm:"not null" json:"coursefip"`

	// db only
	Image  string  `gorm:"not null" json:"-"`
	Flavor string  `gorm:"not null" json:"-"`
	Sshkey string  `gorm:"not null" json:"-"`
	Volume *string `gorm:"size:255" json:"-"`
	MountBool  bool  `gorm:"-" json:"mount"`
	AssociateBool    bool    `gorm:"-" json:"associate"`
        Extraports   string  `gorm:"size:255" json:"-"`

	//json only
	ImageLV  LabelValue `gorm:"-" json:"image"`
	FlavorLV LabelValue `gorm:"-" json:"flavor"`
	SshkeyLV LabelValue `gorm:"-" json:"sshkey"`
	VolumeLV LabelValue `gorm:"-" json:"volume"`
        ExtraportsLV *[]Port  `gorm:"-" json:"extraports,omitempty"`
}

type Job struct {
	Model
	OauthUser
	// foreign key
	CourseID string `gorm:"size:36"`

	//Deployment string `gorm:"not null"`
	Service string `gorm:"not null"`
	//ProxyUrl   string `gorm:"not null"`
	Status     string `gorm:"not null"`
	PrivateIP  string `gorm:"column:privateip" json:"privateip"`
	FloatingIP string `gorm:"column:floatingip" json:"floatingip"`
	VolumeID   string `gorm:"column:volumeid" json:"volumeid"`
}

type CourseID struct {
	Model
}

func (CourseID) TableName() string {
	return "courseid"
}

type ClassroomCourse struct {
	Classroom_ID string `gorm:"not null" json:"classroom_id"`
	Course_ID    string `gorm:"not null" json:"course_id"`
}

func (ClassroomCourse) TableName() string {
	return "classroomCourse"
}

func (Port) TableName() string {
        return "vmPorts"
}

type Port struct {
	Name string `gorm:"size:20;not null" json:"name"`
        Port uint `gorm:"primary_key" sql:"type:SMALLINT UNSIGNED NOT NULL" json:"port"`
	// foreign key
	CourseID string `gorm:"primary_key;size:36" json:"-"`
}
