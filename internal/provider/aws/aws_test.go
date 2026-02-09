package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/emergingrobotics/goloo/internal/config"
)

type fakeCloudFormation struct {
	stackOutput     *StackOutput
	stackID         string
	createError     error
	deleteError     error
	waitCreateError error
	waitDeleteError error
	describeError   error
	createdStacks   []string
	deletedStacks   []string
}

func (f *fakeCloudFormation) CreateStack(_ context.Context, name string, _ string, _ map[string]string) (string, error) {
	f.createdStacks = append(f.createdStacks, name)
	if f.createError != nil {
		return "", f.createError
	}
	return f.stackID, nil
}

func (f *fakeCloudFormation) DeleteStack(_ context.Context, name string) error {
	f.deletedStacks = append(f.deletedStacks, name)
	return f.deleteError
}

func (f *fakeCloudFormation) WaitForCreateComplete(_ context.Context, _ string) error {
	return f.waitCreateError
}

func (f *fakeCloudFormation) WaitForDeleteComplete(_ context.Context, _ string) error {
	return f.waitDeleteError
}

func (f *fakeCloudFormation) DescribeStack(_ context.Context, _ string) (*StackOutput, error) {
	if f.describeError != nil {
		return nil, f.describeError
	}
	return f.stackOutput, nil
}

type fakeEC2 struct {
	defaultVPCID    string
	subnetID        string
	networkStack    *NetworkStack
	instanceState   string
	instanceIP      string
	findVPCError    error
	findSubnetError error
	createNetError  error
	deleteNetError  error
	stoppedInstances []string
	startedInstances []string
	deletedNetworks  []*NetworkStack
}

func (f *fakeEC2) FindDefaultVPC(_ context.Context) (string, error) {
	if f.findVPCError != nil {
		return "", f.findVPCError
	}
	return f.defaultVPCID, nil
}

func (f *fakeEC2) FindPublicSubnet(_ context.Context, _ string) (string, error) {
	if f.findSubnetError != nil {
		return "", f.findSubnetError
	}
	return f.subnetID, nil
}

func (f *fakeEC2) CreateNetworkStack(_ context.Context) (*NetworkStack, error) {
	if f.createNetError != nil {
		return nil, f.createNetError
	}
	return f.networkStack, nil
}

func (f *fakeEC2) DeleteNetworkStack(_ context.Context, stack *NetworkStack) error {
	f.deletedNetworks = append(f.deletedNetworks, stack)
	return f.deleteNetError
}

func (f *fakeEC2) StopInstance(_ context.Context, instanceID string) error {
	f.stoppedInstances = append(f.stoppedInstances, instanceID)
	return nil
}

func (f *fakeEC2) StartInstance(_ context.Context, instanceID string) error {
	f.startedInstances = append(f.startedInstances, instanceID)
	return nil
}

func (f *fakeEC2) DescribeInstance(_ context.Context, _ string) (string, string, error) {
	return f.instanceState, f.instanceIP, nil
}

type fakeRoute53 struct {
	zoneID         string
	findZoneError  error
	upsertedRecords []string
	deletedRecords  []string
	upsertError    error
	deleteError    error
}

func (f *fakeRoute53) FindZoneID(_ context.Context, _ string) (string, error) {
	if f.findZoneError != nil {
		return "", f.findZoneError
	}
	return f.zoneID, nil
}

func (f *fakeRoute53) UpsertARecord(_ context.Context, _ string, name string, ip string, _ int) error {
	f.upsertedRecords = append(f.upsertedRecords, name+"->"+ip)
	return f.upsertError
}

func (f *fakeRoute53) DeleteARecord(_ context.Context, _ string, name string, _ string, _ int) error {
	f.deletedRecords = append(f.deletedRecords, name)
	return f.deleteError
}

type fakeSSM struct {
	parameters map[string]string
	err        error
}

