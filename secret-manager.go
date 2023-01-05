package cloudygcp

import (
	"context"
	"fmt"
	"hash/crc32"
	"strings"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/secrets"
)

const GcpSecretsManager = "gcp-secrets"
const SecretManagerCachedId = "gcp-secrets-cached"

func init() {
	secrets.SecretProviders.Register(GcpSecretsManager, &SecretManagerFactory{})
}

type SecretManagerFactory struct{}

type SecretManagerConfig struct {
	GcpCredentials
	Project  string
	VaultURL string `cloudyenv:"AZ_VAULT_URL"`
}

func (c *SecretManagerFactory) Create(cfg interface{}) (secrets.SecretProvider, error) {
	sec := cfg.(*SecretManagerConfig)
	if sec == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}
	return NewSecretManager(context.Background(), sec.VaultURL, sec.GcpCredentials)
}

func (c *SecretManagerFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
	cfg := &SecretManagerConfig{}
	cfg.VaultURL = env.Force("AZ_VAULT_URL")
	cfg.GcpCredentials = GetCredentialsFromEnv(env)
	return cfg, nil
}

type SecretManager struct {
	GcpCredentials
	Project string
	Client  secretmanagerpb.SecretManagerServiceClient
}

func NewSecretManager(ctx context.Context, project string, credentials GcpCredentials) (*SecretManager, error) {
	k := &SecretManager{
		GcpCredentials: credentials,
		Project:        project,
	}
	err := k.Configure(ctx)
	return k, err
}

func (k *SecretManager) Configure(ctx context.Context) error {

	cred, err := GetCredentials(k.GcpCredentials)
	if err != nil {
		return err
	}

	client := secretmanagerpb.NewSecretManagerServiceClient(creds)

	k.Client = client
	return nil
}

func (k *SecretManager) SaveSecretBinary(ctx context.Context, key string, secret []byte) error {
	name := k.toName(ctx, key, "")

	// So GCP is a bit stupid here. They require that you "create" a secret first and then
	// set a secret version. This means we first have to "Get" the secret to see if
	// it exists
	s, err := k.Client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: name,
	})
	if err != nil {
		if k.IsNotFound(err) {
			// This is ok
		} else {
			return err
		}
	}

	if s == nil {
		// Secret is not there so we need to create it
		s, err = k.Client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
			Parent:   k.Project,
			SecretId: key,
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					&secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		})

		if err != nil {
			return err
		}
	}

	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: s.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: secret,
		},
	}
	_, err = k.Client.AddSecretVersion(ctx, addSecretVersionReq)

	return err
}

func (k *SecretManager) GetSecretBinary(ctx context.Context, key string) ([]byte, error) {
	name := k.toName(ctx, key, "latest")

	resp, err := k.Client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})

	if err != nil {
		if k.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	//  Verify the data checksum.
	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(resp.Payload.Data, crc32c))
	if checksum != *resp.Payload.DataCrc32C {
		return cloudy.Error(ctx, "Data corruption detected in secret %v", key)
	}

	secretData := resp.Payload.Data
	return secretData, nil
}

func (k *SecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	secretData, err := k.GetSecretBinary(ctx, key)
	if err != nil {
		return "", err
	}

	return string(secretData), nil
}

func (k *SecretManager) SaveSecret(ctx context.Context, key string, data string) error {
	return k.SaveSecretBinary(ctx, key, []byte(data))
}

func (k *SecretManager) DeleteSecret(ctx context.Context, key string) error {
	_, err := k.Client.DeleteSecret(ctx, &secretmanagerpb.DeleteSecretRequest{
		Name: k.toName(ctx, key, ""),
	})

	return err
}

func (k *SecretManager) IsNotFound(err error) bool {
	str := err.Error()
	return strings.Contains(str, "SecretNotFound")
}

func sanitizeName(secretName string) string {
	// CHeck with google on valid secret names
	return secretName
}

// Format projects/my-project/secrets/my-secret/versions/5
func (k *SecretManager) toName(ctx context.Context, key string, version string) string {
	key = sanitizeName(key)
	if version == "" {
		version = "latest"
	}
	return fmt.Sprintf("projects/%v/secrets/%v/versions/%v", k.Project, key, version)
}
