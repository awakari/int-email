# About

Email event source implementation. Basically, an SMTP server.

# Conversion Schema

TODO

# Compatibility

TODO

# Other

## Build locally

## K8s secrets

```shell
kubectl create secret generic gcp-dns-secret --from-file=key.json
```

```shell
kubectl create secret generic int-email \
  --from-literal=rcptsPublish=rcpt1,rcpt2 \
  --from-literal=rcptsInternal=rcpt3,rcpt4,...
```