func (f *fakeSSM) GetParameter(_ context.Context, path string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	value, exists := f.parameters[path]
	if !exists {
		return "", fmt.Errorf("parameter %s not found", path)
	}
	return value, nil
}

func newFakeProvider() (*Provider, *fakeCloudFormation, *fakeEC2, *fakeRoute53, *fakeSSM) {
	cloudFormation := &fakeCloudFormation{
		stackID: "arn:aws:cloudformation:us-east-1:123456:stack/goloo-devbox/abc123",
		stackOutput: &StackOutput{
			InstanceID:      "i-0123456789abcdef0",
			PublicIP:        "54.1.2.3",
			SecurityGroupID: "sg-0123456789abcdef0",
		},
	}
	ec2 := &fakeEC2{
		defaultVPCID:  "vpc-abc123",
		subnetID:      "subnet-def456",
		instanceState: "running",
		instanceIP:    "54.1.2.3",
	}
	route53 := &fakeRoute53{
		zoneID: "Z1234567890",
	}
	ssm := &fakeSSM{
		parameters: map[string]string{
			"/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp2/ami-id": "ami-0123456789abcdef0",
		},
	}
	provider := NewWithClients("us-east-1", cloudFormation, ec2, route53, ssm)
	return provider, cloudFormation, ec2, route53, ssm
}

func createCloudInitFile(t *testing.T) string {
	t.Helper()
	content := "#cloud-config\npackages:\n  - git"
	path := filepath.Join(t.TempDir(), "cloud-init.yaml")
	os.WriteFile(path, []byte(content), 0644)
	return path
}

func TestProviderName(t *testing.T) {
	provider := New("us-east-1")
	if provider.Name() != "aws" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "aws")
	}
}

func TestNewSetsRegion(t *testing.T) {
	provider := New("eu-west-1")
	if provider.Region != "eu-west-1" {
		t.Errorf("Region = %q, want %q", provider.Region, "eu-west-1")
	}
}

func TestCreateWithDefaultVPC(t *testing.T) {
	provider, cloudFormation, _, _, _ := newFakeProvider()
	cloudInitPath := createCloudInitFile(t)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
			OS:           "ubuntu-24.04",
		},
	}

	err := provider.Create(context.Background(), configuration, cloudInitPath)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if configuration.VM.InstanceID != "i-0123456789abcdef0" {
		t.Errorf("InstanceID = %q, want %q", configuration.VM.InstanceID, "i-0123456789abcdef0")
	}
	if configuration.VM.PublicIP != "54.1.2.3" {
		t.Errorf("PublicIP = %q, want %q", configuration.VM.PublicIP, "54.1.2.3")
	}
	if configuration.VM.VpcID != "vpc-abc123" {
		t.Errorf("VpcID = %q, want %q", configuration.VM.VpcID, "vpc-abc123")
	}
	if configuration.VM.SubnetID != "subnet-def456" {
		t.Errorf("SubnetID = %q, want %q", configuration.VM.SubnetID, "subnet-def456")
	}
	if configuration.VM.AMIID != "ami-0123456789abcdef0" {
		t.Errorf("AMIID = %q, want %q", configuration.VM.AMIID, "ami-0123456789abcdef0")
	}
	if configuration.VM.StackName != "goloo-devbox" {
		t.Errorf("StackName = %q, want %q", configuration.VM.StackName, "goloo-devbox")
	}
	if len(cloudFormation.createdStacks) != 1 || cloudFormation.createdStacks[0] != "goloo-devbox" {
		t.Errorf("Expected stack 'goloo-devbox' to be created, got %v", cloudFormation.createdStacks)
	}
}

func TestCreateDefaultsToUbuntu2404(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()
	cloudInitPath := createCloudInitFile(t)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
		},
	}

	err := provider.Create(context.Background(), configuration, cloudInitPath)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if configuration.VM.AMIID != "ami-0123456789abcdef0" {
		t.Errorf("Should default to ubuntu-24.04 AMI, got %q", configuration.VM.AMIID)
	}
}

