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
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/nomad v1.0.0-rc1
	github.com/hashicorp/nomad/api v0.0.0-20201203182648-6ed0ac873f29
	github.com/hashicorp/terraform-plugin-sdk v1.16.0
	github.com/hashicorp/vault v0.10.4
	github.com/stretchr/testify v1.6.1
)

// Force using go-getter version that is compatible with terraform plugin
replace github.com/hashicorp/go-getter => github.com/hashicorp/go-getter v1.4.0
