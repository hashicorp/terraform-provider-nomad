---
name: Bug report
about: Create a report to help us improve
title: ''
labels: ''
assignees: ''

---

Hi there,

Thank you for opening an issue. Please note that we try to keep the Terraform issue tracker reserved for bug reports and feature requests. For general usage questions, please see: https://www.terraform.io/community.html.

### Terraform Version
Run `terraform -v` to show the version. If you are not running the latest version of Terraform, please upgrade because your issue may have already been fixed. 

This command will also output the provider version, please include that as well.

### Nomad Version
Run `nomad server members` in your target node to view which version of Nomad is running. Make sure to include the entire version output.

### Provider Configuration
Which values are you setting in the provider configuration? 

```hcl
provider "nomad" {
  ...
}
```

### Environment Variables
Do you have any Nomad specific environment variable set in the machine running Terraform?

```sh
env | grep "NOMAD_"
```

### Affected Resource(s)
Please list the resources as a list, for example:
- nomad_job
- nomad_acl_policy

If this issue appears to affect multiple resources, it may be an issue with Terraform's core, so please mention this.

### Terraform Configuration Files
```hcl
# Copy-paste your Terraform configurations here - for large Terraform configs,
# please use a service like Dropbox and share a link to the ZIP file. For
# security, you can also encrypt the files using our GPG public key.
```

### Debug Output
Please provider a link to a GitHub Gist containing the complete debug output: https://www.terraform.io/docs/internals/debugging.html. Please do NOT paste the debug output in the issue; just paste a link to the Gist.

### Panic Output
If Terraform produced a panic, please provide a link to a GitHub Gist containing the output of the `crash.log`.

### Expected Behavior
What should have happened?

### Actual Behavior
What actually happened?

### Steps to Reproduce
Please list the steps required to reproduce the issue, for example:
1. `terraform apply`

### Important Factoids
Are there anything atypical about your accounts that we should know? For example: Do you have [ACL](https://www.nomadproject.io/guides/security/acl.html) enabled? Multi-region deployment?

### References
Are there any other GitHub issues (open or closed) or Pull Requests that should be linked here? For example:
- GH-1234