func TestCreateWithDNS(t *testing.T) {
	provider, _, _, route53, _ := newFakeProvider()
	cloudInitPath := createCloudInitFile(t)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
		},
		DNS: &config.DNSConfig{
			Hostname: "devbox",
			Domain:   "example.com",
			TTL:      60,
		},
	}

	err := provider.Create(context.Background(), configuration, cloudInitPath)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if configuration.DNS.ZoneID != "Z1234567890" {
		t.Errorf("ZoneID = %q, want %q", configuration.DNS.ZoneID, "Z1234567890")
	}
	if configuration.DNS.FQDN != "devbox.example.com" {
		t.Errorf("FQDN = %q, want %q", configuration.DNS.FQDN, "devbox.example.com")
	}
	if len(route53.upsertedRecords) != 1 {
		t.Fatalf("Expected 1 upserted record, got %d", len(route53.upsertedRecords))
	}
	if route53.upsertedRecords[0] != "devbox.example.com->54.1.2.3" {
		t.Errorf("Upserted record = %q, want %q", route53.upsertedRecords[0], "devbox.example.com->54.1.2.3")
	}
	if len(configuration.DNS.DNSRecords) != 1 {
		t.Fatalf("Expected 1 DNS record in config, got %d", len(configuration.DNS.DNSRecords))
	}
	if configuration.DNS.DNSRecords[0].Type != "A" {
		t.Errorf("DNS record type = %q, want %q", configuration.DNS.DNSRecords[0].Type, "A")
	}
}

func TestCreateCreatesNetworkWhenNoDefaultVPC(t *testing.T) {
	provider, _, ec2, _, _ := newFakeProvider()
	ec2.findVPCError = fmt.Errorf("no default VPC")
	ec2.networkStack = &NetworkStack{
		VpcID:                 "vpc-new123",
		SubnetID:              "subnet-new456",
		InternetGatewayID:     "igw-789",
		RouteTableID:          "rtb-abc",
		RouteTableAssociation: "rtbassoc-def",
	}
	cloudInitPath := createCloudInitFile(t)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
		},
	}

	err := provider.Create(context.Background(), configuration, cloudInitPath)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if !configuration.VM.CreatedVPC {
		t.Error("CreatedVPC should be true when VPC was created")
	}
	if configuration.VM.VpcID != "vpc-new123" {
		t.Errorf("VpcID = %q, want %q", configuration.VM.VpcID, "vpc-new123")
	}
	if configuration.VM.InternetGatewayID != "igw-789" {
		t.Errorf("InternetGatewayID = %q, want %q", configuration.VM.InternetGatewayID, "igw-789")
	}
}

func TestCreateUsesConfigVPCIfProvided(t *testing.T) {
	provider, _, ec2, _, _ := newFakeProvider()
	cloudInitPath := createCloudInitFile(t)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
			VpcID:        "vpc-explicit",
			SubnetID:     "subnet-explicit",
		},
	}

	err := provider.Create(context.Background(), configuration, cloudInitPath)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if configuration.VM.VpcID != "vpc-explicit" {
		t.Errorf("Should use explicit VPC, got %q", configuration.VM.VpcID)
	}
	if ec2.defaultVPCID == "" {
		t.Log("FindDefaultVPC should not have been needed")
	}
}

func TestCreateFailsOnMissingCloudInit(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
		},
	}

	err := provider.Create(context.Background(), configuration, "/nonexistent/cloud-init.yaml")
	if err == nil {
		t.Fatal("Create() should return error for missing cloud-init file")
	}
}

func TestCreateFailsOnUnsupportedOS(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()
	cloudInitPath := createCloudInitFile(t)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
			OS:           "windows-11",
		},
	}

	err := provider.Create(context.Background(), configuration, cloudInitPath)
	if err == nil {
		t.Fatal("Create() should return error for unsupported OS")
	}
}

