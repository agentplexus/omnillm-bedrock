module github.com/plexusone/omni-aws

go 1.26.4

require (
	// omnistorage dependencies
	github.com/aws/aws-sdk-go-v2 v1.43.0
	// omnillm dependencies
	github.com/aws/aws-sdk-go-v2/config v1.32.31
	github.com/aws/aws-sdk-go-v2/credentials v1.19.30
	// omnimemory dependencies
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.54
	github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager v0.3.5
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.56.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.62.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.106.0
	// omnivault dependencies
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.44.0
	github.com/aws/aws-sdk-go-v2/service/ssm v1.73.0
	github.com/aws/smithy-go v1.27.4
	github.com/google/uuid v1.6.0
	github.com/grokify/mogo v0.74.6
	github.com/plexusone/omnidevx-core v0.3.0
	github.com/plexusone/omnillm-core v0.18.0
	github.com/plexusone/omnimemory v0.1.0
	github.com/plexusone/omnistorage-core v0.5.0
	github.com/plexusone/omnivault v0.5.0
	modernc.org/sqlite v1.54.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.32 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.36.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.32 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/grokify/oscompat v0.5.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.47.0 // indirect
	modernc.org/libc v1.74.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
