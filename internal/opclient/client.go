package opclient

import "github.com/1password/onepassword-sdk-go"

type Client interface {
	Secrets() onepassword.SecretsAPI
	Items() onepassword.ItemsAPI
	Vaults() onepassword.VaultsAPI
	Groups() onepassword.GroupsAPI
}
