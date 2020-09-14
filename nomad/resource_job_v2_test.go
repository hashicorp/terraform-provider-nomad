package nomad

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestResourceJobV2_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceJob_basic,
			},
			{
				Config:       testResourceJob_basic,
				ResourceName: "nomad_job_v2.job",
				ImportState:  true,
			},
		},
	})
}

const testResourceJob_basic = `
// A job with the minimal configuration to test the value of all the default
// blocks
resource "nomad_job_v2" "job" {
	job {
		name        = "example"
		datacenters = ["dc1"]

		group {
			name = "cache"

			task {
				name   = "redis"
				driver = "docker"
				config = jsonencode({
					image = "redis:3.2"
				})
			}
		}
	}
}

// A job with the minimal configuration to test the defaults for the batch
// scheduler
resource "nomad_job_v2" "batch" {
	job {
		name        = "example-batch"
		datacenters = ["dc1"]
		type        = "batch"

		group {
			name = "cache"

			task {
				name   = "redis"
				driver = "docker"
				config = jsonencode({
					image = "redis:3.2"
				})
			}
		}
	}
}

// A job with the minimal configuration to test the defaults for the system
// scheduler
resource "nomad_job_v2" "system" {
	job {
		name        = "example-system"
		datacenters = ["dc1"]
		type        = "system"

		group {
			name = "cache"

			task {
				name   = "redis"
				driver = "docker"
				config = jsonencode({
					image = "redis:3.2"
				})
			}
		}
	}
}

// Setting a block to its default value should not confuse Terraform
resource "nomad_job_v2" "default" {
	job {
		name        = "example-default"
		datacenters = ["dc1"]
		type        = "service"

		update {
			auto_promote = false
		}

		group {
			name = "cache"

			ephemeral_disk {
				migrate = false
				size    = 300
				sticky  = false
			}

			migrate {
				max_parallel = 1
			}

			restart {
				attempts = 2
				delay    = "15s"
				interval = "30m0s"
				mode     = "fail"
			}

			reschedule {
				attempts       = 0
				delay          = "30s"
				delay_function = "exponential"
				interval       = "0s"
				max_delay      = "1h0m0s"
				unlimited      = true
			}

			task {
				name   = "redis"
				driver = "docker"
				config = jsonencode({
					image = "redis:3.2"
				})
			}
		}
	}
}

// A job with all values set to try to test all code paths
resource "nomad_job_v2" "all" {

	job {
		namespace = "default"
		priority  = 100
		type      = "batch"
		region    = "global"
		meta = {
			foo = "bar"
		}
		all_at_once  = true
		datacenters  = ["dc1"]
		name         = "all"
		vault_token  = "foobar"
		consul_token = "var"

		constraint {
			operator  = "distinct_hosts"
			value     = "true"
		}

		affinity {
			attribute = "$${node.datacenter}"
			operator  = ">="
			value     = "us-west1"
			weight    = 100
		}

		spread {
			attribute = "$${node.datacenter}"
			weight    = 100

			target {
				value   = "us-east1"
				percent = 40
			}

			target {
				value   = "us-west1"
				percent = 60
			}
		}

		group {
			name = "cache"
			meta = {
				foo = "bar"
			}
			count                        = 3
			shutdown_delay               = "6s"
			stop_after_client_disconnect = "1h"

			constraint {
				operator  = "distinct_hosts"
				value     = "true"
			}

			affinity {
				attribute = "$${node.datacenter}"
				operator  = "<"
				value     = "us-west1"
				weight    = 100
			}

			spread {
				attribute = "$${node.datacenter}"
				weight    = 100
			}

			ephemeral_disk {
				migrate = true
				size    = 500
				sticky  = true
			}

			network {
				mbits = 20
				mode  = "host"

				port {
					label = "http"
					to    = 1234
				}

				dns {
					servers  = ["1.2.3.4"]
					searches = ["1.2.3.4"]
					options  = ["1.2.3.4"]
				}
			}

			restart {
				attempts = 6
				delay    = "4s"
				interval = "70s"
				mode     = "delay"
			}

			service {
				meta = {
					foo = "bar"
				}
				name                = "test"
				port                = "http"
				tags                = ["http", "test"]
				canary_tags         = ["canay", "test"]
				enable_tag_override = true
				task                = "redis"

				check {
					type     = "tcp"
					port     = "db"
					interval = "10s"
					timeout  = "2s"
				}
			}

			task {
				name   = "redis"
				config = jsonencode({
					image = "redis:3.2"
				})
				env = {
					foo = "bar"
				}
				meta = {
					bar = "var"
				}
				driver         = "docker"
				kill_timeout   = "1m"
				kill_signal    = "SIGTERM"
				leader         = true
				shutdown_delay = "10s"
				user           = "remi"
				kind           = "foo"

				artifact {
					destination = "test"
					mode        = "file"
					source      = "https://example.com/file.tar.gz"
					options = {
						checksum = "md5:df6a4178aec9fbdc1d6d7e3634d1bc33"
					}
				}

				artifact {
					destination = "test2"
					mode        = "dir"
					source      = "https://example.com/file.tar.gz"
					options = {
						checksum = "md5:df6a4178aec9fbdc1d6d7e3634d1bc33"
					}
				}

				constraint {
					attribute = "foo"
					operator  = "version"
					value     = "1"
				}

				affinity {
					attribute = "$${node.datacenter}"
					operator  = "<"
					value     = "us-west1"
					weight    = 100
				}

				dispatch_payload {
					file = "config.json"
				}

				lifecycle {
					hook    = "prestart"
					sidecar = false
				}

				logs {
					max_files     = 1
					max_file_size = 1
				}

				resources {
					cpu    = 1234
					memory = 28

					device {
						name  = "foo"
						count = 2

						constraint {
							attribute = "$${device.attr.memory}"
							operator  = ">="
							value     = "2 GiB"
						}

						affinity {
							attribute = "$${device.attr.memory}"
							operator  = ">="
							value     = "4 GiB"
							weight    = 75
						}
					}

					network {
						mbits = 20
						mode  = "host"

						port {
							to = 1234
						}

						dns {
							servers  = ["1.2.3.4"]
							searches = ["1.2.3.4"]
							options  = ["1.2.3.4"]
						}
					}
				}

				// service {}

				template {
					source          = "local/redis.conf.tpl"
					destination     = "local/redis.conf"
					change_mode     = "signal"
					change_signal   = "SIGINT"
					splay           = "6s"
					left_delimiter  = "[["
					perms           = "755"
					right_delimiter = "]]"
					vault_grace     = "1s"
				}

				template {
					data = <<-EOH
						FOO=bar
					EOH
					destination = "nowhere"
					env         = true
				}

				// We cannot test vault here as it requires an appropriate config
				// for the Nomad server

				volume_mount {
					volume      = "foo"
					destination = "/etc/ssl/certs"
					read_only   = true
				}
			}

			volume {
				name      = "foo"
				type      = "host"
				source    = "ca-certificates"
				read_only = true
			}
		}

		// migrate {
		// 	max_parallel      = 6
		// 	health_check      = "task_states"
		// 	min_healthy_time  = "1m"
		// 	healthy_deadline  = "1h"
		// }

		parameterized {
			meta_optional = ["one"]
			meta_required = ["two"]
			payload       = "required"
		}

		periodic {
			cron             = "*/15 * * * * *"
			prohibit_overlap = true
			time_zone        = "America/New_York"
		}

		// reschedule {
		// 	attempts       = 5
		// 	interval       = "1h"
		// 	delay          = "10m"
		// 	delay_function = "fibonacci"
		// 	max_delay      = "120m"
		// 	unlimited      = false
		// }
	}
}
`
