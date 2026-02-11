package aws

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type sdkEC2Client struct {
	client *ec2.Client
}

func NewSDKEC2Client(configuration awssdk.Config) EC2Client {
	return &sdkEC2Client{client: ec2.NewFromConfig(configuration)}
}

func (e *sdkEC2Client) FindDefaultVPC(context context.Context) (string, error) {
	result, err := e.client.DescribeVpcs(context, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{
				Name:   awssdk.String("is-default"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("DescribeVpcs failed: %w", err)
	}
	if len(result.Vpcs) == 0 {
		return "", fmt.Errorf("no default VPC found")
	}
	return *result.Vpcs[0].VpcId, nil
}

func (e *sdkEC2Client) FindPublicSubnet(context context.Context, vpcID string) (string, error) {
	result, err := e.client.DescribeSubnets(context, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{
				Name:   awssdk.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("DescribeSubnets failed: %w", err)
	}
	if len(result.Subnets) == 0 {
		return "", fmt.Errorf("no subnets found in VPC %s", vpcID)
	}

	for _, subnet := range result.Subnets {
		if subnet.MapPublicIpOnLaunch != nil && *subnet.MapPublicIpOnLaunch {
			return *subnet.SubnetId, nil
		}
	}

	return *result.Subnets[0].SubnetId, nil
}

func (e *sdkEC2Client) CreateNetworkStack(context context.Context) (*NetworkStack, error) {
	stack := &NetworkStack{}

	vpcOutput, err := e.client.CreateVpc(context, &ec2.CreateVpcInput{
		CidrBlock: awssdk.String("10.0.0.0/16"),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeVpc,
				Tags: []ec2types.Tag{
					{Key: awssdk.String("Name"), Value: awssdk.String("goloo-vpc")},
					{Key: awssdk.String("ManagedBy"), Value: awssdk.String("goloo")},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("CreateVpc failed: %w", err)
	}
	stack.VpcID = *vpcOutput.Vpc.VpcId

	vpcWaiter := ec2.NewVpcAvailableWaiter(e.client)
	err = vpcWaiter.Wait(context, &ec2.DescribeVpcsInput{
		VpcIds: []string{stack.VpcID},
	}, 2*time.Minute)
	if err != nil {
		return stack, fmt.Errorf("VPC not available: %w", err)
	}

	_, err = e.client.ModifyVpcAttribute(context, &ec2.ModifyVpcAttributeInput{
		VpcId:              &stack.VpcID,
		EnableDnsHostnames: &ec2types.AttributeBooleanValue{Value: awssdk.Bool(true)},
	})
	if err != nil {
		return stack, fmt.Errorf("ModifyVpcAttribute (DNS hostnames) failed: %w", err)
	}

	igwOutput, err := e.client.CreateInternetGateway(context, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInternetGateway,
				Tags: []ec2types.Tag{
					{Key: awssdk.String("Name"), Value: awssdk.String("goloo-igw")},
					{Key: awssdk.String("ManagedBy"), Value: awssdk.String("goloo")},
				},
			},
		},
	})
	if err != nil {
		return stack, fmt.Errorf("CreateInternetGateway failed: %w", err)
	}
	stack.InternetGatewayID = *igwOutput.InternetGateway.InternetGatewayId

	_, err = e.client.AttachInternetGateway(context, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: &stack.InternetGatewayID,
		VpcId:             &stack.VpcID,
	})
	if err != nil {
		return stack, fmt.Errorf("AttachInternetGateway failed: %w", err)
	}

	azOutput, err := e.client.DescribeAvailabilityZones(context, &ec2.DescribeAvailabilityZonesInput{
		Filters: []ec2types.Filter{
			{
				Name:   awssdk.String("state"),
				Values: []string{"available"},
			},
		},
	})
	if err != nil {
		return stack, fmt.Errorf("DescribeAvailabilityZones failed: %w", err)
	}
	if len(azOutput.AvailabilityZones) == 0 {
		return stack, fmt.Errorf("no availability zones found")
	}
	availabilityZone := *azOutput.AvailabilityZones[0].ZoneName

	subnetOutput, err := e.client.CreateSubnet(context, &ec2.CreateSubnetInput{
		VpcId:            &stack.VpcID,
		CidrBlock:        awssdk.String("10.0.1.0/24"),
		AvailabilityZone: &availabilityZone,
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSubnet,
				Tags: []ec2types.Tag{
					{Key: awssdk.String("Name"), Value: awssdk.String("goloo-public-subnet")},
					{Key: awssdk.String("ManagedBy"), Value: awssdk.String("goloo")},
				},
			},
		},
	})
	if err != nil {
		return stack, fmt.Errorf("CreateSubnet failed: %w", err)
	}
	stack.SubnetID = *subnetOutput.Subnet.SubnetId

	_, err = e.client.ModifySubnetAttribute(context, &ec2.ModifySubnetAttributeInput{
		SubnetId:            &stack.SubnetID,
		MapPublicIpOnLaunch: &ec2types.AttributeBooleanValue{Value: awssdk.Bool(true)},
	})
	if err != nil {
		return stack, fmt.Errorf("ModifySubnetAttribute (auto-assign public IP) failed: %w", err)
	}

	rtOutput, err := e.client.CreateRouteTable(context, &ec2.CreateRouteTableInput{
		VpcId: &stack.VpcID,
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeRouteTable,
				Tags: []ec2types.Tag{
					{Key: awssdk.String("Name"), Value: awssdk.String("goloo-public-rt")},
					{Key: awssdk.String("ManagedBy"), Value: awssdk.String("goloo")},
				},
			},
		},
	})
	if err != nil {
		return stack, fmt.Errorf("CreateRouteTable failed: %w", err)
	}
	stack.RouteTableID = *rtOutput.RouteTable.RouteTableId

	_, err = e.client.CreateRoute(context, &ec2.CreateRouteInput{
		RouteTableId:         &stack.RouteTableID,
		DestinationCidrBlock: awssdk.String("0.0.0.0/0"),
		GatewayId:            &stack.InternetGatewayID,
	})
	if err != nil {
		return stack, fmt.Errorf("CreateRoute (default route) failed: %w", err)
	}

	associationOutput, err := e.client.AssociateRouteTable(context, &ec2.AssociateRouteTableInput{
		RouteTableId: &stack.RouteTableID,
		SubnetId:     &stack.SubnetID,
	})
	if err != nil {
		return stack, fmt.Errorf("AssociateRouteTable failed: %w", err)
	}
	stack.RouteTableAssociation = *associationOutput.AssociationId

	return stack, nil
}

