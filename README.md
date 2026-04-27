<div align="center" class="no-border">
  <h3>Automatic Vault Token Injector</h3>
  <a href="https://github.com/FairwindsOps/vault-token-injector">
    <img src="https://img.shields.io/github/v/release/FairwindsOps/vault-token-injector">
  </a>
  <a href="https://goreportcard.com/report/github.com/FairwindsOps/vault-token-injector">
    <img src="https://goreportcard.com/badge/github.com/FairwindsOps/vault-token-injector">
  </a>
  <a href="https://circleci.com/gh/FairwindsOps/vault-token-injector.svg">
    <img src="https://circleci.com/gh/FairwindsOps/vault-token-injector.svg?style=svg">
  </a>
  <a href="https://insights.fairwinds.com/gh/FairwindsOps/vault-token-injector">
    <img src="https://insights.fairwinds.com/v0/gh/FairwindsOps/vault-token-injector/badge.svg">
  </a>
</div>

# vault-token-injector

A loop to keep vault access tokens up-to-date in circleci and/or terraform cloud

Injects new tokens into circleci build environments or terraform cloud workspaces on startup and every 30 minutes. Also injects the `VAULT_ADDR` variable.

# Configuration

An example configuration file is present [here](example_config.yaml). Whatever circleci projects or terraform cloud workspaces are mentioned will update the given `token_variable` in the project workspace. The vault token for that project is created with the provided `vault_role` and/or `vault_policies`. In addition, the `vault_address` field is injected as the `VAULT_ADDR` environment variable.

## Token TTL and Refresh Interval

The default token TTL is 60 minutes, and the default refresh interval is 30 minutes. This allows some overlap intentionally. If you wish to customize these numbers, you can set the following in your configuration:

```
token_ttl: 10m
token_refresh_interval: 1m
```

## Logging

You can adjust the logging level with the `-vX` flag where X can be 1-10.
*WARNING* - Log level 10 will output secrets into the logs for debugging scenarios. Please do not do this in a production environment.

Note that the time intervals are golang time.Duration strings

## Future Planned Enhancements

* Staggered token injections
* Disable `VAULT_ADDR` injection
* Use Vault API instead of vault binary
* Prometheus endpoint to bubble up errors and successes

## Notice: Registry Migration and Immutable Images (v1.11.0 → v1.12.0)

Starting with **v1.12.0**:

- Images moved to `us-docker.pkg.dev/fairwinds-ops/oss/vault-token-injector`
- `quay.io/fairwinds/vault-token-injector` is deprecated

### Required action

```diff
- quay.io/fairwinds/vault-token-injector:<tag>
+ us-docker.pkg.dev/fairwinds-ops/oss/vault-token-injector:<tag>
```

---

## Immutable and signed images

* Images are now **signed**
* Tags are **immutable**
* No more floating tags:

  * `v1`
  * `v1.11`
  * `latest`

Use full version tags:

```
us-docker.pkg.dev/fairwinds-ops/oss/vault-token-injector:v<major>.<minor>.<patch>
```

Or pin by digest:

```
us-docker.pkg.dev/fairwinds-ops/oss/vault-token-injector@sha256:<digest>
```