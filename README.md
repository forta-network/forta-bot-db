# forta-bot-db

This allows a bot to store files over a HTTP request by validating the Scanner JWT.

## Authentication

Bots should use the JWT defined here, and this API will validate that.
https://docs.forta.network/en/latest/jwt-auth/

## Limits:
- 10 MB file limit
- This uses AWS API Gateway which has certain timeouts

This BOT DB is not meant for high-volume chatty reads/writes, but rather for periodic blob storage.   If you need large/frequent access, consider S3 or DynamoDB directly from your bot (using the secrets storage technique)

## Technique: Secrets Storage
One technique is to only use this to store a configuration file that includes credentials to other services. This allows you to give your bot a hosted database, access to cloud services, or api keys.  

If you with to do this, you'll need to put your secret directly in S3 according to the key pattern described below.

**Do not put highly sensitive secrets here.  It is always possible for your secrets to leak because Scanners are inherently untrusted environments**

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

**All botIds, scannerIds, and owner addresses are Lowercase.**

For `scanner` scope
```
Pattern
{scannerId}/{botId}/{key}

Example
0xabcdefabcdefabcdefabcdefabcdefabcdef/0xabcdefabcdefabcdefabcdefabcdefabcdef0xabcdefabcdefabcdefabcdefabcdefabcdef/cache.json
```

For `bot` scope
```
Pattern
{botId}/{key}

Example
0xabcdefabcdefabcdefabcdefabcdefabcdef0xabcdefabcdefabcdefabcdefabcdefabcdef/object.gz
```

For `owner` scope
```
Pattern
owner/{owner-address}/{key}

Example
owner/0xabcdefabcdefabcdefabcdefabcdefabcdef/secrets.json
```

## Configuration

In the `serverless.yml` there is a reference to an AWS SSM parameter POLYGON_JSON_RPC.  You can set this to any polygon rpc you wish.  If you don't wish to use SSM, replace this value with whatever polygon json-rpc provider you wish to use.  If you remove this ENV reference entirely, the system will fall back to https://polygon-rpc.com, which can be rate limited.

## Deploy

Make sure you have the right `--profile` referenced in Makefile's deploy target and in the serverless.yml.

```
make deploy
```
