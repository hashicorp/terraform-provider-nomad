## 1.2.0 (Unreleased)

FEATURES:

* **New Resource**: `nomad_acl_token` allows management of ACL tokens [GH-20]
* **New Resource**: `nomad_acl_policy` allows management of ACL policies [GH-22]
* **New Data Source**: `nomad_regions` lists the regions in a Nomad cluster [GH-24]
* **New Resource**: `nomad_quota_specification` allows management of quotas (Nomad Enterprise only) [GH-25]
* **New Resource**: `nomad_namespace` allows management of namespaces (Nomad Enterprise only) [GH-26]
* **New Resource**: `nomad_sentinel_policy` manages Sentinel policies (Nomad Enterprise only) [GH-27]

IMPROVEMENTS:

* Add support for Sentinel policy overrides to Nomad jobs [GH-28]
* Add support for specifying Vault tokens at the provider level [GH-30]

## 1.1.0 (December 15, 2017)

IMPROVEMENTS:

* Add support for ACL tokens ([#17](https://github.com/terraform-providers/terraform-provider-nomad/issues/17))

## 1.0.0 (September 27, 2017)

IMPROVEMENTS:

* provider: Update Nomad to v0.6.0 ([#7](https://github.com/terraform-providers/terraform-provider-nomad/issues/7))

## 0.1.0 (June 21, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
