# nomad/core

The Nomad Terraform provider project uses
[Go modules](https://github.com/golang/go/wiki/Modules) to manage its
dependencies, but [Nomad](https://github.com/hashicorp/nomad/) is not
fully migrated yet, so this directory contains dependencies from Nomad that
are not exposed in the `github.com/hashicorp/nomad/api` module.

Current base version: [ca5cd15eeffd40b68043c94eefff1ec7e6dc703f](https://github.com/hashicorp/nomad/tree/ca5cd15eeffd40b68043c94eefff1ec7e6dc703f)

## Updating Nomad dependency

Ideally you shouldn't have to, but if you need to access something from Nomad
that is not exposed via the `github.com/hashicorp/nomad/api` module, copy it
over to this directory mimicking the original folder and file path structure
of the commit linked above.

If your update requires a newer version of Nomad, you should also update all
other dependencies that are currently in this directory and the commit hash
above.

The only modification you should do from the original files is to change
`import` paths from `github.com/hashicorp/nomad` to
`github.com/github.com/hashicorp/terraform-provider-nomad/nomad/core`.

You can also copy just a few chucks of the original file to avoid bringing
extra dependencie.
