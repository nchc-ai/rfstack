package model

import (
	"time"
)

type LabelValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ImagesListResponse struct {
	Error  bool         `json:"error"`
	Images []LabelValue `json:"images"`
}

type FlavorsListResponse struct {
	Error   bool         `json:"error"`
	Flavors []LabelValue `json:"flavors"`
}

type VolumesListResponse struct {
	Error   bool         `json:"error"`
	Volumes []LabelValue `json:"volumes"`
}

type SharesListResponse struct {
	Error  bool         `json:"error"`
	Shares []LabelValue `json:"shares"`
}

type KeysListResponse struct {
	Error bool         `json:"error"`
	Keys  []LabelValue `json:"keys"`
}

type JobInfo struct {
	Id           string     `json:"id"`
	CourseID     string     `json:"course_id"`
	StartAt      time.Time  `json:"startAt"`
	Status       string     `json:"status"`
	Name         string     `json:"name"`
	Introduction string     `json:"introduction"`
	Level        string     `json:"level"`
	VMName       string     `json:"vmname"`
	Image        LabelValue `json:"image"`
	Flavor       LabelValue `json:"flavor"`
	SSHKey       LabelValue `json:"sshkey"`
	ExtraPorts *[]Port  `json:"extraports"`
	//	Image        string       `json:"image"`
	//	Flavor       string       `json:"flavor"`
	//        PrivateIP    interface{}  `json:"privateip"`
	PrivateIP string `json:"privateip"`
	//        FloatingIP   interface{}  `json:"floatingip"`
	FloatingIP string `json:"floatingip"`
	//ExtraPorts string `json:"extraports"`
	//      ExtraPorts   []string     `json:"extraports"`
	//        SSHKey       string       `json:"sshkey"`
	//        Volume       interface{}  `json:"volume"`
	Volume      string       `json:"volume"`
	Service     []LabelValue `json:"service"`
	CanSnapshot bool         `json:"canSnapshot"`
}

type JobListResponse struct {
	Error bool      `json:"error"`
	Jobs  []JobInfo `json:"jobs"`
}

type ListCourseResponse struct {
	Error   bool     `json:"error"`
	Courses []Course `json:"courses"`
}

type GenericResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

type LaunchCourseRequest struct {
	User     string `json:"user"`
	CourseId string `json:"course_id"`
}

type LaunchCourseResponse struct {
	Error bool      `json:"error"`
	Job   JobStatus `json:"job"`
}

type JobStatus struct {
	JobId  string `json:"job_id"`
	Ready  bool   `json:"ready"`
	Status string `json:"status"`
}

type GetCourseResponse struct {
	Error  bool   `json:"error"`
	Course Course `json:"course"`
}

type Search struct {
	Query string `json:"query"`
}

type SnapshotRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