func TestCreateFailsOnStackCreationError(t *testing.T) {
	provider, cloudFormation, _, _, _ := newFakeProvider()
	cloudFormation.createError = fmt.Errorf("stack limit exceeded")
	cloudInitPath := createCloudInitFile(t)

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:         "devbox",
			InstanceType: "t3.micro",
		},
	}

	err := provider.Create(context.Background(), configuration, cloudInitPath)
	if err == nil {
		t.Fatal("Create() should return error when stack creation fails")
	}
}

func TestDeleteCleansUpResources(t *testing.T) {
	provider, cloudFormation, _, route53, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:      "devbox",
			StackName: "goloo-devbox",
			PublicIP:  "54.1.2.3",
		},
		DNS: &config.DNSConfig{
			ZoneID: "Z1234567890",
			FQDN:   "devbox.example.com",
			DNSRecords: []config.DNSRecord{
				{Name: "devbox.example.com", Type: "A", Value: "54.1.2.3", TTL: 300},
			},
		},
	}

	err := provider.Delete(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if len(route53.deletedRecords) != 1 || route53.deletedRecords[0] != "devbox.example.com" {
		t.Errorf("Expected DNS record deletion, got %v", route53.deletedRecords)
	}
	if len(cloudFormation.deletedStacks) != 1 || cloudFormation.deletedStacks[0] != "goloo-devbox" {
		t.Errorf("Expected stack deletion, got %v", cloudFormation.deletedStacks)
	}
	if configuration.VM.InstanceID != "" {
		t.Error("InstanceID should be cleared after delete")
	}
	if configuration.VM.PublicIP != "" {
		t.Error("PublicIP should be cleared after delete")
	}
	if configuration.VM.StackName != "" {
		t.Error("StackName should be cleared after delete")
	}
	if configuration.DNS.ZoneID != "" {
		t.Error("DNS ZoneID should be cleared after delete")
	}
}

func TestDeleteCleansUpCreatedNetwork(t *testing.T) {
	provider, _, ec2, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:              "devbox",
			StackName:         "goloo-devbox",
			CreatedVPC:        true,
			VpcID:             "vpc-created",
			SubnetID:          "subnet-created",
			InternetGatewayID: "igw-created",
			RouteTableID:      "rtb-created",
		},
	}

	err := provider.Delete(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if len(ec2.deletedNetworks) != 1 {
		t.Fatalf("Expected 1 network deletion, got %d", len(ec2.deletedNetworks))
	}
	if ec2.deletedNetworks[0].VpcID != "vpc-created" {
		t.Errorf("Deleted network VpcID = %q, want %q", ec2.deletedNetworks[0].VpcID, "vpc-created")
	}
	if configuration.VM.CreatedVPC != false {
		t.Error("CreatedVPC should be false after delete")
	}
}

func TestDeleteSkipsNetworkCleanupWhenNotCreated(t *testing.T) {
	provider, _, ec2, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:       "devbox",
			StackName:  "goloo-devbox",
			CreatedVPC: false,
		},
	}

	err := provider.Delete(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if len(ec2.deletedNetworks) != 0 {
		t.Error("Should not delete network when CreatedVPC is false")
	}
}

func TestStatusReturnsInstanceInfo(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:       "devbox",
			InstanceID: "i-0123456789abcdef0",
		},
	}

	status, err := provider.Status(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Status() returned error: %v", err)
	}

	if status.Name != "devbox" {
		t.Errorf("Status.Name = %q, want %q", status.Name, "devbox")
	}
	if status.State != "running" {
		t.Errorf("Status.State = %q, want %q", status.State, "running")
	}
	if status.IP != "54.1.2.3" {
		t.Errorf("Status.IP = %q, want %q", status.IP, "54.1.2.3")
	}
	if status.Provider != "aws" {
		t.Errorf("Status.Provider = %q, want %q", status.Provider, "aws")
	}
}

func TestStatusFailsWithoutInstanceID(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{Name: "devbox"},
	}

	_, err := provider.Status(context.Background(), configuration)
	if err == nil {
		t.Fatal("Status() should return error when no instance ID")
	}
}

