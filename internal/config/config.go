package config

type Config struct {
	VM  *VMConfig  `json:"vm,omitempty"`
	DNS *DNSConfig `json:"dns,omitempty"`
}

type VMConfig struct {
	Name          string  `json:"name"`
	CloudInitFile string  `json:"cloud_init_file,omitempty"`
	Users         []User  `json:"users,omitempty"`
	CPUs          int     `json:"cpus,omitempty"`
	Memory        string  `json:"memory,omitempty"`
	Disk          string  `json:"disk,omitempty"`
	Image         string  `json:"image,omitempty"`
	Mounts        []Mount `json:"mounts,omitempty"`

	InstanceType string `json:"instance_type,omitempty"`
	OS           string `json:"os,omitempty"`
	Region       string `json:"region,omitempty"`
	VpcID        string `json:"vpc_id,omitempty"`
	SubnetID     string `json:"subnet_id,omitempty"`

	PublicIP      string `json:"public_ip,omitempty"`
	InstanceID    string `json:"instance_id,omitempty"`
	StackID       string `json:"stack_id,omitempty"`
	StackName     string `json:"stack_name,omitempty"`
	SecurityGroup string `json:"security_group,omitempty"`
	AMIID         string `json:"ami_id,omitempty"`

	CreatedVPC            bool   `json:"created_vpc,omitempty"`
	CreatedSubnet         bool   `json:"created_subnet,omitempty"`
	InternetGatewayID     string `json:"internet_gateway_id,omitempty"`
	RouteTableID          string `json:"route_table_id,omitempty"`
	RouteTableAssociation string `json:"route_table_association_id,omitempty"`
}

type DNSConfig struct {
	Hostname     string   `json:"hostname,omitempty"`
	Domain       string   `json:"domain,omitempty"`
	TTL          int      `json:"ttl,omitempty"`
	IsApexDomain bool     `json:"is_apex_domain,omitempty"`
	CNAMEAliases []string `json:"cname_aliases,omitempty"`

	ZoneID     string      `json:"zone_id,omitempty"`
	FQDN       string      `json:"fqdn,omitempty"`
	DNSRecords []DNSRecord `json:"dns_records,omitempty"`
}

type User struct {
	Username       string `json:"username"`
	GitHubUsername string `json:"github_username"`
}

type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type DNSRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}
