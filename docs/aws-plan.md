# Plan: Implement AWS SDK Client Wrappers

## Context

`goloo create --aws` fails with "AWS provider not initialized" because `New(region)` returns a `Provider` with nil SDK clients. The interfaces exist (`CloudFormationClient`, `EC2Client`, `Route53Client`, `SSMClient` in `clients.go`) and tests use fakes, but there are no real AWS SDK implementations. The working reference implementation at `./aws-ec2/main.go` has all the SDK v2 call patterns we need.

## Constraint

`New(region)` must remain unchanged — 4 existing tests call it directly and depend on it succeeding without AWS credentials. Add `NewWithSDK(region) (*Provider, error)` alongside it.

## Files Created

### 1. `internal/provider/aws/sdk_ssm.go` — SSMClient implementation

Implements `SSMClient` interface using `ssm.NewFromConfig(awsCfg)`.

- `GetParameter(ctx, path)` -> `ssmClient.GetParameter(ctx, &ssm.GetParameterInput{Name: &path})` -> `*result.Parameter.Value`

### 2. `internal/provider/aws/sdk_cloudformation.go` — CloudFormationClient implementation

Implements `CloudFormationClient` interface using `cloudformation.NewFromConfig(awsCfg)`.

- `CreateStack(ctx, name, templateBody, parameters)` -> converts `parameters` map to `[]types.Parameter`, calls `cfClient.CreateStack` -> returns `*result.StackId`
- `DeleteStack(ctx, name)` -> `cfClient.DeleteStack`
- `WaitForCreateComplete(ctx, name)` -> `cloudformation.NewStackCreateCompleteWaiter(cfClient).Wait(ctx, input, 10*time.Minute)`
- `WaitForDeleteComplete(ctx, name)` -> `cloudformation.NewStackDeleteCompleteWaiter(cfClient).Wait(ctx, input, 10*time.Minute)`
- `DescribeStack(ctx, name)` -> `cfClient.DescribeStacks` -> extract `InstanceId`, `PublicIP`, `SecurityGroupId` from `Stacks[0].Outputs` -> return `*StackOutput`

### 3. `internal/provider/aws/sdk_ec2.go` — EC2Client implementation

Implements `EC2Client` interface using `ec2.NewFromConfig(awsCfg)`. This is the largest file.

- `FindDefaultVPC(ctx)` -> `DescribeVpcs` with filter `is-default=true` -> `*result.Vpcs[0].VpcId`
- `FindPublicSubnet(ctx, vpcID)` -> `DescribeSubnets` with filter `vpc-id` -> prefer subnet with `MapPublicIpOnLaunch=true`, fall back to first
- `CreateNetworkStack(ctx)` -> sequence: CreateVpc -> wait VpcAvailable -> ModifyVpcAttribute (DNS hostnames) -> CreateInternetGateway -> AttachInternetGateway -> DescribeAvailabilityZones -> CreateSubnet -> ModifySubnetAttribute (auto-assign public IP) -> CreateRouteTable -> CreateRoute (0.0.0.0/0 -> IGW) -> AssociateRouteTable -> return `*NetworkStack`
- `DeleteNetworkStack(ctx, stack)` -> reverse: DisassociateRouteTable -> DeleteRouteTable -> DeleteSubnet -> DetachInternetGateway -> DeleteInternetGateway -> DeleteVpc
- `StopInstance(ctx, instanceID)` -> `ec2Client.StopInstances`
- `StartInstance(ctx, instanceID)` -> `ec2Client.StartInstances`
- `DescribeInstance(ctx, instanceID)` -> `ec2Client.DescribeInstances` -> return (state, publicIP, error)

### 4. `internal/provider/aws/sdk_route53.go` — Route53Client implementation

Implements `Route53Client` interface using `route53.NewFromConfig(awsCfg)`.

All domain names get a trailing dot appended before API calls (Route53 requirement).

- `FindZoneID(ctx, domain)` -> `r53Client.ListHostedZonesByName(ctx, &route53.ListHostedZonesByNameInput{DNSName: &domain})` -> match zone name -> strip `/hostedzone/` prefix from ID
- `UpsertARecord(ctx, zoneID, name, ip, ttl)` -> `ChangeResourceRecordSets` with action=UPSERT, type=A
- `DeleteARecord(ctx, zoneID, name, ip, ttl)` -> `ChangeResourceRecordSets` with action=DELETE, type=A
- `UpsertCNAMERecord(ctx, zoneID, name, target, ttl)` -> `ChangeResourceRecordSets` with action=UPSERT, type=CNAME
- `DeleteCNAMERecord(ctx, zoneID, name, target, ttl)` -> `ChangeResourceRecordSets` with action=DELETE, type=CNAME

## Files Modified

### 5. `internal/provider/aws/aws.go` — Added `NewWithSDK`

Constructor that loads real AWS config and creates SDK clients:

```go
func NewWithSDK(region string) (*Provider, error) {
    awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }
    return &Provider{
        Region:         region,
        CloudFormation: NewSDKCloudFormationClient(awsCfg),
        EC2:            NewSDKEC2Client(awsCfg),
        Route53:        NewSDKRoute53Client(awsCfg),
        SSM:            NewSDKSSMClient(awsCfg),
    }, nil
}
```

### 6. `cmd/goloo/main.go` — Uses `NewWithSDK`

Changed `getProvider` and `cmdDNSSwap` to call `awsprovider.NewWithSDK(region)` instead of `awsprovider.New(region)`.

### 7. `go.mod` / `go.sum` — AWS SDK v2 dependencies added

## Verification

1. `go build ./...` — compiles with no errors
2. `go test ./...` — all 167 existing tests pass (they use fakes, not real SDK)
3. `goloo create <name> --aws` — creates an EC2 instance (manual test with AWS credentials)
4. `goloo delete <name>` — deletes the instance and cleans up
