package cloudygcp

import (
	"testing"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/secrets"
	"github.com/appliedres/cloudy/testutil"
	"github.com/stretchr/testify/assert"
)

func TestKeyVault(t *testing.T) {

	ctx := cloudy.StartContext()

	sm, err := NewSecretManager(ctx, "arklouddev")
	assert.Nil(t, err)

	secrets.SecretsTest(t, ctx, kv)
}
