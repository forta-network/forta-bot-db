# forta-bot-db

This allows a bot to store files over a HTTP request.

## APIs

These are the following APIs
```
GET https://{host}/database/{key}
PUT https://{host}/database/{key}   (body = payload)
DELETE https://{host}/database/{key}
```

Files are stored in S3 under the following key format
```
{scannerId}/{botId}/{key}
```

## Authentication

Bots should use the JWT defined here:
https://docs.forta.network/en/latest/jwt-auth/

## Deploy

```
make deploy-research
```
