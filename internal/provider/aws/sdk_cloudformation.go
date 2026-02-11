package aws

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type sdkCloudFormationClient struct {
	client *cloudformation.Client
}

func NewSDKCloudFormationClient(configuration awssdk.Config) CloudFormationClient {
	return &sdkCloudFormationClient{client: cloudformation.NewFromConfig(configuration)}
}

func (c *sdkCloudFormationClient) CreateStack(context context.Context, name string, templateBody string, parameters map[string]string) (string, error) {
	cfnParameters := make([]types.Parameter, 0, len(parameters))
	for key, value := range parameters {
		cfnParameters = append(cfnParameters, types.Parameter{
			ParameterKey:   awssdk.String(key),
			ParameterValue: awssdk.String(value),
		})
	}

	result, err := c.client.CreateStack(context, &cloudformation.CreateStackInput{
		StackName:    &name,
		TemplateBody: &templateBody,
		Parameters:   cfnParameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
		},
	})
	if err != nil {
		return "", fmt.Errorf("CloudFormation CreateStack %s failed: %w", name, err)
	}
	return *result.StackId, nil
}

func (c *sdkCloudFormationClient) DeleteStack(context context.Context, name string) error {
	_, err := c.client.DeleteStack(context, &cloudformation.DeleteStackInput{
		StackName: &name,
	})
	if err != nil {
		return fmt.Errorf("CloudFormation DeleteStack %s failed: %w", name, err)
	}
	return nil
}

func (c *sdkCloudFormationClient) WaitForCreateComplete(context context.Context, name string) error {
	waiter := cloudformation.NewStackCreateCompleteWaiter(c.client)
	return waiter.Wait(context, &cloudformation.DescribeStacksInput{
		StackName: &name,
	}, 10*time.Minute)
}

func (c *sdkCloudFormationClient) WaitForDeleteComplete(context context.Context, name string) error {
	waiter := cloudformation.NewStackDeleteCompleteWaiter(c.client)
	return waiter.Wait(context, &cloudformation.DescribeStacksInput{
		StackName: &name,
	}, 10*time.Minute)
}

func (c *sdkCloudFormationClient) DescribeStack(context context.Context, name string) (*StackOutput, error) {
	result, err := c.client.DescribeStacks(context, &cloudformation.DescribeStacksInput{
		StackName: &name,
	})
	if err != nil {
		return nil, fmt.Errorf("CloudFormation DescribeStacks %s failed: %w", name, err)
	}
	if len(result.Stacks) == 0 {
		return nil, fmt.Errorf("stack %s not found", name)
	}

	output := &StackOutput{}
	for _, stackOutput := range result.Stacks[0].Outputs {
		switch *stackOutput.OutputKey {
		case "InstanceId":
			output.InstanceID = *stackOutput.OutputValue
		case "PublicIP":
			output.PublicIP = *stackOutput.OutputValue
		case "SecurityGroupId":
			output.SecurityGroupID = *stackOutput.OutputValue
		}
	}
	return output, nil
}
