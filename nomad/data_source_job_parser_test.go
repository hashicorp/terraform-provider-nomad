package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

// get conf, test, destroy

// passing case

func TestAccDataSourceNomadJobParser_Basic(t *testing.T) {

	resourceName := "data.nomad_job_parser.test_job"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: testJobParseConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName, "hcl", testDataSourceHCLJob),
					resource.TestCheckResourceAttr(
						resourceName, "canonicalize", "false"),
					resource.TestCheckResourceAttr(
						resourceName, "json", testDataSourceJobParserJSON),
				),
			},
		},
	})
}

func testJobParseConfig() string {
	return fmt.Sprintf(`
	data "nomad_job_parser" "test_job" {
	  hcl = <<EOT
%s
	  EOT
	}
	`, testDataSourceHCLJob)

}

const testDataSourceHCLJob = `
job "example" {
  datacenters = ["dc1"]
  
  group "cache" {
    task "redis" {
      driver = "docker"
  
      config {
        image = "redis:3.2"
  
        port_map {
          db = 6379
        }
      }
  
      resources {
        cpu    = 500
        memory = 256
  
        network {
          mbits = 10
          port "db" {}
        }
      }
    }
  }
}`

const testDataSourceJobParserJSON = `
{
	"Stop": null,
	"Region": null,
	"Namespace": null,
	"ID": "example",
	"ParentID": null,
	"Name": "example",
	"Type": null,
	"Priority": null,
	"AllAtOnce": null,
	"Datacenters": [
	  "dc1"
	],
	"Constraints": null,
	"Affinities": null,
	"TaskGroups": [
	  {
		"Name": "cache",
		"Count": null,
		"Constraints": null,
		"Affinities": null,
		"Tasks": [
		  {
			"Name": "redis",
			"Driver": "docker",
			"User": "",
			"Lifecycle": null,
			"Config": {
			  "image": "redis:3.2",
			  "port_map": [
				{
				  "db": 6379
				}
			  ]
			},
			"Constraints": null,
			"Affinities": null,
			"Env": null,
			"Services": null,
			"Resources": {
			  "CPU": 500,
			  "MemoryMB": 256,
			  "DiskMB": null,
			  "Networks": [
				{
				  "Mode": "",
				  "Device": "",
				  "CIDR": "",
				  "IP": "",
				  "MBits": 10,
				  "DNS": null,
				  "ReservedPorts": null,
				  "DynamicPorts": [
					{
					  "Label": "db",
					  "Value": 0,
					  "To": 0,
					  "HostNetwork": ""
					}
				  ]
				}
			  ],
			  "Devices": null,
			  "IOPS": null
			},
			"RestartPolicy": null,
			"Meta": null,
			"KillTimeout": null,
			"LogConfig": null,
			"Artifacts": null,
			"Vault": null,
			"Templates": null,
			"DispatchPayload": null,
			"VolumeMounts": null,
			"Leader": false,
			"ShutdownDelay": 0,
			"KillSignal": "",
			"Kind": ""
		  }
		],
		"Spreads": null,
		"Volumes": null,
		"RestartPolicy": null,
		"ReschedulePolicy": null,
		"EphemeralDisk": null,
		"Update": null,
		"Migrate": null,
		"Networks": null,
		"Meta": null,
		"Services": null,
		"ShutdownDelay": null,
		"StopAfterClientDisconnect": null,
		"Scaling": null
	  }
	],
	"Update": null,
	"Multiregion": null,
	"Spreads": null,
	"Periodic": null,
	"ParameterizedJob": null,
	"Dispatched": false,
	"Payload": null,
	"Reschedule": null,
	"Migrate": null,
	"Meta": null,
	"ConsulToken": null,
	"VaultToken": null,
	"NomadTokenID": null,
	"Status": null,
	"StatusDescription": null,
	"Stable": null,
	"Version": null,
	"SubmitTime": null,
	"CreateIndex": null,
	"ModifyIndex": null,
	"JobModifyIndex": null
  }
  `
