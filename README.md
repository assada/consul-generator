# Consul File Generator

Create files from consul keys

### Usage
```bash
consul-generator \
 -from="${SERVICE_REGION}/apps/${SERVICE_NAME}/${SERVICE_ENV}"/keys/ \
 -to=./storage/keys/ \
 -host="localhost"
 -port="8500"
```

