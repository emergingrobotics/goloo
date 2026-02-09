package aws

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/goloo/internal/config"
)

func BuildFQDN(hostname string, domain string) string {
	return fmt.Sprintf("%s.%s", hostname, domain)
}

func (p *Provider) createDNSRecords(context context.Context, configuration *config.Config) error {
	if p.Route53 == nil {
		return fmt.Errorf("Route53 client not configured")
	}

	hostname := configuration.DNS.Hostname
	if hostname == "" {
		hostname = configuration.VM.Name
	}

	zoneID, err := p.Route53.FindZoneID(context, configuration.DNS.Domain)
	if err != nil {
		return fmt.Errorf("failed to find hosted zone for %s: %w", configuration.DNS.Domain, err)
	}
	configuration.DNS.ZoneID = zoneID

	fqdn := BuildFQDN(hostname, configuration.DNS.Domain)
	configuration.DNS.FQDN = fqdn

	ttl := configuration.DNS.TTL
	if ttl == 0 {
		ttl = 300
	}

	if err := p.Route53.UpsertARecord(context, zoneID, fqdn, configuration.VM.PublicIP, ttl); err != nil {
		return fmt.Errorf("failed to create A record for %s: %w", fqdn, err)
	}

	configuration.DNS.DNSRecords = []config.DNSRecord{
		{Name: fqdn, Type: "A", Value: configuration.VM.PublicIP, TTL: ttl},
	}

	return nil
}

func (p *Provider) deleteDNSRecords(context context.Context, configuration *config.Config) error {
	if p.Route53 == nil {
		return nil
	}

	for _, record := range configuration.DNS.DNSRecords {
		if record.Type == "A" {
			if err := p.Route53.DeleteARecord(context, configuration.DNS.ZoneID, record.Name, record.Value, record.TTL); err != nil {
				return fmt.Errorf("failed to delete A record %s: %w", record.Name, err)
			}
		}
	}

	return nil
}

func (p *Provider) SwapDNS(context context.Context, configuration *config.Config) error {
	if err := p.validateClients(); err != nil {
		return err
	}
	if p.Route53 == nil {
		return fmt.Errorf("Route53 client not configured")
	}
	if configuration.DNS == nil || configuration.DNS.Domain == "" {
		return fmt.Errorf("DNS configuration required for dns swap: add 'dns' section to config")
	}
	if configuration.VM.PublicIP == "" {
		return fmt.Errorf("no public IP: VM must be running for dns swap")
	}

	hostname := configuration.DNS.Hostname
	if hostname == "" {
		hostname = configuration.VM.Name
	}

	zoneID := configuration.DNS.ZoneID
	if zoneID == "" {
		var err error
		zoneID, err = p.Route53.FindZoneID(context, configuration.DNS.Domain)
		if err != nil {
			return fmt.Errorf("failed to find hosted zone for %s: %w", configuration.DNS.Domain, err)
		}
		configuration.DNS.ZoneID = zoneID
	}

	fqdn := BuildFQDN(hostname, configuration.DNS.Domain)
	ttl := configuration.DNS.TTL
	if ttl == 0 {
		ttl = 300
	}

	if err := p.Route53.UpsertARecord(context, zoneID, fqdn, configuration.VM.PublicIP, ttl); err != nil {
		return fmt.Errorf("failed to swap DNS record for %s: %w", fqdn, err)
	}

	configuration.DNS.FQDN = fqdn
	configuration.DNS.DNSRecords = []config.DNSRecord{
		{Name: fqdn, Type: "A", Value: configuration.VM.PublicIP, TTL: ttl},
	}

	return nil
}
