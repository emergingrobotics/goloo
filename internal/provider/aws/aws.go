package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"

	"github.com/emergingrobotics/goloo/internal/config"
	"github.com/emergingrobotics/goloo/internal/provider"
)

var operatingSystemSSMPaths = map[string]string{
	"ubuntu-24.04":      "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
	"ubuntu-22.04":      "/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
	"ubuntu-20.04":      "/aws/service/canonical/ubuntu/server/20.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
	"amazon-linux-2023": "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64",
	"amazon-linux-2":    "/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2",
	"debian-12":         "/aws/service/debian/release/12/latest/amd64",
	"debian-11":         "/aws/service/debian/release/11/latest/amd64",
}

type Provider struct {
	Region         string
	CloudFormation CloudFormationClient
	EC2            EC2Client
	Route53        Route53Client
	SSM            SSMClient
}

func New(region string) *Provider {
	return &Provider{Region: region}
}

func NewWithClients(region string, cloudFormation CloudFormationClient, ec2 EC2Client, route53 Route53Client, ssm SSMClient) *Provider {
	return &Provider{
		Region:         region,
		CloudFormation: cloudFormation,
		EC2:            ec2,
		Route53:        route53,
		SSM:            ssm,
	}
}

func (p *Provider) Name() string {
	return "aws"
}

func LookupAMIPath(operatingSystem string) (string, error) {
	path, exists := operatingSystemSSMPaths[operatingSystem]
	if !exists {
		supported := SupportedOperatingSystems()
		return "", fmt.Errorf("unsupported OS %q: supported values are %v", operatingSystem, supported)
	}
	return path, nil
}

func SupportedOperatingSystems() []string {
	systems := make([]string, 0, len(operatingSystemSSMPaths))
	for name := range operatingSystemSSMPaths {
		systems = append(systems, name)
	}
	return systems
}

func (p *Provider) lookupAMI(context context.Context, operatingSystem string) (string, error) {
	path, err := LookupAMIPath(operatingSystem)
	if err != nil {
		return "", err
	}
	return p.SSM.GetParameter(context, path)
}

func (p *Provider) Create(context context.Context, configuration *config.Config, cloudInitPath string) error {
	if err := p.validateClients(); err != nil {
		return err
	}

	cloudInitContent, err := os.ReadFile(cloudInitPath)
	if err != nil {
		return fmt.Errorf("failed to read cloud-init file %s: %w", cloudInitPath, err)
	}
	userData := base64.StdEncoding.EncodeToString(cloudInitContent)

	osName := configuration.VM.OS
	if osName == "" {
		osName = "ubuntu-24.04"
	}
	amiID, err := p.lookupAMI(context, osName)
	if err != nil {
		return fmt.Errorf("AMI lookup failed: %w", err)
	}
	configuration.VM.AMIID = amiID

	vpcID, subnetID, err := p.discoverOrCreateNetwork(context, configuration)
	if err != nil {
		return fmt.Errorf("network setup failed: %w", err)
	}
	configuration.VM.VpcID = vpcID
	configuration.VM.SubnetID = subnetID

	template := GenerateTemplate(userData)
	stackName := BuildStackName(configuration.VM.Name)

	parameters := map[string]string{
		"ImageId":      amiID,
		"InstanceType": configuration.VM.InstanceType,
		"VpcId":        vpcID,
		"SubnetId":     subnetID,
	}

	stackID, err := p.CloudFormation.CreateStack(context, stackName, template, parameters)
	if err != nil {
		return fmt.Errorf("CloudFormation stack creation failed: %w", err)
	}
	configuration.VM.StackID = stackID
	configuration.VM.StackName = stackName

	if err := p.CloudFormation.WaitForCreateComplete(context, stackName); err != nil {
		return fmt.Errorf("CloudFormation stack failed to create: %w", err)
	}

	outputs, err := p.CloudFormation.DescribeStack(context, stackName)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}
	configuration.VM.InstanceID = outputs.InstanceID
	configuration.VM.PublicIP = outputs.PublicIP
	configuration.VM.SecurityGroup = outputs.SecurityGroupID

	if configuration.DNS != nil && configuration.DNS.Domain != "" {
		if err := p.createDNSRecords(context, configuration); err != nil {
			return fmt.Errorf("DNS record creation failed: %w", err)
		}
	}

	return nil
}

