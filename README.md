# Container Health Checker

Small utility to check if my containers are running

## Configuration

1. Copy `config.toml.dist` to `config.toml`
1. Set `interval` (time between retries), `timeout` (request timeout) and `retries`
1. Add an entry `[containers.<container handle>]` for each container. `port` is optional
