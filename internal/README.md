# Framework Provider And Mux Architecture

This directory contains the Terraform Plugin Framework provider and the mux
layer used to combine it with the existing SDKv2 Nomad provider.

## Why This Exists

The provider under [nomad/provider.go](../nomad/provider.go) is still the main
implementation for resources and data sources. Newer Terraform capabilities,
such as ephemeral resources, are implemented with the Terraform Plugin
Framework.

Rather than migrating the entire provider in one step, this repository uses the
Terraform plugin mux pattern to expose both implementations behind a single
`nomad` provider.

## Directory Layout

- [mux](./mux/mux.go): Composes the SDKv2 provider and the framework provider into one
  protocol v6 provider server.
- [framework/provider](./framework/provider/provider.go): Minimal framework provider used to
  expose framework-only capabilities.
- [framework/provider/\<sub-section\>](./framework/provider/): Framework provider
  subpackages for domain-specific framework-managed capabilities.

## Request Flow

At a high level, the provider starts in [main.go](../main.go):

1. `main` calls `mux.MuxServer(...)`.
2. `MuxServer` constructs the legacy SDKv2 provider with `nomad.Provider()`.
3. The SDKv2 gRPC provider is upgraded to protocol v6 using `tf5to6server`.
4. A framework provider is constructed with `framework.New(sdkProvider.Meta)`.
5. `tf6muxserver.NewMuxServer(...)` combines both provider servers.
6. Terraform interacts with the combined provider as a single `nomad` provider.

This keeps the existing SDKv2 surface area intact while allowing framework-only
features to be added incrementally.

## Why The Framework Provider Accepts `sdkProvider.Meta`

The framework provider is created with a callback to the SDKv2 provider's
`Meta` function:

```go
framework.New(sdkProvider.Meta)
```

This does not pass the full SDKv2 provider definition. It passes a function
that can return the configured SDKv2 provider metadata later.

For this provider, the configured SDKv2 metadata is `nomad.ProviderConfig`,
which contains:

- a configured `*api.Client`
- the resolved `*api.Config`

The callback is passed instead of eagerly resolving metadata so the framework
side can access the final configured SDKv2 client after provider configuration
has completed.

## Provider Data Bridge

The framework provider's `Configure` method passes the SDKv2 metadata callback
through `ResourceData`, `DataSourceData`, and `EphemeralResourceData`.

That allows framework-managed components to receive the callback as
`req.ProviderData` in their own `Configure` methods.

For the current ephemeral resource:

1. the framework provider stores `sdkProvider.Meta`
2. framework `Configure` assigns it to `resp.EphemeralResourceData`
3. the ephemeral resource receives it as `req.ProviderData`
4. the ephemeral resource calls the callback to get `nomad.ProviderConfig`
5. it uses `ProviderConfig.Client()` to access the existing configured Nomad API
   client

This avoids duplicating Nomad client setup logic in the framework provider.

## What Lives On The Framework Side

The framework provider is intended to hold capabilities that are better served
by the Terraform Plugin Framework than by extending the SDKv2 provider.

More generally, this side of the provider is the place for:

- framework-native capabilities such as ephemeral resources
- newly added resources and data sources implemented with the framework
- components that need framework or provider-protocol features not available in
  the SDKv2 provider
- code that can benefit from sharing configured provider access through the
  SDKv2 metadata bridge

## Testing Approach

Legacy SDKv2 resources and data sources continue to use the existing provider
implementation and test patterns.

Framework and mux behavior is tested through protocol v6 provider factories,
since the framework provider and muxed provider are exposed as protocol v6
provider servers rather than raw SDKv2 `*schema.Provider` values.

## Long-Term Direction

The goal of this structure is incremental adoption:

- keep stable SDKv2 behavior in place
- add framework-only functionality where needed
- avoid a large one-shot migration
- keep future migration options open if more framework features are added later

The expected direction from here is a gradual migration toward the framework
provider:

- existing SDKv2 resources and data sources can be migrated incrementally over
  time
- new resources and data sources should be added on the framework side rather
  than extending the SDKv2 provider further
- mux remains the compatibility layer while both implementations coexist
