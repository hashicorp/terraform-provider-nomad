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
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/hcl v1.0.1-0.20201016140508-a07e7d50bbee // indirect
	github.com/hashicorp/nomad v0.12.5-0.20201029140339-d6255129a300
	github.com/hashicorp/nomad/api v0.0.0-20201028165800-38e23b62a770
	github.com/hashicorp/terraform-plugin-sdk v1.16.0
	github.com/hashicorp/terraform-plugin-test v1.4.3 // indirect
	github.com/hashicorp/vault v0.10.4
	github.com/mitchellh/mapstructure v1.3.1 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/stretchr/testify v1.6.1
	golang.org/x/build v0.0.0-20190111050920-041ab4dc3f9d // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
)

replace (
    github.com/hashicorp/go-getter => github.com/hashicorp/go-getter v1.4.0
)
