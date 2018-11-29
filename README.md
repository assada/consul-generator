# Consul File Generator
[![Build Status](https://travis-ci.org/Assada/consul-generator.svg?branch=master)](https://travis-ci.org/Assada/consul-generator)

Create files from consul keys

### Usage
```bash
consul-generator \
 -from="${SERVICE_REGION}/apps/${SERVICE_NAME}/${SERVICE_ENV}/keys/" \
 -to="./storage/keys/"
 -consul-addr="localhost:8500"
```