func (e *sdkEC2Client) DeleteNetworkStack(context context.Context, stack *NetworkStack) error {
	if stack.RouteTableAssociation != "" {
		_, err := e.client.DisassociateRouteTable(context, &ec2.DisassociateRouteTableInput{
			AssociationId: &stack.RouteTableAssociation,
		})
		if err != nil {
			fmt.Printf("Warning: DisassociateRouteTable failed: %v\n", err)
		}
	}

	if stack.RouteTableID != "" {
		_, err := e.client.DeleteRouteTable(context, &ec2.DeleteRouteTableInput{
			RouteTableId: &stack.RouteTableID,
		})
		if err != nil {
			fmt.Printf("Warning: DeleteRouteTable failed: %v\n", err)
		}
	}

	if stack.SubnetID != "" {
		_, err := e.client.DeleteSubnet(context, &ec2.DeleteSubnetInput{
			SubnetId: &stack.SubnetID,
		})
		if err != nil {
			fmt.Printf("Warning: DeleteSubnet failed: %v\n", err)
		}
	}

	if stack.InternetGatewayID != "" && stack.VpcID != "" {
		_, err := e.client.DetachInternetGateway(context, &ec2.DetachInternetGatewayInput{
			InternetGatewayId: &stack.InternetGatewayID,
			VpcId:             &stack.VpcID,
		})
		if err != nil {
			fmt.Printf("Warning: DetachInternetGateway failed: %v\n", err)
		}

		_, err = e.client.DeleteInternetGateway(context, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: &stack.InternetGatewayID,
		})
		if err != nil {
			fmt.Printf("Warning: DeleteInternetGateway failed: %v\n", err)
		}
	}

	if stack.VpcID != "" {
		_, err := e.client.DeleteVpc(context, &ec2.DeleteVpcInput{
			VpcId: &stack.VpcID,
		})
		if err != nil {
			fmt.Printf("Warning: DeleteVpc failed: %v\n", err)
		}
	}

	return nil
}

func (e *sdkEC2Client) StopInstance(context context.Context, instanceID string) error {
	_, err := e.client.StopInstances(context, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("StopInstances %s failed: %w", instanceID, err)
	}
	return nil
}

func (e *sdkEC2Client) StartInstance(context context.Context, instanceID string) error {
	_, err := e.client.StartInstances(context, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("StartInstances %s failed: %w", instanceID, err)
	}
	return nil
}

func (e *sdkEC2Client) DescribeInstance(context context.Context, instanceID string) (string, string, error) {
	result, err := e.client.DescribeInstances(context, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return "", "", fmt.Errorf("DescribeInstances %s failed: %w", instanceID, err)
	}
	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return "", "", fmt.Errorf("instance %s not found", instanceID)
	}

	instance := result.Reservations[0].Instances[0]
	state := string(instance.State.Name)
	publicIP := ""
	if instance.PublicIpAddress != nil {
		publicIP = *instance.PublicIpAddress
	}
	return state, publicIP, nil
}
