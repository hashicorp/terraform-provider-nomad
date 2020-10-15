module github.com/hashicorp/terraform-provider-nomad

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
	github.com/google/go-cmp v0.5.0
	github.com/hashicorp/consul/api v1.7.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/hcl v1.0.1-0.20191016231534-914dc3f8dd7c
	github.com/hashicorp/nomad v0.12.5
	github.com/hashicorp/nomad/api v0.0.0-20200917200329-ec7bf9de21bf
	github.com/hashicorp/terraform-plugin-sdk v1.16.0
	github.com/hashicorp/vault v0.10.4
	github.com/kr/pretty v0.2.0
	github.com/mitchellh/mapstructure v1.3.1
	github.com/stretchr/testify v1.5.1
)
