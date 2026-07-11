package terraform

import (
	"fmt"
	"strings"
)

// Policy defines a pre-deploy check
type Policy struct {
	Name        string
	Severity    string
	Description string
	Check       func(r Resource) *Finding
}

// DefaultPolicies returns the built-in pre-deploy policies
func DefaultPolicies() []Policy {
	return []Policy{
		{
			Name:        "require-cost-tags",
			Severity:    "medium",
			Description: "All resources must have owner, env, and costcenter tags",
			Check:       checkCostTags,
		},
		{
			Name:        "no-oversized-staging",
			Severity:    "high",
			Description: "Non-prod environments should not use large instance types",
			Check:       checkOversizedStaging,
		},
		{
			Name:        "s3-encryption-required",
			Severity:    "high",
			Description: "S3 buckets must have encryption configured",
			Check:       checkS3Encryption,
		},
		{
			Name:        "no-public-s3",
			Severity:    "critical",
			Description: "S3 buckets must not have public ACL",
			Check:       checkPublicS3,
		},
		{
			Name:        "ec2-requires-metadata-v2",
			Severity:    "medium",
			Description: "EC2 instances should use IMDSv2 (metadata hop limit)",
			Check:       checkMetadataV2,
		},
		{
			Name:        "rds-no-public-access",
			Severity:    "high",
			Description: "RDS instances must not be publicly accessible",
			Check:       checkRDSPublic,
		},
	}
}

func checkCostTags(r Resource) *Finding {
	requiredTags := []string{"owner", "env", "costcenter"}
	missing := []string{}

	for _, tag := range requiredTags {
		found := false
		for k := range r.Tags {
			if strings.EqualFold(k, tag) || strings.EqualFold(k, "cost_center") || strings.EqualFold(k, "cost-center") {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, tag)
		}
	}

	if len(missing) > 0 {
		return &Finding{
			PolicyName: "require-cost-tags",
			Severity:   "medium",
			ResourceID: r.Address,
			Type:       r.SofeType,
			Message:    fmt.Sprintf("Missing tags: %s", strings.Join(missing, ", ")),
		}
	}
	return nil
}

func checkOversizedStaging(r Resource) *Finding {
	if r.SofeType != "aws.ec2" {
		return nil
	}

	instanceType, _ := r.Attributes["instance_type"].(string)
	if instanceType == "" {
		return nil
	}

	// Check if environment is non-prod
	env := ""
	for k, v := range r.Tags {
		if strings.EqualFold(k, "env") || strings.EqualFold(k, "environment") {
			env = strings.ToLower(v)
		}
	}

	if env == "" || env == "prod" || env == "production" {
		return nil // only check non-prod
	}

	// Large instance types
	oversized := []string{"xlarge", "2xlarge", "4xlarge", "8xlarge", "12xlarge", "16xlarge", "24xlarge", "metal"}
	for _, suffix := range oversized {
		if strings.Contains(instanceType, suffix) {
			return &Finding{
				PolicyName: "no-oversized-staging",
				Severity:   "high",
				ResourceID: r.Address,
				Type:       r.SofeType,
				Message:    fmt.Sprintf("Instance type %s is oversized for env=%s", instanceType, env),
			}
		}
	}
	return nil
}

func checkS3Encryption(r Resource) *Finding {
	if r.SofeType != "aws.s3" || r.Type != "aws_s3_bucket" {
		return nil
	}

	// In tfplan.json, encryption is often a nested block or separate resource
	// Basic check: if we see the bucket but no encryption config in attributes
	if _, ok := r.Attributes["server_side_encryption_configuration"]; !ok {
		// Check if encryption attribute exists
		if _, ok2 := r.Attributes["bucket_encryption"]; !ok2 {
			return &Finding{
				PolicyName: "s3-encryption-required",
				Severity:   "high",
				ResourceID: r.Address,
				Type:       r.SofeType,
				Message:    "S3 bucket has no encryption configuration (SSE not detected)",
			}
		}
	}
	return nil
}

func checkPublicS3(r Resource) *Finding {
	if r.SofeType != "aws.s3" {
		return nil
	}

	// Check ACL
	acl, _ := r.Attributes["acl"].(string)
	if acl == "public-read" || acl == "public-read-write" {
		return &Finding{
			PolicyName: "no-public-s3",
			Severity:   "critical",
			ResourceID: r.Address,
			Type:       r.SofeType,
			Message:    fmt.Sprintf("S3 bucket has public ACL: %s", acl),
		}
	}
	return nil
}

func checkMetadataV2(r Resource) *Finding {
	if r.SofeType != "aws.ec2" || r.Type != "aws_instance" {
		return nil
	}

	// Check metadata_options
	if meta, ok := r.Attributes["metadata_options"]; ok {
		if metaMap, ok := meta.(map[string]interface{}); ok {
			if endpoint, ok := metaMap["http_tokens"].(string); ok && endpoint == "required" {
				return nil // IMDSv2 enforced
			}
		}
	}

	return &Finding{
		PolicyName: "ec2-requires-metadata-v2",
		Severity:   "medium",
		ResourceID: r.Address,
		Type:       r.SofeType,
		Message:    "EC2 instance does not enforce IMDSv2 (http_tokens != required)",
	}
}

func checkRDSPublic(r Resource) *Finding {
	if r.SofeType != "aws.rds" {
		return nil
	}

	if pub, ok := r.Attributes["publicly_accessible"]; ok {
		if pub == true || pub == "true" {
			return &Finding{
				PolicyName: "rds-no-public-access",
				Severity:   "high",
				ResourceID: r.Address,
				Type:       r.SofeType,
				Message:    "RDS instance is publicly accessible",
			}
		}
	}
	return nil
}

// Evaluate runs all policies against all resources
func Evaluate(resources []Resource, policies []Policy) []Finding {
	var findings []Finding
	for _, r := range resources {
		for _, p := range policies {
			if f := p.Check(r); f != nil {
				findings = append(findings, *f)
			}
		}
	}
	return findings
}
