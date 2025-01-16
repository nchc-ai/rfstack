package docs

import "time"

type LaunchCourseRequest struct {
	User     string `json:"user" example:"shan-test@teacher"`
	CourseId string `json:"course_id" example:"5ab02011-9ab7-40c3-b691-d335f93a12ee"`
}

type LaunchCourseResponse struct {
	Error bool      `json:"error" example:"false" format:"bool"`
	Job   JobStatus `json:"job"`
}

type JobStatus struct {
	JobId  string `json:"job_id" example:"5ab02011-9ab7-40c3-b691-d335f93a12ee"`
	Ready  bool   `json:"ready" example:"false" format:"bool"`
	Status string `json:"status" example:"Created"`
}

type JobListResponse struct {
	Error bool      `json:"error" example:"false" format:"bool"`
	Jobs  []JobInfo `json:"jobs"`
}

type JobInfo struct {
	Id           string       `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	CourseID     string       `json:"course_id" example:"b86b2893-b876-45c2-a3f6-5e099c15d638"`
	StartAt      time.Time    `json:"startAt" example:"2018-06-25T09:24:38Z"`
	Status       string       `json:"status" example:"Ready"`
	Name         string       `json:"name" example:"hadoop course"`
	Introduction string       `json:"introduction" example:"for big data"`
        Level        string       `json:"level" example:"basic"`
        VMName       string       `json:"vmname" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
        PrivateIP    string       `json:"privateip" example:"10.0.2.10"`
        FloatingIP   string       `json:"floatingip" example:"140.110.5.105"`

        Image  ImageLabelValue  `json:"image"`
        Flavor FlavorLabelValue `json:"flavor"`
        SSHKey SshkeyLabelValue `json:"sshkey"`
        Volume VolumeLabelValue `json:"volume"`
        ExtraPorts []PortLabelValue `json:"extraports"`

	Service      []SVCLabelValue `json:"service"`
	CanSnapshot  bool            `json:"canSnapshot" example:"true" format:"bool"`
}

type SVCLabelValue struct {
	Label string `json:"label" example:"VNC"`
	Value string `json:"value" example:"https://140.110.5.20:6080/vnc_auto.html?token=f093722f-1f8c-4649-a55c-004fe2525cae"`
}
