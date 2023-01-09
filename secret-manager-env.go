package cloudygcp

import (
	"context"

	"github.com/appliedres/cloudy"
)

func init() {
	cloudy.EnvironmentProviders.Register(GoogleSecretsManager, &SecretManagerEnvironmentFactory{})
	cloudy.EnvironmentProviders.Register(GoogleSecretsManagerCached, &SecretManagerEnvironmentCachedFactory{})
}

type SecretManagerEnvironmentConfig struct {
	Project string
	Prefix  string
}

type SecretManagerEnvironmentFactory struct{}

func (c *SecretManagerEnvironmentFactory) Create(cfg interface{}) (cloudy.EnvironmentService, error) {
	sec := cfg.(*SecretManagerEnvironmentConfig)
	if sec == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}
	kve, err := NewSecretManagerEnvironmentService(context.Background(), sec.Project, sec.Prefix)
	return kve, err
}

func (c *SecretManagerEnvironmentFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
	cfg := &SecretManagerEnvironmentConfig{}
	cfg.Project = env.Force("GCP_PROJECT")
	cfg.Prefix = env.Get("prefix")

	return cfg, nil
}

type SecretManagerEnvironmentCachedFactory struct{}

func (c *SecretManagerEnvironmentCachedFactory) Create(cfg interface{}) (cloudy.EnvironmentService, error) {
	sec := cfg.(*SecretManagerEnvironmentConfig)
	if sec == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}
	kve, err := NewSecretManagerEnvironmentService(context.Background(), sec.Project, sec.Prefix)
	if err != nil {
		return nil, err
	}
	return cloudy.NewCachedEnvironment(kve), nil
}

func (c *SecretManagerEnvironmentCachedFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
	cfg := &SecretManagerEnvironmentConfig{}
	cfg.Project = env.Force("GCP_PROJECT")
	cfg.Prefix = env.Get("prefix")

	return cfg, nil
}

type SecretManagerEnvironment struct {
	Vault  *SecretManager
	Prefix string
}

func NewSecretManagerEnvironmentService(ctx context.Context, project string, prefix string) (*SecretManagerEnvironment, error) {
	sm, err := NewSecretManager(ctx, project, GcpCredentials{})
	env := &SecretManagerEnvironment{
		Vault:  sm,
		Prefix: prefix,
	}
	return env, err
}

func LoadEnvironment(ctx context.Context) (*cloudy.Environment, error) {
	return nil, nil
}

func (kve *SecretManagerEnvironment) Get(name string) (string, error) {
	ctx := cloudy.StartContext()

	val, err := kve.Vault.GetSecret(ctx, name)
	if err != nil {
		return "", err
	}
	if val == "" {
		return "", cloudy.ErrKeyNotFound
	}
	return val, nil
}

func (kve *SecretManagerEnvironment) SaveAll(ctx context.Context, items map[string]string) error {
	for k, v := range items {
		name := cloudy.NormalizeEnvName(k)
		err := kve.Vault.SaveSecret(ctx, name, v)
		if err != nil {
			return err
		}
	}
	return nil
}
