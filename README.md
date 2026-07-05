# Omni-AWS

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omni-aws/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omni-aws/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omni-aws/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omni-aws/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omni-aws/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omni-aws/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omni-aws
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omni-aws
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/omni-aws
 [viz-svg]: https://img.shields.io/badge/Go-visualizaton-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomni-aws
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omni-aws
 [repo-url]: https://github.com/plexusone/omni-aws
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omni-aws/blob/main/LICENSE

AWS provider packages for [PlexusOne](https://github.com/plexusone) libraries.

## Modules

This repository contains multiple Go modules for AWS integrations:

| Module | Description | Install |
|--------|-------------|---------|
| [`omnillm`](omnillm/) | AWS Bedrock provider for [omnillm-core](https://github.com/plexusone/omnillm-core) | `go get github.com/plexusone/omni-aws/omnillm` |
| [`omnistorage`](omnistorage/) | S3 backend for [omnistorage-core](https://github.com/plexusone/omnistorage-core) | `go get github.com/plexusone/omni-aws/omnistorage` |
| [`omnivault`](omnivault/) | AWS Secrets Manager & Parameter Store for [omnivault](https://github.com/plexusone/omnivault) | `go get github.com/plexusone/omni-aws/omnivault` |
| [`omnimemory`](omnimemory/) | DynamoDB provider for [omnimemory](https://github.com/plexusone/omnimemory) | `go get github.com/plexusone/omni-aws/omnimemory` |

## Quick Start

### OmniLLM - AWS Bedrock

```go
import (
    "github.com/plexusone/omni-aws/omnillm"
    "github.com/plexusone/omnillm-core"
)

// Create Bedrock provider
provider := omnillm.NewProvider("us-east-1")

// Use with OmniLLM
client := omnillm.NewClient(provider)
resp, err := client.CreateChatCompletion(ctx, omnillm.ChatCompletionRequest{
    Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
    Messages: []omnillm.Message{
        {Role: "user", Content: "Hello!"},
    },
})
```

See [omnillm/README.md](omnillm/README.md) for full documentation.

### OmniStorage - S3 Backend

```go
import (
    "github.com/plexusone/omni-aws/omnistorage/backend/s3"
)

// Create S3 backend
backend, err := s3.New(s3.Config{
    Bucket: "my-bucket",
    Region: "us-east-1",
})

// Write
w, _ := backend.NewWriter(ctx, "path/to/file.txt")
w.Write([]byte("hello"))
w.Close()

// Read
r, _ := backend.NewReader(ctx, "path/to/file.txt")
data, _ := io.ReadAll(r)
r.Close()
```

See [omnistorage/README.md](omnistorage/) for full documentation including S3-compatible services (R2, MinIO, Wasabi).

### OmniVault - AWS Secrets Manager & Parameter Store

```go
import (
    aws "github.com/plexusone/omni-aws/omnivault"
)

// Create Secrets Manager provider
provider, err := aws.NewSecretsManager(aws.Config{
    Region: "us-east-1",
})

// Get a secret
secret, err := provider.Get(ctx, "prod/database/credentials")
fmt.Println("Password:", secret.Value)
fmt.Println("Username:", secret.Fields["username"])

// Or use Parameter Store
ssmProvider, err := aws.NewParameterStore(aws.Config{
    Region: "us-east-1",
})
param, err := ssmProvider.Get(ctx, "/myapp/prod/api-key")
```

See [omnivault/README.md](omnivault/) for full documentation including IRSA, versioning, and rotation.

### OmniMemory - DynamoDB Provider

```go
import (
    "github.com/plexusone/omnimemory"
    "github.com/plexusone/omnimemory/core"
    _ "github.com/plexusone/omni-aws/omnimemory/dynamodb"
)

// Create DynamoDB-backed memory client
client, err := omnimemory.NewClient(core.ClientConfig{
    Providers: []core.ProviderConfig{
        {
            Name: core.ProviderNameAWSDynamoDB,
            Options: map[string]any{
                "table_name": "omnimemory",
                "region":     "us-east-1",
            },
        },
    },
})

// Add a memory
memory, err := client.Add(ctx, &core.AddRequest{
    Context: core.Context{
        TenantID:  "tenant-123",
        SubjectID: "user-456",
    },
    Type:    core.MemoryTypeObservation,
    Content: "User prefers dark mode interfaces",
})

// Search memories
results, err := client.Search(ctx, &core.SearchRequest{
    Context: core.Context{
        TenantID:  "tenant-123",
        SubjectID: "user-456",
    },
    Query: "interface preferences",
    Limit: 10,
})
```

See [omnimemory/README.md](omnimemory/) for full documentation including local development, TTL, and multi-tenancy.

## License

MIT
