# LEGO Charm Terraform Provider 

This repository is a Terraform provider intended for use with Juju and the [httprequest-lego-provider charm from Canonical](https://charmhub.io/httprequest-lego-provider).

This repository is based on the [terraform provider template repository provided by Hashicorp](https://github.com/hashicorp/terraform-provider-scaffolding).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Provide the `address` where the httprequest provider is being served, and `username` + `password` credentials for a superuser of the httprequest-lego-provider. A superuser can be created using [a Juju action on the charm](https://charmhub.io/httprequest-lego-provider/actions#create-superuser).

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
