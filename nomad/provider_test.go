package nomad

import (
	"os"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// How to run the acceptance tests for this provider:
//
// - Obtain an official Nomad release from https://nomadproject.io
//   and extract the "nomad" binary
//
// - Run the following to start the Nomad agent in development mode:
//       nomad agent -dev -acl-enabled
//
// - Run the Terraform acceptance tests as usual:
//       make testacc TEST=./builtin/providers/nomad
//
// The tests expect to be run in a fresh, empty Nomad server.

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

var testProvider *schema.Provider
var testProviders map[string]terraform.ResourceProvider
var managementToken string

func init() {
	testProvider = Provider().(*schema.Provider)
	testProviders = map[string]terraform.ResourceProvider{
		"nomad": testProvider,
	}

	if v := os.Getenv(resource.TestEnvVar); v == "1" {
		setupAcl()
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("NOMAD_ADDR"); v == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}
	os.Setenv("NOMAD_TOKEN", managementToken)
}

func testAccPreCheckAnonymous(t *testing.T) {
	if v := os.Getenv("NOMAD_ADDR"); v == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}
	os.Unsetenv("NOMAD_TOKEN")
}

func setupAcl() {
	client, err := api.NewClient(&api.Config{})
	if err != nil {
		panic(err)
	}

	acl, _, err := client.ACLTokens().Bootstrap(nil)
	if err != nil {
		panic(err)
	}

	managementToken = acl.SecretID
}
