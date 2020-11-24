# Running accpetance test suite as a Nomad Job

This folder contains some WIP artifacts for running the provider acceptance
test suite as a Nomad dispatch job.

## Build the image

```shell-session
$ cd docker
$ docker build -t tftest:0.0.1 .
Sending build context to Docker daemon  3.072kB
Step 1/7 : FROM golang:1.15.5
 ---> 6d8772fbd285
Step 2/7 : RUN apt-get update &&     apt-get install -y         build-essential         httpie         jq
 ---> Running in b3e46685c3fe
...
 ---> 942d0d9a9ddc
Successfully built 942d0d9a9ddc
Successfully tagged tftest:0.0.1
$ cd ..
```

## Start a local Nomad cluster

In order for the jobs to be properly started, the Nomad client needs special
Docker capabilities, so start a Nomad agent with this command:

```shell-session
$ nomad agent -dev -config ./config-docker.hcl
==> Loaded configuration from config-docker.hcl
==> Starting Nomad agent...
...
```

## Register batch job

```shell-session
$ nomad run tftest.nomad
Job registration successful
```

## Dispatch job

```shell-session
$ nomad job dispatch \
    -meta provider_branch=master \
    -meta terraform_version=0.14.0-rc1 \
    -meta nomad_version=1.0.0-beta3+ent \
    tftest
Dispatched Job ID = tftest/dispatch-1606180820-183263b7
Evaluation ID     = e24afa98

==> Monitoring evaluation "e24afa98"
    Evaluation triggered by job "tftest/dispatch-1606180820-183263b7"
    Allocation "4aad3a5c" created: node "a33c3b32", group "foo"
    Evaluation status changed: "pending" -> "complete"
==> Evaluation "e24afa98" finished with status "complete"
```
