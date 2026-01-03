package provider

import (
	"github.com/haileyok/cocoon/oauth/client"
	"github.com/haileyok/cocoon/oauth/dpop"
)

type Provider struct {
	ClientManager *client.Manager
	DpopManager   *dpop.Manager

	hostname           string
	SupportedGrantTypes []string
}

type Args struct {
	Hostname          string
	ClientManagerArgs client.ManagerArgs
	DpopManagerArgs   dpop.ManagerArgs
	SupportedGrantTypes []string
}

func NewProvider(args Args) *Provider {
	grantTypes := args.SupportedGrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}

	return &Provider{
		ClientManager:      client.NewManager(args.ClientManagerArgs),
		DpopManager:        dpop.NewManager(args.DpopManagerArgs),
		hostname:           args.Hostname,
		SupportedGrantTypes: grantTypes,
	}
}

func (p *Provider) NextNonce() string {
	return p.DpopManager.NextNonce()
}
