package docs

type ListCourseResponse struct {
	Error   bool     `json:"error" example:"false" format:"bool"`
	Courses []Course `json:"courses"`
}

type GetCourseResponse struct {
	Error  bool   `json:"error" example:"false" format:"bool"`
	Course Course `json:"course"`
}

type Search struct {
	Query string `json:"query" example:"course keyword"`
}

type Course struct {
	Id           string  `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	CreatedAt    string  `json:"createAt" example:"2018-06-25T09:24:38Z"`
	Name         string  `json:"name" example:"shan的課" format:"string"`
	Introduction *string `json:"introduction" example:"課程說明" format:"string"`
	Level        string  `json:"level" example:"basic" format:"string"`
	Vmname       string  `json:"vmname" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c" format:"string"`
	Associate    bool    `json:"associate" example:true format:"string"`
	CourseFIP    string  `json:"coursefip" example:"140.110.5.86" format:"string"`

	Image  ImageLabelValue  `json:"image"`
	Flavor FlavorLabelValue `json:"flavor"`
	Sshkey SshkeyLabelValue `json:"sshkey"`
	Volume VolumeLabelValue `json:"volume"`
	Extraorts []PortLabelValue `json:"extraports"`

	Mount bool `json:"mount" example:"true" format:"bool"`
}

type AddCourse struct {
	OauthUser
	Name         string  `json:"name" example:"shan的課" format:"string"`
	Introduction *string `json:"introduction" example:"課程說明" format:"string"`
	Level        string  `json:"level" example:"basic" format:"string"`
	Vmname       string  `json:"vmname" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c" format:"string"`
	Associate    bool    `json:"associate" example:true format:"string"`

	Image  ImageLabelValue  `json:"image"`
	Flavor FlavorLabelValue `json:"flavor"`
	Sshkey SshkeyLabelValue `json:"sshkey"`
	Volume VolumeLabelValue `json:"volume"`
	Extraorts []PortLabelValue `json:"extraports"`

	Mount bool `json:"mount" example:"true" format:"bool"`
}

type UpdateCourse struct {
	Id           string  `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	Name         string  `json:"name" example:"shan的課" format:"string"`
	Introduction *string `json:"introduction" example:"課程說明" format:"string"`
	Level        string  `json:"level" example:"basic" format:"string"`
	Vmname       string  `json:"vmname" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c" format:"string"`
	Associate    bool    `json:"associate" example:true format:"string"`

	Image  ImageLabelValue  `json:"image"`
	Flavor FlavorLabelValue `json:"flavor"`
	Sshkey SshkeyLabelValue `json:"sshkey"`
	Volume VolumeLabelValue `json:"volume"`
	Extraorts []PortLabelValue `json:"extraports"`

	Mount bool `json:"mount" example:"true" format:"bool"`
}

type ImageLabelValue struct {
	Label string `json:"label" example:"Ubuntu 18.04"`
	Value string `json:"value" example:"88b51eda-a81f-4d5c-bd74-a77bba03c5d4"`
}

type FlavorLabelValue struct {
	Label string `json:"label" example:"m1.small"`
	Value string `json:"value" example:"2"`
}

type SshkeyLabelValue struct {
	Label string `json:"label" example:"mykey"`
	Value string `json:"value" example:"mykey"`
}

type VolumeLabelValue struct {
	Label string `json:"label" example:"10"`
	Value string `json:"value" example:"10"`
}

type PortLabelValue struct {
	Name string `json:"name" example:"https"`
	Port uint `json:"port" example:"443" format:"int64"`
}
