## 1.4.10 (Unreleased)
## 1.4.9 (August 13, 2020)

* **Target Nomad 0.12.2**: updated the nomad client to support Nomad API version 0.12.2 ([#140](https://github.com/hashicorp/terraform-provider-nomad/issues/140))

IMPROVEMENTS:
* resource/nomad_job: added `purge_on_destroy` option ([#127](https://github.com/hashicorp/terraform-provider-nomad/issues/127) [#130](https://github.com/hashicorp/terraform-provider-nomad/issues/130))
* data source/nomad_namespace: added new data source to fetch info a Nomad namespace ([#126](https://github.com/hashicorp/terraform-provider-nomad/issues/126))
* data source/nomad_acl_tokens: added new data source to fetch Nomad ACL tokens ([#128](https://github.com/hashicorp/terraform-provider-nomad/issues/128))
* data source/nomad_job_parser: added new data source `nomad_job_parser` parses a jobspec from HCL to JSON ([#136](https://github.com/hashicorp/terraform-provider-nomad/issues/136))


## 1.4.8 (July 09, 2020)

* **Target Nomad 0.12.0**: updated the nomad client to support Nomad API version 0.12.0 ([#121](https://github.com/hashicorp/terraform-provider-nomad/issues/121))

IMPROVEMENTS:
* data source/nomad_plugin: for fetching information on a single CSI plugin ([#123](https://github.com/hashicorp/terraform-provider-nomad/issues/123))
* data source/nomad_plugins: for fetching the list of available CSI plugins ([#123](https://github.com/hashicorp/terraform-provider-nomad/issues/123))
* data source/nomad_volumes: for fetching the list of available CSI volumes ([#123](https://github.com/hashicorp/terraform-provider-nomad/issues/123))
* resource/nomad_volume: for registering a CSI volume ([#213](https://github.com/hashicorp/terraform-provider-nomad/issues/213))

## 1.4.7 (June 12, 2020)

* **Target Nomad 0.11.3**: updated the nomad client to support Nomad API version 0.11.3 ([#113](https://github.com/hashicorp/terraform-provider-nomad/issues/113))

IMPROVEMENTS:
* resource/nomad_job: allow JSON input to have the same format as produced by the `nomad` CLI ([#111](https://github.com/hashicorp/terraform-provider-nomad/pull/111))

BUG FIXES:
* resource/nomad_job: don't panic when JSON input is invalid ([#111](https://github.com/hashicorp/terraform-provider-nomad/pull/111))
* resource/nomad_job: modified job monitoring to account for scenarios where no deployment is created ([#116](https://github.com/hashicorp/terraform-provider-nomad/issues/116))


## 1.4.6 (May 18, 2020)

* **Target Nomad 0.11.2**: updated the nomad client to support Nomad API version 0.11.2 ([#103](https://github.com/hashicorp/terraform-provider-nomad/pull/103))

BUG FIXES:
* resource/nomad_job: deployment info was not being set/unset according to `detach` ([#104](https://github.com/hashicorp/terraform-provider-nomad/issues/104))
* resource/nomad_job: customize diff cannot be run when new jobspec isn't available at time of diff ([#92](https://github.com/hashicorp/terraform-provider-nomad/issues/92))

## 1.4.5 (April 08, 2020)

IMPROVEMENTS:

* **Target Nomad 0.11.0**: updated the nomad client to support Nomad API version 0.11.0 ([#99](https://github.com/hashicorp/terraform-provider-nomad/pull/99))

## 1.4.4 (February 10, 2020)

IMPROVEMENTS:

* **Target Nomad 0.10.3**: updated the nomad client to support Nomad API version 0.10.3 ([#96](https://github.com/hashicorp/terraform-provider-nomad/issues/96))

## 1.4.3 (February 07, 2020)

IMPROVEMENTS:
* data source for ACL tokens ([#88](https://github.com/hashicorp/terraform-provider-nomad/pull/88))
* data source for ACL policies ([#89](https://github.com/hashicorp/terraform-provider-nomad/pull/89))

BUG FIXES:

* resource/nomad_job: Sort TaskGroup volumes to avoid `inconsistent final plan` errors ([#93](https://github.com/hashicorp/terraform-provider-nomad/issues/93))

## 1.4.2 (October 11, 2019)

IMPROVEMENTS:

* **Target Nomad 0.10.0**: updated the nomad client to support Nomad API version 0.10.0 ([#77](https://github.com/hashicorp/terraform-provider-nomad/issues/77))

BUG FIXES:

* resource/nomad_namespace: Prevent Terraform from trying to delete the `default` namespace since this operation is not allowed ([#72](https://github.com/hashicorp/terraform-provider-nomad/issues/72))
* resource/nomad_job: Don't fail planning if the Nomad server is not available ([#66](https://github.com/hashicorp/terraform-provider-nomad/issues/66))

NOTES:

* The provider has switched to the standalone TF SDK, there should be no noticeable impact on compatibility. ([#78](https://github.com/hashicorp/terraform-provider-nomad/issues/78))

## 1.4.1 (July 31, 2019)

IMPROVEMENTS:
* Better support for Nomad namespaces ([#70](https://github.com/hashicorp/terraform-provider-nomad/issues/70))

## 1.4.0 (May 20, 2019)

IMPROVEMENTS:
* **Target Nomad 0.9.1**: updated the nomad client to support Nomad API verison 0.9.1 ([#61](https://github.com/hashicorp/terraform-provider-nomad/issues/61))
* **Option to monitor Job resources until deployment is complete** ([#64](https://github.com/hashicorp/terraform-provider-nomad/issues/64))
* **Richer diffs for Nomad jobs** ([#63](https://github.com/hashicorp/terraform-provider-nomad/issues/63))
* Added `allocation_ids` as an attribute to Nomad Job resources. ([#63](https://github.com/hashicorp/terraform-provider-nomad/issues/63))

INTERNAL:

* Acceptance tests are gated based on capability (skipped if Nomad endpoint doesn't have certain capability being tested) ([#53](https://github.com/hashicorp/terraform-provider-nomad/issues/53))
* Terraform SDK bumped to v0.12.0-rc1, Nomad SDK to 0.9.x ([#61](https://github.com/hashicorp/terraform-provider-nomad/issues/61))
* Provider is switched to go modules ([#55](https://github.com/hashicorp/terraform-provider-nomad/issues/55))

## 1.3.0 (December 20, 2018)

IMPROVEMENTS:

* **Target Nomad 0.8.6**: updated the nomad client to support Nomad API version 0.8.6 ([#46](https://github.com/hashicorp/terraform-provider-nomad/issues/46))
* **New Data Source**: `nomad_namespaces` lists the available namespaces (Nomad Enterprise only) (thanks, @jorgemarey!) ([#41](https://github.com/hashicorp/terraform-provider-nomad/pull/41))
* **New Data Source**: `nomad_deployments` lists all deployments (thanks, @slapula!) ([#34](https://github.com/hashicorp/terraform-provider-nomad/pull/34))
* **New Data Source**: `nomad_job` lists the jobs (thanks, @slapula!) ([#32](https://github.com/hashicorp/terraform-provider-nomad/pull/32))
* **JSON Job Spec**: Job specification can be provided as JSON (thanks, @smintz!) ([#42](https://github.com/hashicorp/terraform-provider-nomad/pull/42))

## 1.2.0 (March 28, 2018)

FEATURES:

* **New Resource**: `nomad_acl_token` allows management of ACL tokens ([#20](https://github.com/hashicorp/terraform-provider-nomad/issues/20))
* **New Resource**: `nomad_acl_policy` allows management of ACL policies ([#22](https://github.com/hashicorp/terraform-provider-nomad/issues/22))
* **New Data Source**: `nomad_regions` lists the regions in a Nomad cluster ([#24](https://github.com/hashicorp/terraform-provider-nomad/issues/24))
* **New Resource**: `nomad_quota_specification` allows management of quotas (Nomad Enterprise only) ([#25](https://github.com/hashicorp/terraform-provider-nomad/issues/25))
* **New Resource**: `nomad_namespace` allows management of namespaces (Nomad Enterprise only) ([#26](https://github.com/hashicorp/terraform-provider-nomad/issues/26))
* **New Resource**: `nomad_sentinel_policy` manages Sentinel policies (Nomad Enterprise only) ([#27](https://github.com/hashicorp/terraform-provider-nomad/issues/27))

IMPROVEMENTS:

* Add support for Sentinel policy overrides to Nomad jobs ([#28](https://github.com/hashicorp/terraform-provider-nomad/issues/28))
* Add support for specifying Vault tokens at the provider level ([#30](https://github.com/hashicorp/terraform-provider-nomad/issues/30))

## 1.1.0 (December 15, 2017)

IMPROVEMENTS:

* Add support for ACL tokens ([#17](https://github.com/hashicorp/terraform-provider-nomad/issues/17))

## 1.0.0 (September 27, 2017)

IMPROVEMENTS:

* provider: Update Nomad to v0.6.0 ([#7](https://github.com/hashicorp/terraform-provider-nomad/issues/7))

## 0.1.0 (June 21, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
