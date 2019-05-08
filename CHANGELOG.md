## 1.4.0 (Unreleased)

IMPROVEMENTS:
* **Target Nomad 0.9.1**: updated the nomad client to support Nomad API verison 0.9.1 ([#61](https://github.com/terraform-providers/terraform-provider-nomad/issues/61))

INTERNAL:

* Acceptance tests are gated based on capability (skipped if Nomad endpoint doesn't have certain capability being tested) ([#53](https://github.com/terraform-providers/terraform-provider-nomad/issues/53))
* Terraform SDK bumped to v0.12-early11 ([#61](https://github.com/terraform-providers/terraform-provider-nomad/issues/61))
* Provider is switched to go modules and the Terraform SDK vendored is bumped to v0.11 ([#55](https://github.com/terraform-providers/terraform-provider-nomad/issues/55))

## 1.3.0 (December 20, 2018)

IMPROVEMENTS: 

* **Target Nomad 0.8.6**: updated the nomad client to support Nomad API version 0.8.6 ([#46](https://github.com/terraform-providers/terraform-provider-nomad/issues/46))
* **New Data Source**: `nomad_namespaces` lists the available namespaces (Nomad Enterprise only) (thanks, @jorgemarey!) ([#41](https://github.com/terraform-providers/terraform-provider-nomad/pull/41))
* **New Data Source**: `nomad_deployments` lists all deployments (thanks, @slapula!) ([#34](https://github.com/terraform-providers/terraform-provider-nomad/pull/34))
* **New Data Source**: `nomad_job` lists the jobs (thanks, @slapula!) ([#32](https://github.com/terraform-providers/terraform-provider-nomad/pull/32))
* **JSON Job Spec**: Job specification can be provided as JSON (thanks, @smintz!) ([#42](https://github.com/terraform-providers/terraform-provider-nomad/pull/42))

## 1.2.0 (March 28, 2018)

FEATURES:

* **New Resource**: `nomad_acl_token` allows management of ACL tokens ([#20](https://github.com/terraform-providers/terraform-provider-nomad/issues/20))
* **New Resource**: `nomad_acl_policy` allows management of ACL policies ([#22](https://github.com/terraform-providers/terraform-provider-nomad/issues/22))
* **New Data Source**: `nomad_regions` lists the regions in a Nomad cluster ([#24](https://github.com/terraform-providers/terraform-provider-nomad/issues/24))
* **New Resource**: `nomad_quota_specification` allows management of quotas (Nomad Enterprise only) ([#25](https://github.com/terraform-providers/terraform-provider-nomad/issues/25))
* **New Resource**: `nomad_namespace` allows management of namespaces (Nomad Enterprise only) ([#26](https://github.com/terraform-providers/terraform-provider-nomad/issues/26))
* **New Resource**: `nomad_sentinel_policy` manages Sentinel policies (Nomad Enterprise only) ([#27](https://github.com/terraform-providers/terraform-provider-nomad/issues/27))

IMPROVEMENTS:

* Add support for Sentinel policy overrides to Nomad jobs ([#28](https://github.com/terraform-providers/terraform-provider-nomad/issues/28))
* Add support for specifying Vault tokens at the provider level ([#30](https://github.com/terraform-providers/terraform-provider-nomad/issues/30))

## 1.1.0 (December 15, 2017)

IMPROVEMENTS:

* Add support for ACL tokens ([#17](https://github.com/terraform-providers/terraform-provider-nomad/issues/17))

## 1.0.0 (September 27, 2017)

IMPROVEMENTS:

* provider: Update Nomad to v0.6.0 ([#7](https://github.com/terraform-providers/terraform-provider-nomad/issues/7))

## 0.1.0 (June 21, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
