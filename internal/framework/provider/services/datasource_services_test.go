// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package services_test

import (
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
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
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.nomad_services.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("nomad-services"),
					),
				},
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

func registerTestService(t *testing.T, serviceName, namespace string) {
	t.Helper()

	providerData := testutil.SDKV2ProviderMeta(t)()
	providerConfig, ok := providerData.(nomad.ProviderConfig)
	if !ok {
		t.Fatalf("expected nomad.ProviderConfig, got %T", providerData)
	}

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
	if err != nil {
		t.Fatalf("failed to register test job: %v", err)
	}

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
