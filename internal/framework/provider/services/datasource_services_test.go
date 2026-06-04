// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package services_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
	"github.com/shoenig/test/must"
)

func TestAccDataSourceNomadServices_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() { registerTestService(t, "services-list-webapp", "default") },
				Config:    testAccDataSourceNomadServicesConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.nomad_services.test", "services.#"),
					testCheckServicesContain(t, "services-list-webapp", "default"),
				),
			},
		},
	})
}

func TestAccDataSourceNomadServices_namespace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() { registerTestService(t, "services-ns-webapp", "default") },
				Config:    testAccDataSourceNomadServicesNamespaceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.nomad_services.test", "services.#"),
					testCheckServicesContain(t, "services-ns-webapp", "default"),
				),
			},
		},
	})
}

func testAccDataSourceNomadServicesConfig() string {
	return `
data "nomad_services" "test" {}
`
}

func testAccDataSourceNomadServicesNamespaceConfig() string {
	return `
data "nomad_services" "test" {
  namespace = "default"
}
`
}

// testCheckServicesContain verifies that the services list in state contains a
// service with the given name and namespace, and validates its tags.
func testCheckServicesContain(t *testing.T, serviceName, namespace string) resource.TestCheckFunc {
	t.Helper()
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["data.nomad_services.test"]
		must.True(t, ok, must.Sprintf("data.nomad_services.test not found in state"))

		attrs := rs.Primary.Attributes
		numServices := attrs["services.#"]
		must.NotEq(t, "0", numServices, must.Sprintf("expected at least one service"))

		for i := 0; ; i++ {
			name, exists := attrs[fmt.Sprintf("services.%d.name", i)]
			if !exists {
				break
			}
			ns := attrs[fmt.Sprintf("services.%d.namespace", i)]
			if name == serviceName && ns == namespace {
				// Verify tags are present.
				tagsCount := attrs[fmt.Sprintf("services.%d.tags.#", i)]
				must.Eq(t, "2", tagsCount,
					must.Sprintf("expected 2 tags for service %s", serviceName))
				// Collect tags and verify both expected tags exist.
				tags := make(map[string]struct{})
				for j := 0; j < 2; j++ {
					tags[attrs[fmt.Sprintf("services.%d.tags.%d", i, j)]] = struct{}{}
				}
				_, hasHTTP := tags["http"]
				_, hasTest := tags["test"]
				must.True(t, hasHTTP, must.Sprintf("expected 'http' tag"))
				must.True(t, hasTest, must.Sprintf("expected 'test' tag"))
				return nil
			}
		}
		t.Fatalf("service %q in namespace %q not found in services list", serviceName, namespace)
		return nil
	}
}

func registerTestService(t *testing.T, serviceName, namespace string) {
	t.Helper()

	providerData := testutil.SDKV2ProviderMeta(t)()
	providerConfig, ok := providerData.(nomad.ProviderConfig)
	must.True(t, ok, must.Sprintf("expected nomad.ProviderConfig, got %T", providerData))

	client := providerConfig.Client()

	// Register a job with a Nomad-native service to create a service registration.
	job := &api.Job{
		ID:          pointerOf("services-test-" + serviceName),
		Name:        pointerOf("services-test-" + serviceName),
		Type:        pointerOf("service"),
		Datacenters: []string{"dc1"},
		TaskGroups: []*api.TaskGroup{
			{
				Name:  pointerOf("web"),
				Count: pointerOf(1),
				Networks: []*api.NetworkResource{
					{
						DynamicPorts: []api.Port{
							{Label: "http", To: 8080},
						},
					},
				},
				Services: []*api.Service{
					{
						Name:      serviceName,
						PortLabel: "http",
						Provider:  "nomad",
						Tags:      []string{"http", "test"},
					},
				},
				Tasks: []*api.Task{
					{
						Name:   "server",
						Driver: "docker",
						Config: map[string]interface{}{
							"image":   "busybox:1",
							"command": "httpd",
							"args":    []string{"-v", "-f", "-p", "8080"},
							"ports":   []string{"http"},
						},
					},
				},
			},
		},
	}

	_, _, err := client.Jobs().Register(job, &api.WriteOptions{Namespace: namespace})
	must.NoError(t, err, must.Sprintf("failed to register test job"))

	// Wait for the service to be registered.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		services, _, err := client.Services().Get(serviceName, &api.QueryOptions{Namespace: namespace})
		if err == nil && len(services) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Cleanup(func() {
		client.Jobs().Deregister(*job.ID, true, &api.WriteOptions{Namespace: namespace})
	})
}

func pointerOf[T any](v T) *T {
	return &v
}
