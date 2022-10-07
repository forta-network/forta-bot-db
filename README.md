# forta-bot-db

This allows a bot to store files over a HTTP request.

## APIs

These are the following APIs
```
GET https://{host}/database/{scope}/{key}
PUT https://{host}/database/{scope}/{key}   (body = payload)
DELETE https://{host}/database/{scope}/{key}
```

Valid scopes
- `bot` means the bot can see the object regardless of scanner
- `scanner` means only one scanner can see this object

Files are stored in S3 under the following key format

For `scanner` scope
```
{scannerId}/{botId}/{key}
```

For `bot` scope
```
{botId}/{key}
```

## Authentication

Bots should use the JWT defined here:
https://docs.forta.network/en/latest/jwt-auth/

## Deploy

```
make deploy-research
```
