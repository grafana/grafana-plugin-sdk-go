package azcredentials

type AzureCredentials interface {
	AzureAuthType() string
}
