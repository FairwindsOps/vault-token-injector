# vault-token-injector
A loop to keep vault access tokens up to date in circleci. 
This will minimize the lift during key rotations in our build environments by removing all but the short lived vault tokens from CircleCi.

Injects new tokens in to circleci on startup and every 30 minutes. 

Configure which repos are monitored and adjust their permission mapping with a configfile.

Future: 
Configure for TF Cloud.