func (p *Provider) Delete(context context.Context, configuration *config.Config) error {
	if err := p.validateClients(); err != nil {
		return err
	}

	if configuration.DNS != nil && configuration.DNS.ZoneID != "" && configuration.VM.PublicIP != "" {
		if err := p.deleteDNSRecords(context, configuration); err != nil {
			return fmt.Errorf("DNS record deletion failed: %w", err)
		}
	}

	if configuration.VM.StackName != "" {
		if err := p.CloudFormation.DeleteStack(context, configuration.VM.StackName); err != nil {
			return fmt.Errorf("CloudFormation stack deletion failed: %w", err)
		}
		if err := p.CloudFormation.WaitForDeleteComplete(context, configuration.VM.StackName); err != nil {
			return fmt.Errorf("CloudFormation stack failed to delete: %w", err)
		}
	}

	if configuration.VM.CreatedVPC {
		networkStack := &NetworkStack{
			VpcID:                 configuration.VM.VpcID,
			SubnetID:              configuration.VM.SubnetID,
			InternetGatewayID:     configuration.VM.InternetGatewayID,
			RouteTableID:          configuration.VM.RouteTableID,
			RouteTableAssociation: configuration.VM.RouteTableAssociation,
		}
		if err := p.EC2.DeleteNetworkStack(context, networkStack); err != nil {
			return fmt.Errorf("network cleanup failed: %w", err)
		}
	}

	configuration.VM.InstanceID = ""
	configuration.VM.PublicIP = ""
	configuration.VM.StackID = ""
	configuration.VM.StackName = ""
	configuration.VM.SecurityGroup = ""
	configuration.VM.AMIID = ""
	configuration.VM.CreatedVPC = false
	configuration.VM.CreatedSubnet = false
	configuration.VM.InternetGatewayID = ""
	configuration.VM.RouteTableID = ""
	configuration.VM.RouteTableAssociation = ""

	if configuration.DNS != nil {
		configuration.DNS.ZoneID = ""
		configuration.DNS.FQDN = ""
		configuration.DNS.DNSRecords = nil
	}

	return nil
}

func (p *Provider) Status(context context.Context, configuration *config.Config) (*provider.VMStatus, error) {
	if err := p.validateClients(); err != nil {
		return nil, err
	}
	if configuration.VM.InstanceID == "" {
		return nil, fmt.Errorf("no instance ID: VM may not have been created with AWS")
	}

	state, publicIP, err := p.EC2.DescribeInstance(context, configuration.VM.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance %s: %w", configuration.VM.InstanceID, err)
	}

	return &provider.VMStatus{
		Name:     configuration.VM.Name,
		State:    state,
		IP:       publicIP,
		Provider: "aws",
	}, nil
}

func (p *Provider) List(_ context.Context) ([]provider.VMStatus, error) {
	return nil, fmt.Errorf("AWS list requires CloudFormation stack enumeration: use 'goloo status <name>' for individual VMs")
}

func (p *Provider) SSH(_ context.Context, configuration *config.Config) error {
	if configuration.VM.PublicIP == "" {
		return fmt.Errorf("no public IP: run 'goloo status %s' to check VM state", configuration.VM.Name)
	}
	username := "ubuntu"
	if len(configuration.VM.Users) > 0 {
		username = configuration.VM.Users[0].Username
	}
	command := exec.Command("ssh", username+"@"+configuration.VM.PublicIP)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func (p *Provider) Stop(context context.Context, configuration *config.Config) error {
	if err := p.validateClients(); err != nil {
		return err
	}
	if configuration.VM.InstanceID == "" {
		return fmt.Errorf("no instance ID: VM may not have been created with AWS")
	}
	return p.EC2.StopInstance(context, configuration.VM.InstanceID)
}

func (p *Provider) Start(context context.Context, configuration *config.Config) error {
	if err := p.validateClients(); err != nil {
		return err
	}
	if configuration.VM.InstanceID == "" {
		return fmt.Errorf("no instance ID: VM may not have been created with AWS")
	}
	return p.EC2.StartInstance(context, configuration.VM.InstanceID)
}

func (p *Provider) discoverOrCreateNetwork(context context.Context, configuration *config.Config) (string, string, error) {
	if configuration.VM.VpcID != "" && configuration.VM.SubnetID != "" {
		return configuration.VM.VpcID, configuration.VM.SubnetID, nil
	}

	vpcID, err := p.EC2.FindDefaultVPC(context)
	if err != nil {
		networkStack, createErr := p.EC2.CreateNetworkStack(context)
		if createErr != nil {
			return "", "", fmt.Errorf("no VPC found and failed to create one: %w", createErr)
		}
		configuration.VM.CreatedVPC = true
		configuration.VM.CreatedSubnet = true
		configuration.VM.InternetGatewayID = networkStack.InternetGatewayID
		configuration.VM.RouteTableID = networkStack.RouteTableID
		configuration.VM.RouteTableAssociation = networkStack.RouteTableAssociation
		return networkStack.VpcID, networkStack.SubnetID, nil
	}

	subnetID, err := p.EC2.FindPublicSubnet(context, vpcID)
	if err != nil {
		return "", "", fmt.Errorf("VPC %s found but no public subnet available: %w", vpcID, err)
	}

	return vpcID, subnetID, nil
}

func (p *Provider) validateClients() error {
	if p.CloudFormation == nil || p.EC2 == nil || p.SSM == nil {
		return fmt.Errorf("AWS provider not initialized: configure credentials with 'aws configure'")
	}
	return nil
}