func TestStopCallsEC2(t *testing.T) {
	provider, _, ec2, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:       "devbox",
			InstanceID: "i-abc123",
		},
	}

	err := provider.Stop(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	if len(ec2.stoppedInstances) != 1 || ec2.stoppedInstances[0] != "i-abc123" {
		t.Errorf("Expected instance i-abc123 to be stopped, got %v", ec2.stoppedInstances)
	}
}

func TestStartCallsEC2(t *testing.T) {
	provider, _, ec2, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:       "devbox",
			InstanceID: "i-abc123",
		},
	}

	err := provider.Start(context.Background(), configuration)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if len(ec2.startedInstances) != 1 || ec2.startedInstances[0] != "i-abc123" {
		t.Errorf("Expected instance i-abc123 to be started, got %v", ec2.startedInstances)
	}
}

func TestStopFailsWithoutInstanceID(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{Name: "devbox"},
	}

	err := provider.Stop(context.Background(), configuration)
	if err == nil {
		t.Fatal("Stop() should return error when no instance ID")
	}
}

func TestSSHFailsWithoutPublicIP(t *testing.T) {
	provider := New("us-east-1")

	configuration := &config.Config{
		VM: &config.VMConfig{Name: "devbox"},
	}

	err := provider.SSH(context.Background(), configuration)
	if err == nil {
		t.Fatal("SSH() should return error when no public IP")
	}
}

func TestUninitializedProviderReturnsError(t *testing.T) {
	provider := New("us-east-1")

	configuration := &config.Config{
		VM: &config.VMConfig{Name: "devbox"},
	}

	_, err := provider.Status(context.Background(), configuration)
	if err == nil {
		t.Fatal("Uninitialized provider should return error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("Error should mention 'not initialized', got: %v", err)
	}
}

func TestSwapDNSUpdatesRecord(t *testing.T) {
	provider, _, _, route53, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:     "devbox",
			PublicIP: "54.9.8.7",
		},
		DNS: &config.DNSConfig{
			Hostname: "devbox",
			Domain:   "example.com",
			TTL:      60,
		},
	}

	err := provider.SwapDNS(context.Background(), configuration)
	if err != nil {
		t.Fatalf("SwapDNS() returned error: %v", err)
	}

	if len(route53.upsertedRecords) != 1 {
		t.Fatalf("Expected 1 upserted record, got %d", len(route53.upsertedRecords))
	}
	if route53.upsertedRecords[0] != "devbox.example.com->54.9.8.7" {
		t.Errorf("Upserted record = %q, want %q", route53.upsertedRecords[0], "devbox.example.com->54.9.8.7")
	}
	if configuration.DNS.FQDN != "devbox.example.com" {
		t.Errorf("FQDN = %q, want %q", configuration.DNS.FQDN, "devbox.example.com")
	}
}

func TestSwapDNSFailsWithoutDNSConfig(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:     "devbox",
			PublicIP: "54.1.2.3",
		},
	}

	err := provider.SwapDNS(context.Background(), configuration)
	if err == nil {
		t.Fatal("SwapDNS() should return error when no DNS config")
	}
}

func TestSwapDNSFailsWithoutPublicIP(t *testing.T) {
	provider, _, _, _, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{Name: "devbox"},
		DNS: &config.DNSConfig{
			Domain: "example.com",
		},
	}

	err := provider.SwapDNS(context.Background(), configuration)
	if err == nil {
		t.Fatal("SwapDNS() should return error when no public IP")
	}
}

