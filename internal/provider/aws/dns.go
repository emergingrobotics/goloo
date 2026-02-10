package aws

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/goloo/internal/config"
)

func BuildFQDN(hostname string, domain string) string {
	return fmt.Sprintf("%s.%s", hostname, domain)
}

func (p *Provider) resolveZoneID(context context.Context, configuration *config.Config) (string, error) {
	if configuration.AWS != nil && configuration.AWS.ZoneID != "" {
		return configuration.AWS.ZoneID, nil
	}
	if configuration.DNS.ZoneID != "" {
		return configuration.DNS.ZoneID, nil
	}
	zoneID, err := p.Route53.FindZoneID(context, configuration.DNS.Domain)
	if err != nil {
		return "", fmt.Errorf("failed to find hosted zone for %s: %w", configuration.DNS.Domain, err)
	}
	if configuration.AWS != nil {
		configuration.AWS.ZoneID = zoneID
	}
	return zoneID, nil
}

func (p *Provider) resolveHostname(configuration *config.Config) string {
	if configuration.DNS.Hostname != "" {
		return configuration.DNS.Hostname
	}
	return configuration.VM.Name
}

func (p *Provider) createDNSRecords(context context.Context, configuration *config.Config) error {
	if p.Route53 == nil {
		return fmt.Errorf("Route53 client not configured")
	}

	hostname := p.resolveHostname(configuration)

	zoneID, err := p.resolveZoneID(context, configuration)
	if err != nil {
		return err
	}

	fqdn := BuildFQDN(hostname, configuration.DNS.Domain)
	configuration.AWS.FQDN = fqdn
	configuration.AWS.ZoneID = zoneID

	ttl := configuration.DNS.TTL
	if ttl == 0 {
		ttl = 300
	}

	if err := p.Route53.UpsertARecord(context, zoneID, fqdn, configuration.AWS.PublicIP, ttl); err != nil {
		return fmt.Errorf("failed to create A record for %s: %w", fqdn, err)
	}

	records := []config.DNSRecord{
		{Name: fqdn, Type: "A", Value: configuration.AWS.PublicIP, TTL: ttl},
	}

	if configuration.DNS.IsApexDomain {
		apexName := configuration.DNS.Domain
		if err := p.Route53.UpsertARecord(context, zoneID, apexName, configuration.AWS.PublicIP, ttl); err != nil {
			return fmt.Errorf("failed to create apex A record for %s: %w", apexName, err)
		}
		records = append(records, config.DNSRecord{
			Name: apexName, Type: "A", Value: configuration.AWS.PublicIP, TTL: ttl,
		})
	}

	for _, alias := range configuration.DNS.CNAMEAliases {
		aliasName := BuildFQDN(alias, configuration.DNS.Domain)
		if err := p.Route53.UpsertCNAMERecord(context, zoneID, aliasName, fqdn, ttl); err != nil {
			return fmt.Errorf("failed to create CNAME record %s -> %s: %w", aliasName, fqdn, err)
		}
		records = append(records, config.DNSRecord{
			Name: aliasName, Type: "CNAME", Value: fqdn, TTL: ttl,
		})
	}

	configuration.AWS.DNSRecords = records
	return nil
}

func (p *Provider) deleteDNSRecords(context context.Context, configuration *config.Config) error {
	if p.Route53 == nil {
		return nil
	}

	for _, record := range configuration.AWS.DNSRecords {
		switch record.Type {
		case "A":
			if err := p.Route53.DeleteARecord(context, configuration.AWS.ZoneID, record.Name, record.Value, record.TTL); err != nil {
				return fmt.Errorf("failed to delete A record %s: %w", record.Name, err)
			}
		case "CNAME":
			if err := p.Route53.DeleteCNAMERecord(context, configuration.AWS.ZoneID, record.Name, record.Value, record.TTL); err != nil {
				return fmt.Errorf("failed to delete CNAME record %s: %w", record.Name, err)
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
	if configuration.AWS == nil || configuration.AWS.PublicIP == "" {
		return fmt.Errorf("no public IP: VM must be running for dns swap")
	}

	hostname := p.resolveHostname(configuration)

	zoneID, err := p.resolveZoneID(context, configuration)
	if err != nil {
		return err
	}

	fqdn := BuildFQDN(hostname, configuration.DNS.Domain)
	ttl := configuration.DNS.TTL
	if ttl == 0 {
		ttl = 300
	}

	if err := p.Route53.UpsertARecord(context, zoneID, fqdn, configuration.AWS.PublicIP, ttl); err != nil {
		return fmt.Errorf("failed to swap DNS record for %s: %w", fqdn, err)
	}

	records := []config.DNSRecord{
		{Name: fqdn, Type: "A", Value: configuration.AWS.PublicIP, TTL: ttl},
	}

	if configuration.DNS.IsApexDomain {
		apexName := configuration.DNS.Domain
		if err := p.Route53.UpsertARecord(context, zoneID, apexName, configuration.AWS.PublicIP, ttl); err != nil {
			return fmt.Errorf("failed to swap apex A record for %s: %w", apexName, err)
		}
		records = append(records, config.DNSRecord{
			Name: apexName, Type: "A", Value: configuration.AWS.PublicIP, TTL: ttl,
		})
	}

	for _, alias := range configuration.DNS.CNAMEAliases {
		aliasName := BuildFQDN(alias, configuration.DNS.Domain)
		if err := p.Route53.UpsertCNAMERecord(context, zoneID, aliasName, fqdn, ttl); err != nil {
			return fmt.Errorf("failed to swap CNAME record %s -> %s: %w", aliasName, fqdn, err)
		}
		records = append(records, config.DNSRecord{
			Name: aliasName, Type: "CNAME", Value: fqdn, TTL: ttl,
		})
	}

	configuration.AWS.FQDN = fqdn
	configuration.AWS.DNSRecords = records
	configuration.AWS.ZoneID = zoneID
	return nil
}
