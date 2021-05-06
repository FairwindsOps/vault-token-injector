# vault-token-injector
A loop to keep vault access tokens up to date in circleci. 
This will minimize the lift during key rotations in our build environments by removing all but the short lived vault tokens from CircleCi.

Injects new tokens in to circleci on startup and every 30 minutes.

# Configuration

An example configuration file is present [here](example_config.yaml). Whatever circleci projects are mentioned will update the given `env_variable` in the project workspace. The vault token for that project is created with the provided `vault_role`.

Future: 
Configure for TF Cloud.
