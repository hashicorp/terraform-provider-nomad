Nomad Terraform Provider
========================

- Website: https://registry.terraform.io/providers/hashicorp/nomad

Maintainers
-----------

This provider plugin is maintained by the Nomad team at [HashiCorp](https://www.hashicorp.com/).

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 1.0.x
-	[Go](https://golang.org/doc/install) 1.21 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/hashicorp/terraform-provider-nomad`

```sh
$ mkdir -p $GOPATH/src/github.com/hashicorp; cd $GOPATH/src/github.com/hashicorp
$ git clone git@github.com:hashicorp/terraform-provider-nomad
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/hashicorp/terraform-provider-nomad
$ make build
```

Using the provider
----------------------

To use a released provider in your Terraform environment, run
[`terraform init`](https://www.terraform.io/docs/commands/init.html) and
Terraform will automatically install the provider. To specify a particular
provider version when installing released providers, see the [Terraform
documentation on provider versioning](https://www.terraform.io/docs/configuration/providers.html#version-provider-versions).

Refer to the section below for instructions on how to to use a custom-built
version of the provider.

For either installation method, documentation about the provider specific
configuration options can be found on the
[provider's website](https://www.terraform.io/docs/providers/nomad/).

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need
[Go](http://www.golang.org) installed on your machine (version 1.20+ is
*required*). You'll also need to correctly setup a
[GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding
`$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put
the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-nomad
...
```

To use the provider binary in a local configuration, create a file called
`.terraformrc` in your home directory and specify a [development
override][tf_docs_dev_overrides] for the `nomad` provider.

```hcl
provider_installation {
  dev_overrides {
    "hashicorp/nomad" = "<ABSOLUTE PATH TO YOUR GOPATH>/bin/"
  }
}
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests:

1. setup test environment
  ```sh
  nomad agent -dev -acl-enabled
  ```

2. obtain a management token
  ```sh
  nomad acl bootstrap
  ```

4. set nomad agent address (if differs from `http://localhost:4646`) and token secret ID and run tests
  ```sh
  NOMAD_TOKEN=<output of nomad acl bootstrap's Secret ID> NOMAD_ADDR=http://<host>:<port> make testacc
  ```

Acceptance tests expect fresh instance of nomad agent, so all steps must be
performed every time tests are executed.

*Note:* Acceptance tests create real resources, and may cost money to run.

[tf_docs_dev_overrides]: https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers
