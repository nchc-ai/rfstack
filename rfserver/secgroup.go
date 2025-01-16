package rfserver

import(
 "github.com/gophercloud/gophercloud"
 "github.com/gophercloud/gophercloud/openstack"
 "fmt"
 "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
 "github.com/gophercloud/gophercloud/pagination"
 log "github.com/golang/glog"
 "strconv"
 "strings"
)

func (server *RFServer) deleteCourseSecgroup(group_Name string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
		errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
		log.Errorf(errStr)
                return
        }

	group_ID:=server.getsecgroupID(group_Name)

        if group_ID != "" {
          err = secgroups.Delete(computeclient, group_ID).ExtractErr()
          if err != nil {
                log.Errorf("Failed to delete Security Group from OpenStack: %s", err.Error())
                return
          }
        }

}

func (server *RFServer) createCourseSecgroup(course_ID string, course_User string, extraports string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return
        }

           //security group, add port
            groupcreate,_ := secgroups.Create(computeclient, secgroups.CreateOpts{
                Name:        course_ID,
                Description: course_User,
            }).Extract()

	    server.createRule(groupcreate.ID,extraports)

}

func (server *RFServer) resetCourseSecgroup(course_ID string, course_User string, extraports string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return
        }

	  group_ID:=server.getsecgroupID(course_ID)

          if group_ID == "" {
            groupcreate,_ := secgroups.Create(computeclient, secgroups.CreateOpts{
                Name:        course_ID,
                Description: course_User,
            }).Extract()
            group_ID = groupcreate.ID
          }else{
            server.deleteRule(group_ID,extraports)
          }

	server.createRule(group_ID,extraports)
}

func (server *RFServer) createRule(group_ID string, extraports string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return
        }

            //parse extraports, add new ports
            stringSlice := strings.Split(extraports, "#")

            for _, slice := range stringSlice {
              port,_ := strconv.Atoi(slice)
              secgroups.CreateRule(computeclient, secgroups.CreateRuleOpts{
                ParentGroupID: group_ID,
                FromPort:      port,
                ToPort:        port,
                IPProtocol:    "TCP",
                CIDR:          "0.0.0.0/0",
              }).Extract()
            }

}

func (server *RFServer) deleteRule(group_ID string, extraports string) {

        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return
        }
            groupget,err := secgroups.Get(computeclient, group_ID).Extract()
 	    if err != nil {
                log.Errorf("Failed to get Security Group information from OpenStack: %s", err.Error())
                return 
           }

            //if group_ID exist, delete old rules
            for _, r := range groupget.Rules {
              secgroups.DeleteRule(computeclient,r.ID)
            }

}

func (server *RFServer) getsecgroupID(group_Name string) (string){
 
        computeclient, err := openstack.NewComputeV2(server.client, gophercloud.EndpointOpts{
                Region: "RegionOne",
        })
        if err != nil {
                errStr := fmt.Sprintf("Failed to create OpenStack Compute Client: %s", err.Error())
                log.Errorf(errStr)
                return "ERROR"
        }

        var group_ID string

        secgroups.List(computeclient).EachPage(func(page pagination.Page) (bool, error) {
           groupList, err := secgroups.ExtractSecurityGroups(page)
           if err != nil {
                log.Errorf("Failed to get Security Group information from OpenStack: %s", err.Error())
                return false, err
           }
           //check if "security group for course" exist or not
           for _, g := range groupList {
             if g.Name == group_Name {
             group_ID = g.ID
             }
           }
           return true,nil
        })

        return group_ID
}