func TestSwapDNSUsesExistingZoneID(t *testing.T) {
	provider, _, _, route53, _ := newFakeProvider()

	configuration := &config.Config{
		VM: &config.VMConfig{
			Name:     "devbox",
			PublicIP: "54.1.2.3",
		},
		DNS: &config.DNSConfig{
			Hostname: "devbox",
			Domain:   "example.com",
			ZoneID:   "Z-EXISTING",
			TTL:      60,
		},
	}

	err := provider.SwapDNS(context.Background(), configuration)
	if err != nil {
		t.Fatalf("SwapDNS() returned error: %v", err)
	}

	if configuration.DNS.ZoneID != "Z-EXISTING" {
		t.Errorf("Should preserve existing ZoneID, got %q", configuration.DNS.ZoneID)
	}
	if len(route53.upsertedRecords) != 1 {
		t.Error("Should still upsert the record")
	}
}

func TestLookupAMIPathUbuntu2404(t *testing.T) {
	path, err := LookupAMIPath("ubuntu-24.04")
	if err != nil {
		t.Fatalf("LookupAMIPath() returned error: %v", err)
	}
	expected := "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp2/ami-id"
	if path != expected {
		t.Errorf("LookupAMIPath(\"ubuntu-24.04\") = %q, want %q", path, expected)
	}
}

func TestLookupAMIPathUnsupportedOS(t *testing.T) {
	_, err := LookupAMIPath("windows-11")
	if err == nil {
		t.Fatal("LookupAMIPath() should return error for unsupported OS")
	}
}

func TestSupportedOperatingSystems(t *testing.T) {
	systems := SupportedOperatingSystems()
	if len(systems) != 7 {
		t.Errorf("SupportedOperatingSystems() returned %d entries, want 7", len(systems))
	}
	expected := []string{
		"amazon-linux-2", "amazon-linux-2023", "debian-11", "debian-12",
		"ubuntu-20.04", "ubuntu-22.04", "ubuntu-24.04",
	}
	sort.Strings(systems)
	for i, name := range expected {
		if i >= len(systems) || systems[i] != name {
			t.Errorf("SupportedOperatingSystems() missing %q", name)
		}
	}
}

func TestGenerateTemplateContainsUserData(t *testing.T) {
	template := GenerateTemplate("base64encodeddata")
	if !strings.Contains(template, "base64encodeddata") {
		t.Error("GenerateTemplate() should embed UserData in template")
	}
}

func TestGenerateTemplateContainsRequiredResources(t *testing.T) {
	template := GenerateTemplate("test")
	if !TemplateContainsResource(template, "SSHSecurityGroup") {
		t.Error("Template should contain SSHSecurityGroup resource")
	}
	if !TemplateContainsResource(template, "EC2Instance") {
		t.Error("Template should contain EC2Instance resource")
	}
}

func TestGenerateTemplateContainsOutputs(t *testing.T) {
	template := GenerateTemplate("test")
	for _, output := range []string{"InstanceId:", "PublicIP:", "SecurityGroupId:"} {
		if !strings.Contains(template, output) {
			t.Errorf("Template should contain output %s", output)
		}
	}
}

func TestBuildStackName(t *testing.T) {
	if BuildStackName("devbox") != "goloo-devbox" {
		t.Errorf("BuildStackName(\"devbox\") = %q, want %q", BuildStackName("devbox"), "goloo-devbox")
	}
}

func TestBuildNetworkStackName(t *testing.T) {
	if BuildNetworkStackName("devbox") != "goloo-devbox-network" {
		t.Errorf("BuildNetworkStackName(\"devbox\") = %q, want %q", BuildNetworkStackName("devbox"), "goloo-devbox-network")
	}
}

func TestBuildFQDN(t *testing.T) {
	if BuildFQDN("devbox", "example.com") != "devbox.example.com" {
		t.Errorf("BuildFQDN() = %q, want %q", BuildFQDN("devbox", "example.com"), "devbox.example.com")
	}
}

func TestAllAMIPathsStartWithAWSServicePrefix(t *testing.T) {
	for operatingSystem, path := range operatingSystemSSMPaths {
		if !strings.HasPrefix(path, "/aws/service/") {
			t.Errorf("AMI path for %q = %q, should start with '/aws/service/'", operatingSystem, path)
		}
	}
}
