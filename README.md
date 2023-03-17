# forta-bot-db

This allows a bot to store files over a HTTP request.  

## Limits:
- 10 MB file limit
- This uses AWS API Gateway which has certain timeouts

## Note:
One technique is to only use this to store a configuration file that includes credentials to other services. This allows you to give your bot a hosted database, access to cloud services, or api keys.  This BOT DB is not meant for high-volume chatty reads/writes, but rather for periodic blob storage.   If you need large/frequent access, consider S3 or DynamoDB directly from your bot. 

## Setup
1. Install Serverless: https://www.serverless.com/framework/docs/getting-started
2. Install AWS CLI: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html
3. Setup AWS Credentials for Deployment (~/.aws/credentials)
4. Modify the Makefile to use the apprpriate AWS Profile for your credentials (unless you use [default])

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
- `owner` any bot owned by the same owner as the requesting bot can see the object

## S3 Storage 

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

Make sure you have the right `--profile` referenced in Makefile's deploy target

```
make deploy
```
