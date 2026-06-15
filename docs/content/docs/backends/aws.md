---
title: AWS Secrets Manager
weight: 2
---

The `aws` backend stores namespaces as secrets in AWS Secrets Manager. Each namespace maps to one AWS secret; individual keys are stored as JSON fields within it.

## Global config

```yaml
backends:
  aws:
    region: us-east-1         # required
    # profile: my-aws-profile # optional, uses default credential chain if omitted
```

AWS credentials are resolved via the standard SDK credential chain: environment variables, `~/.aws/credentials`, instance metadata, etc. This means you can store AWS credentials in the OS keychain and chain the AWS backend to retrieve secrets from AWS.

## `.envoke` usage

```yaml
namespaces:
- name: aws-dev
  backend: aws
```

The secret name in AWS Secrets Manager is the namespace name (`aws-dev` in this example).

## IAM permissions

The IAM principal needs at minimum:

```json
{
  "Effect": "Allow",
  "Action": [
    "secretsmanager:GetSecretValue",
    "secretsmanager:DescribeSecret"
  ],
  "Resource": "arn:aws:secretsmanager:*:*:secret:*"
}
```

Scope the `Resource` to your specific secrets in production.
