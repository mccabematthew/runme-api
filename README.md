# RunMe API

Go Lambda backend for RunMe. Handles run logging and retrieval.

## Stack

- Go + AWS Lambda
- DynamoDB
- API Gateway (HTTP API)
- Deployed via S3 + Lambda

## Local Development
```bash
go run main.go
```

## Environment Variables

Create a `.env` file in the root, reference `.env.example` for required variables.

## Deployment

Deployed via GitHub Actions on merge to `main`. Builds Go binary, zips, and uploads to Lambda.