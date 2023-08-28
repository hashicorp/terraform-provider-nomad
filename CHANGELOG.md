## 2.0.1 (Unreleased)

## 2.0.0 (August 28th, 2023)

* **New Resource**: `nomad_node_pool` manages node pools ([#340](https://github.com/hashicorp/terraform-provider-nomad/pull/340))
* **New Resource**: `nomad_variable` manages Nomad variables ([#325](https://github.com/hashicorp/terraform-provider-nomad/pull/325))
* **New Data Source**: `nomad_allocations` to retrieve a list of allocations ([#358](https://github.com/hashicorp/terraform-provider-nomad/pull/358))
* **New Data Source**: `nomad_node_pool` and `nomad_node_pools` to retrieves one or mode node pools ([#340](https://github.com/hashicorp/terraform-provider-nomad/pull/340))
* **New Data Source**: `nomad_variable` retrieves a Nomad variable ([#325](https://github.com/hashicorp/terraform-provider-nomad/pull/325))

BACKWARDS INCOMPATIBILITIES:
* provider: Terraform Plugin SDK upgraded to v2.10.1. **Terraform versions prior to 0.12 are no longer supported.** ([#339](https://github.com/hashicorp/terraform-provider-nomad/issues/339))
* resource/nomad_job: Switch to HCL2 parsing by default. Jobs that require HCL1 parsing must set `hcl1 = true`. ([#343](https://github.com/hashicorp/terraform-provider-nomad/pull/343))
* resource/nomad_job: Deprecate field `allocation_ids` and do not retrieve the job's allocations by default. Set `read_allocation_ids` to `true` if you must retain existing behavior, but consider using the `nomad_allocations` data source instead. ([#357](https://github.com/hashicorp/terraform-provider-nomad/pull/357))

DEPRECATIONS:
* resource/nomad_volume: The `nomad_volume` resource has been deprecated. Use the new `nomad_csi_volume_registration` resource instead. ([#344](https://github.com/hashicorp/terraform-provider-nomad/pull/344))
* resource/nomad_external_volume: The `nomad_external_volume` resource has been deprecated. Use the new `nomad_csi_volume` resource instead. ([#344](https://github.com/hashicorp/terraform-provider-nomad/pull/344))

IMPROVEMENTS:
* **Target Nomad 1.6.0**: updated the nomad client to support Nomad API and jobspec version 1.6.0 ([#345](https://github.com/hashicorp/terraform-provider-nomad/pull/345))
* provider: add `skip_verify` configuration to skip TLS verification ([#319](https://github.com/hashicorp/terraform-provider-nomad/pull/319))
* provider: update Go to 1.21.0 ([#369](https://github.com/hashicorp/terraform-provider-nomad/pull/369))
* data source/nomad_namespace: add `node_pool_config` attribute ([#355](https://github.com/hashicorp/terraform-provider-nomad/pull/355))
* resource/nomad_acl_policy: add support for `job_acl` ([#314](https://github.com/hashicorp/terraform-provider-nomad/pull/314))
* resource/nomad_csi_volume: add support to import existing volumes. ([#359](https://github.com/hashicorp/terraform-provider-nomad/pull/359)]
* resource/nomad_csi_volume_registration: add support to import existing volumes. ([#359](https://github.com/hashicorp/terraform-provider-nomad/pull/359)]
* resource/nomad_job: add support to import existing jobs. ([#359](https://github.com/hashicorp/terraform-provider-nomad/pull/359)]
* resource/nomad_namespace: add `node_pool_config` attribute ([#355](https://github.com/hashicorp/terraform-provider-nomad/pull/355))

BUG FIXES:
* data source/acl_auth_method: fix a bug where the values of `max_token_ttl` and `discovery_ca_pem` were not persisted to state. ([#339](https://github.com/hashicorp/terraform-provider-nomad/issues/339))
* data source/acl_token: fix a bug where the value of `expiration_ttl` was not persisted to state. ([#339](https://github.com/hashicorp/terraform-provider-nomad/issues/339))
* data source/namespace: use type list to represent capabilities so its values can be indexed with Terraform SDKv2 ([#339](https://github.com/hashicorp/terraform-provider-nomad/issues/339))
* data source/nomad_volume: fix panic when reading volume ([#323](https://github.com/hashicorp/terraform-provider-nomad/pull/323))
* resources/nomad_acl_binding_rule: fix a bug where `bind_name` was required even when `bind_type` was `management`. ([#330](https://github.com/hashicorp/terraform-provider-nomad/pull/330))
* resources/nomad_job: fix a bug that prevented deployments for jobs in namespaces other than `default` from being monitored. ([#347](https://github.com/hashicorp/terraform-provider-nomad/pull/347))
* resource/nomad_job: fix a bug that could result in unnecessary plan diffs from irrelevant changes. ([#356](https://github.com/hashicorp/terraform-provider-nomad/pull/356))
* resource/nomad_volume: fix a bug that caused `capacity_min` and `capacity_max` to be mandatory. ([#363](https://github.com/hashicorp/terraform-provider-nomad/pull/363))
* resource/nomad_volume: fix a bug where `topology_request` was not persisted to state. ([#342](https://github.com/hashicorp/terraform-provider-nomad/pull/342)
* resource/nomad_external_volume: fix a bug where `topology_request` was not persisted to state. ([#342](https://github.com/hashicorp/terraform-provider-nomad/pull/342)

## 1.4.20 (April 20, 2023)

IMPROVEMENTS:
* **Target Nomad 1.5.2**: updated the nomad client to support Nomad API, jobspec, and features of version 1.5.2 ([#305](https://github.com/hashicorp/terraform-provider-nomad/pull/305))
* **New Resource**: `nomad_acl_auth_method` manages ACL auth methods in Nomad ([#305](https://github.com/hashicorp/terraform-provider-nomad/pull/305))
* **New Resource**: `nomad_acl_binding_rule` manages ACL binding rules in Nomad ([#305](https://github.com/hashicorp/terraform-provider-nomad/pull/305))

## 1.4.19 (October 20, 2022)

IMPROVEMENTS:
* **Target Nomad 1.4.1**: updated the nomad client to support Nomad API, jobspec, and fetures of version 1.4.1 ([#291](https://github.com/hashicorp/terraform-provider-nomad/pull/291))
* **New Resource**: `nomad_acl_role` manages ACL roles in Nomad ([#284](https://github.com/hashicorp/terraform-provider-nomad/pull/284))
* **New Data Source**: `nomad_acl_role` and `nomad_acl_roles` retrieves and lists ACL roles ([#284](https://github.com/hashicorp/terraform-provider-nomad/pull/284))

IMPROVEMENTS:
* resources/nomad_acl_token: add support for `role` and `expiration_ttl` ([#286](https://github.com/hashicorp/terraform-provider-nomad/pull/286))
* resources/nomad_namespace: add support for `meta` and `capabilities` ([#287](https://github.com/hashicorp/terraform-provider-nomad/pull/287))
* data source/nomad_namespace: add support for `meta` and `capabilities` ([#287](https://github.com/hashicorp/terraform-provider-nomad/pull/287))

## 1.4.18 (August 31, 2022)

* **Target Nomad 1.3.4**: updated the nomad client to support Nomad API and jobspec version 1.3.4 ([#282](https://github.com/hashicorp/terraform-provider-nomad/pull/282))

IMPROVEMENTS:
* provider: add `ignore_env_vars` configuration to allow specifying environment variables that should not be loaded by the provider. ([#281](https://github.com/hashicorp/terraform-provider-nomad/pull/281))
* provider: ignore `NOMAD_NAMESPACE` and `NOMAD_REGION` when running in Terraform Cloud by default. ([#281](https://github.com/hashicorp/terraform-provider-nomad/pull/281))

## 1.4.17 (June 9, 2022)

* **Target Nomad 1.3.0**: updated the Nomad client to support Nomad API and jobspec version 1.3.0 ([#270](https://github.com/hashicorp/terraform-provider-nomad/issues/270))

BACKWARDS INCOMPATIBILITIES:
* provider: Don't read the `NOMAD_NAMESPACE` environment variable. This is a potentially breaking change, as it modifies the previous behaviour, but this was never intended to be supported. If you relied on this environment variable make sure you set the namespace in each resource before upgrading. ([#271](https://github.com/hashicorp/terraform-provider-nomad/pull/271))

IMPROVEMENTS:
* provider: update Go to 1.17 ([#270](https://github.com/hashicorp/terraform-provider-nomad/pull/270))
* resource/nomad_external_volume: add support for topologies ([#270](https://github.com/hashicorp/terraform-provider-nomad/issues/270))
* resource/nomad_job: allow passing Consul and Vault token through the job resource ([#261](https://github.com/hashicorp/terraform-provider-nomad/pull/261))
* resource/nomad_volume: add support for topologies ([#270](https://github.com/hashicorp/terraform-provider-nomad/issues/270))

BUG FIXES:
* resource/external_volume: fixed a bug where volume mount flags were not set ([#266](https://github.com/hashicorp/terraform-provider-nomad/pull/266))
* resource/scheduler_config: fixed an issue that caused the value of `memory_oversubscription_enabled` to never be set ([#259](https://github.com/hashicorp/terraform-provider-nomad/pull/259))
* resource/volume: fixed a bug where volume mount flags were not set ([#269](https://github.com/hashicorp/terraform-provider-nomad/pull/269))

## 1.4.16 (November 24, 2021)

* **Target Nomad 1.2.0**: updated the Nomad client to support Nomad API and jobspec version 1.2.0 ([#256](https://github.com/hashicorp/terraform-provider-nomad/pull/256))

BUG FIXES:
* data source/nomad_plugin: wait for the correct amount of expected controllers ([#234](https://github.com/hashicorp/terraform-provider-nomad/pull/234))
* data source/nomad_plugin: wait for the correct amount of healthy nodes ([#235](https://github.com/hashicorp/terraform-provider-nomad/pull/235))

## 1.4.15 (May 19, 2021)

***This is the last release to support Terraform v0.11***

* **Target Nomad 1.1.0**: updated the Nomad client to support Nomad API and jobspec version 1.1.0 ([#229](https://github.com/hashicorp/terraform-provider-nomad/issues/229))
* **New Resource**: `nomad_external_volume` creates and registers an external volume with Nomad ([#228](https://github.com/hashicorp/terraform-provider-nomad/pull/228))

IMPROVEMENTS:
* provider: update Go to 1.16 to add support for `darwin/arm64` ([#218](https://github.com/hashicorp/terraform-provider-nomad/pull/218))
* resource/nomad_job: allow custom timeout values on create and update ([#227](https://github.com/hashicorp/terraform-provider-nomad/pull/227)

BUG FIXES:
* provider: update Nomad API to prevent header access panic ([#226](https://github.com/hashicorp/terraform-provider-nomad/pull/226))

## 1.4.14 (April 06, 2021)

* **Target Nomad 1.0.4**: updated the nomad client to support Nomad API and jobspec version 1.0.4 ([#202](https://github.com/hashicorp/terraform-provider-nomad/pull/202))([#206](https://github.com/hashicorp/terraform-provider-nomad/issues/206))

IMPROVEMENTS:
* provider: add support for custom HTTP headers ([#203](https://github.com/hashicorp/terraform-provider-nomad/issues/203))
* resource/nomad_job: allow passing HCL2 variables to job ([#211](https://github.com/hashicorp/terraform-provider-nomad/issues/211))
* resource/nomad_job: support injecting consul token into job ([#213](https://github.com/hashicorp/terraform-provider-nomad/issues/213))


## 1.4.13 (January 20, 2021)

BUG FIXES:
* provider: Revert Terraform Plugin SDK to v1.16.0 to support Terraform 0.11.x ([#196](https://github.com/hashicorp/terraform-provider-nomad/pull/196))

## 1.4.12 (January 20, 2021)

***This release will not work with Terraform v0.11.x. Please use v1.14.13***

* **Target Nomad 1.0.2**: updated the nomad client to support Nomad API version 1.0.2 ([#161](https://github.com/hashicorp/terraform-provider-nomad/issues/161))

BACKWARDS INCOMPATIBILITIES:
* resource/nomad_volume: `mount_options` is now a list, so configuration files need to be updated to remove the `=` character (from `mount_options = {...}` to `mount_options {...}`) ([#188](https://github.com/hashicorp/terraform-provider-nomad/pull/188))

FEATURES:
* provider: add support for HTTP basic auth ([#189](https://github.com/hashicorp/terraform-provider-nomad/pull/189))
* resource/nomad_job: add initial support for jobspec parsing with HCL2 ([#185](https://github.com/hashicorp/terraform-provider-nomad/pull/185))

IMPROVEMENTS:
* provider: Terraform Plugin SDK bumped to v2.4.0 ([#161](https://github.com/hashicorp/terraform-provider-nomad/issues/161))
* provider: support TLS certs configured by strings ([#184](https://github.com/hashicorp/terraform-provider-nomad/pull/184))
* resource/nomad_job: Nomad job parsing is pulled directly from nomad instead of being copied over ([#161](https://github.com/hashicorp/terraform-provider-nomad/issues/161))

BUG FIXES:
* numerous (invalid but unused) fields from the task group and job schema have been removed ([#161](https://github.com/hashicorp/terraform-provider-nomad/issues/161))
* resource/nomad_volume: fixed an issue where `mount_options` would always cause a change ([#188](https://github.com/hashicorp/terraform-provider-nomad/pull/188))

## 1.4.11 (December 07, 2020)

* **Target Nomad 1.0.0-rc1**: updated the nomad client to support Nomad API version 1.0.0-rc1 ([#175](https://github.com/hashicorp/terraform-provider-nomad/issues/175))
* **New Data Source**: `nomad_scheduler_config` retrieves the scheduler configuration in Nomad ([#168](https://github.com/hashicorp/terraform-provider-nomad/pull/168))

BUG FIXES:
* resource/nomad_volume: register volume in the namespace specified in the configuration ([#169](https://github.com/hashicorp/terraform-provider-nomad/pull/169))

## 1.4.10 (October 30, 2020)

* **Target Nomad 1.0.0-beta2**: updated the nomad client to support Nomad API version 1.0.0-beta2 ([#158](https://github.com/hashicorp/terraform-provider-nomad/issues/158))

FEATURES:
* **New Resource**: `nomad_scheduler_config` allows management of cluster scheduler configuration ([#157](https://github.com/hashicorp/terraform-provider-nomad/pull/157))
* **New Data Source**: `nomad_scaling_policies` and `nomad_scaling_policies` retrieves and lists scaling policies ([#162](https://github.com/hashicorp/terraform-provider-nomad/pull/162))
* **New Data Source**: `nomad_datacenters` lists the datacenters in a Nomad cluster ([#165](https://github.com/hashicorp/terraform-provider-nomad/pull/165))

IMPROVEMENTS:
* resource/nomad_volume: added `mount_options` argument ([#147](https://github.com/hashicorp/terraform-provider-nomad/pull/147))

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
