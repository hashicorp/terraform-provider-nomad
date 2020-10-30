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
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/nomad v0.12.5-0.20201029140339-d6255129a300
	github.com/hashicorp/nomad/api v0.0.0-20201028165800-38e23b62a770
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.1.0
	github.com/hashicorp/vault v0.10.4
	github.com/stretchr/testify v1.6.1
	github.com/ulikunitz/xz v0.5.7 // indirect
)

// Force using go-getter version that is compatible with terraform plugin
replace github.com/hashicorp/go-getter => github.com/hashicorp/go-getter v1.4.0
