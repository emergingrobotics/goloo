package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type sdkSSMClient struct {
	client *ssm.Client
}

func NewSDKSSMClient(configuration aws.Config) SSMClient {
	return &sdkSSMClient{client: ssm.NewFromConfig(configuration)}
}

func (s *sdkSSMClient) GetParameter(context context.Context, path string) (string, error) {
	result, err := s.client.GetParameter(context, &ssm.GetParameterInput{
		Name: &path,
	})
	if err != nil {
		return "", fmt.Errorf("SSM GetParameter %s failed: %w", path, err)
	}
	return *result.Parameter.Value, nil
}
