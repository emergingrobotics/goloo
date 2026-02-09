package aws

import "fmt"

type NetworkStack struct {
	VpcID                 string
	SubnetID              string
	InternetGatewayID     string
	RouteTableID          string
	RouteTableAssociation string
}

func BuildStackName(vmName string) string {
	return fmt.Sprintf("goloo-%s", vmName)
}

func BuildNetworkStackName(vmName string) string {
	return fmt.Sprintf("goloo-%s-network", vmName)
}
