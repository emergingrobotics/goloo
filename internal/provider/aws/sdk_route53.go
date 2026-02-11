package aws

import (
	"context"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type sdkRoute53Client struct {
	client *route53.Client
}

func NewSDKRoute53Client(configuration awssdk.Config) Route53Client {
	return &sdkRoute53Client{client: route53.NewFromConfig(configuration)}
}

func ensureTrailingDot(name string) string {
	if !strings.HasSuffix(name, ".") {
		return name + "."
	}
	return name
}

func (r *sdkRoute53Client) FindZoneID(context context.Context, domain string) (string, error) {
	domain = ensureTrailingDot(domain)

	result, err := r.client.ListHostedZonesByName(context, &route53.ListHostedZonesByNameInput{
		DNSName: &domain,
	})
	if err != nil {
		return "", fmt.Errorf("ListHostedZonesByName failed: %w", err)
	}

	for _, zone := range result.HostedZones {
		if *zone.Name == domain {
			zoneID := strings.TrimPrefix(*zone.Id, "/hostedzone/")
			return zoneID, nil
		}
	}

	return "", fmt.Errorf("hosted zone not found for domain %s", domain)
}

func (r *sdkRoute53Client) UpsertARecord(context context.Context, zoneID string, name string, ip string, ttl int) error {
	name = ensureTrailingDot(name)
	_, err := r.client.ChangeResourceRecordSets(context, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &zoneID,
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{
				{
					Action: r53types.ChangeActionUpsert,
					ResourceRecordSet: &r53types.ResourceRecordSet{
						Name: &name,
						Type: r53types.RRTypeA,
						TTL:  awssdk.Int64(int64(ttl)),
						ResourceRecords: []r53types.ResourceRecord{
							{Value: &ip},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("UpsertARecord %s -> %s failed: %w", name, ip, err)
	}
	return nil
}

func (r *sdkRoute53Client) DeleteARecord(context context.Context, zoneID string, name string, ip string, ttl int) error {
	name = ensureTrailingDot(name)
	_, err := r.client.ChangeResourceRecordSets(context, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &zoneID,
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{
				{
					Action: r53types.ChangeActionDelete,
					ResourceRecordSet: &r53types.ResourceRecordSet{
						Name: &name,
						Type: r53types.RRTypeA,
						TTL:  awssdk.Int64(int64(ttl)),
						ResourceRecords: []r53types.ResourceRecord{
							{Value: &ip},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("DeleteARecord %s failed: %w", name, err)
	}
	return nil
}

func (r *sdkRoute53Client) UpsertCNAMERecord(context context.Context, zoneID string, name string, target string, ttl int) error {
	name = ensureTrailingDot(name)
	target = ensureTrailingDot(target)
	_, err := r.client.ChangeResourceRecordSets(context, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &zoneID,
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{
				{
					Action: r53types.ChangeActionUpsert,
					ResourceRecordSet: &r53types.ResourceRecordSet{
						Name: &name,
						Type: r53types.RRTypeCname,
						TTL:  awssdk.Int64(int64(ttl)),
						ResourceRecords: []r53types.ResourceRecord{
							{Value: &target},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("UpsertCNAMERecord %s -> %s failed: %w", name, target, err)
	}
	return nil
}

func (r *sdkRoute53Client) DeleteCNAMERecord(context context.Context, zoneID string, name string, target string, ttl int) error {
	name = ensureTrailingDot(name)
	target = ensureTrailingDot(target)
	_, err := r.client.ChangeResourceRecordSets(context, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &zoneID,
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{
				{
					Action: r53types.ChangeActionDelete,
					ResourceRecordSet: &r53types.ResourceRecordSet{
						Name: &name,
						Type: r53types.RRTypeCname,
						TTL:  awssdk.Int64(int64(ttl)),
						ResourceRecords: []r53types.ResourceRecord{
							{Value: &target},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("DeleteCNAMERecord %s failed: %w", name, err)
	}
	return nil
}
