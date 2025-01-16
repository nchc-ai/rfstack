package rfserver

import (
        "github.com/gophercloud/gophercloud"
	log "github.com/golang/glog"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

func GetVMVncUrl(client *gophercloud.ServiceClient, server_id string) (string) {

        vnc_url, err := Vnc(client, server_id, "novnc").Extract()
        if err != nil {
                errStr := fmt.Sprintf("Unable to get VNC URL: %s", err.Error())
                log.Errorf(errStr)
                return "error"
        }

        return vnc_url
}

// VncType indicates what kind of VNC connection to request.
type VncType string
 // These constants determine what kind of VNC connection to request in Vnc()
const (
	NoVnc VncType = "novnc"
	XvpVnc = "xvpvnc"
)
 // Vnc returns the VNC URL for the given VncType.
func Vnc(client *gophercloud.ServiceClient, id string, t VncType) VncResult {
	var res VncResult
 	if id == "" {
		res.Err = fmt.Errorf("ID is required")
		return res
	} else if t == "" {
		res.Err = fmt.Errorf("vnc type is required")
		return res
	}
 	reqBody := struct {
		C map[string]string `json:"os-getVNCConsole"`
	}{
		map[string]string{"type": string(t)},
	}
 	_, res.Err = client.Post(actionURL(client, id), reqBody, &res.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200},
	})
	return res
}

func actionURL(client *gophercloud.ServiceClient, id string) string {
	return client.ServiceURL("servers", id, "action")
}

// ActionResult represents the result of server action operations, like reboot
type ActionResult struct {
	gophercloud.ErrResult
}

// VncResult represents the result of a server VNC request
type VncResult struct {
	ActionResult
}
 // VncConsole represents VNC call response.
type VncConsole struct {
	Type string `mapstructure:"type"`
	Url string `mapstructure:"url"`
}
 // Extract interprets VncResult as a VNC URL if possible.
func (r VncResult) Extract() (string, error) {
	if r.Err != nil {
		return "", r.Err
	}
 	var response struct {
		Console VncConsole `mapstructure:"console"`
	}
 	err := mapstructure.Decode(r.Body, &response)
	return response.Console.Url, err
}

