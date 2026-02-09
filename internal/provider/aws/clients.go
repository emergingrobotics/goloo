package aws

import "context"

type CloudFormationClient interface {
	CreateStack(context context.Context, name string, templateBody string, parameters map[string]string) (string, error)
	DeleteStack(context context.Context, name string) error
	WaitForCreateComplete(context context.Context, name string) error
	WaitForDeleteComplete(context context.Context, name string) error
	DescribeStack(context context.Context, name string) (*StackOutput, error)
}

type EC2Client interface {
	FindDefaultVPC(context context.Context) (string, error)
	FindPublicSubnet(context context.Context, vpcID string) (string, error)
	CreateNetworkStack(context context.Context) (*NetworkStack, error)
	DeleteNetworkStack(context context.Context, stack *NetworkStack) error
	StopInstance(context context.Context, instanceID string) error
	StartInstance(context context.Context, instanceID string) error
	DescribeInstance(context context.Context, instanceID string) (string, string, error)
}

type Route53Client interface {
	FindZoneID(context context.Context, domain string) (string, error)
	UpsertARecord(context context.Context, zoneID string, name string, ip string, ttl int) error
	DeleteARecord(context context.Context, zoneID string, name string, ip string, ttl int) error
}

type SSMClient interface {
	GetParameter(context context.Context, path string) (string, error)
}

type StackOutput struct {
	InstanceID      string
	PublicIP        string
	SecurityGroupID string
}
