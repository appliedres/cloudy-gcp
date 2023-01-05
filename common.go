package cloudygcp

type GcpCredentials struct {
}

func GetCredentialsFromEnv() GcpCredentials {
	return GcpCredentials{}
}
