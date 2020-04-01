module github.com/terraform-providers/terraform-provider-nomad

go 1.14

exclude (
	github.com/Sirupsen/logrus v1.1.0
	github.com/Sirupsen/logrus v1.1.1
	github.com/Sirupsen/logrus v1.2.0
	github.com/Sirupsen/logrus v1.3.0
	github.com/Sirupsen/logrus v1.4.0
	github.com/Sirupsen/logrus v1.4.1
)

require (
	github.com/google/go-cmp v0.3.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl v0.0.0-20170504190234-a4b07c25de5f
	github.com/hashicorp/nomad/api v0.0.0-20200401211951-3e99ef7b1f7c
	github.com/hashicorp/terraform-plugin-sdk v1.0.0
	github.com/hashicorp/vault v0.10.4
	github.com/mitchellh/mapstructure v1.1.2
	github.com/stretchr/testify v1.3.0
)
