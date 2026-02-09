package aws

import (
	"fmt"
	"strings"
)

const cloudFormationTemplate = `AWSTemplateFormatVersion: '2010-09-09'
Description: Goloo EC2 instance with SSH access

Parameters:
  ImageId:
    Type: String
  InstanceType:
    Type: String
    Default: t3.micro
  VpcId:
    Type: String
  SubnetId:
    Type: String

Resources:
  SSHSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Allow SSH/HTTP/HTTPS
      VpcId: !Ref VpcId
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 22
          ToPort: 22
          CidrIp: 0.0.0.0/0
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 0.0.0.0/0

  EC2Instance:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: !Ref InstanceType
      ImageId: !Ref ImageId
      NetworkInterfaces:
        - DeviceIndex: "0"
          SubnetId: !Ref SubnetId
          AssociatePublicIpAddress: true
          GroupSet:
            - !GetAtt SSHSecurityGroup.GroupId
      UserData: %s

Outputs:
  InstanceId:
    Value: !Ref EC2Instance
  PublicIP:
    Value: !GetAtt EC2Instance.PublicIp
  SecurityGroupId:
    Value: !Ref SSHSecurityGroup`

func GenerateTemplate(userData string) string {
	return fmt.Sprintf(cloudFormationTemplate, userData)
}

func TemplateContainsResource(template string, resourceName string) bool {
	return strings.Contains(template, resourceName+":")
}
