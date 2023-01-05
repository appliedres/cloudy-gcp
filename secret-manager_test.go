package cloudygcp

import (
	"log"
	"testing"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/secrets"
	"github.com/stretchr/testify/assert"
)

func TestKeyVault(t *testing.T) {

	ctx := cloudy.StartContext()

	sm, err := NewSecretManager(ctx, "arklouddev", GcpCredentials{})
	assert.Nil(t, err)
	if err != nil {
		log.Fatalln(err)
	}

	secrets.SecretsTest(t, ctx, sm)
}
