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

An example configuration file is present [here](example_config.yaml). Whatever circleci projects or terraform cloud workspaces are mentioned will update the given `token_variable` in the project workspace. The vault token for that project is created with the provided `vault_role`. In addition, the `vault_address` field is injected as the `VAULT_ADDR` environment variable.

## Future Planned Enhancements

* Customizable Timing (not hard-coded to 30m)
* Staggered token injections
* Disable `VAULT_ADDR` injection
* Use Vault API instead of vault binary
* Prometheus endpoint to bubble up errors and successes
