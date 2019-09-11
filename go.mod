module github.com/terraform-providers/terraform-provider-nomad

exclude (
	github.com/Sirupsen/logrus v1.1.0
	github.com/Sirupsen/logrus v1.1.1
	github.com/Sirupsen/logrus v1.2.0
	github.com/Sirupsen/logrus v1.3.0
	github.com/Sirupsen/logrus v1.4.0
	github.com/Sirupsen/logrus v1.4.1
)

require (
	github.com/hashicorp/go-hclog v0.8.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-version v1.1.0
	github.com/hashicorp/hcl v0.0.0-20170504190234-a4b07c25de5f
	github.com/hashicorp/nomad v0.9.1
	github.com/hashicorp/nomad/api v0.0.0-20190506160519-84306b0bfbed
	github.com/hashicorp/terraform v0.12.8
	github.com/hashicorp/vault v0.10.4
	github.com/hashicorp/yamux v0.0.0-20180917205041-7221087c3d28 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/stretchr/testify v1.3.0
)
