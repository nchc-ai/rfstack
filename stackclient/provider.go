package stackclient

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

func NewStackClient(Tenant string, IdEndpoint string, User string, PassWD string) (*gophercloud.ProviderClient, error) {

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: IdEndpoint,
		Username:         User,
		Password:         PassWD,
		//        UserID: "376929f0e2b8413692be2eaf300c52b9",
		DomainID: "default",
		//      DomainName: "Default",
		TenantID: Tenant,
		//        TenantName: "admin",
                AllowReauth: true,
	}
	provider, err := openstack.AuthenticatedClient(opts)
	return provider, err
}
