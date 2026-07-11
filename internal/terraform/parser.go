package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resource represents a planned resource extracted from Terraform
type Resource struct {
	Type       string            `json:"type"`        // aws_instance, aws_s3_bucket, etc.
	Name       string            `json:"name"`        // resource name in .tf
	Address    string            `json:"address"`     // module.x.aws_instance.y
	Provider   string            `json:"provider"`    // aws, google, azurerm
	SofeType   string            `json:"sofe_type"`   // mapped to aws.ec2, aws.s3, etc.
	Tags       map[string]string `json:"tags"`        // resource tags
	Attributes map[string]interface{} `json:"attributes"` // all attributes
}

// Finding represents a policy violation found in planned resources
type Finding struct {
	PolicyName string `json:"policy_name"`
	Severity   string `json:"severity"`
	ResourceID string `json:"resource_id"`
	Type       string `json:"resource_type"`
	Message    string `json:"message"`
}

// ScanResult is the output of a terraform scan
type ScanResult struct {
	Path           string    `json:"path"`
	Mode           string    `json:"mode"` // "plan" or "dir"
	ResourceCount  int       `json:"resource_count"`
	FindingsCount  int       `json:"findings_count"`
	PoliciesChecked int      `json:"policies_checked"`
	Resources      []Resource `json:"resources"`
	Findings       []Finding  `json:"findings"`
}

// ParsePlanJSON parses a terraform show -json output (tfplan.json)
func ParsePlanJSON(path string) ([]Resource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read plan file: %w", err)
	}

	var plan struct {
		PlannedValues struct {
			RootModule struct {
				Resources []struct {
					Type    string                 `json:"type"`
					Name    string                 `json:"name"`
					Address string                 `json:"address"`
					Provider string                `json:"provider_name"`
					Values  map[string]interface{} `json:"values"`
				} `json:"resources"`
				ChildModules []struct {
					Resources []struct {
						Type    string                 `json:"type"`
						Name    string                 `json:"name"`
						Address string                 `json:"address"`
						Provider string                `json:"provider_name"`
						Values  map[string]interface{} `json:"values"`
					} `json:"resources"`
				} `json:"child_modules"`
			} `json:"root_module"`
		} `json:"planned_values"`
		ResourceChanges []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Address string `json:"address"`
			Change  struct {
				Actions []string               `json:"actions"`
				After   map[string]interface{} `json:"after"`
			} `json:"change"`
		} `json:"resource_changes"`
	}

	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("invalid plan JSON: %w", err)
	}

	var resources []Resource

	// From planned_values
	for _, r := range plan.PlannedValues.RootModule.Resources {
		resources = append(resources, newResource(r.Type, r.Name, r.Address, r.Provider, r.Values))
	}
	for _, mod := range plan.PlannedValues.RootModule.ChildModules {
		for _, r := range mod.Resources {
			resources = append(resources, newResource(r.Type, r.Name, r.Address, r.Provider, r.Values))
		}
	}

	// If planned_values is empty, try resource_changes (create/update actions)
	if len(resources) == 0 {
		for _, rc := range plan.ResourceChanges {
			for _, action := range rc.Change.Actions {
				if action == "create" || action == "update" {
					resources = append(resources, newResource(rc.Type, rc.Name, rc.Address, "", rc.Change.After))
					break
				}
			}
		}
	}

	return resources, nil
}

// ParseDirectory scans .tf files in a directory for resource blocks
func ParseDirectory(dir string) ([]Resource, error) {
	var resources []Resource

	files, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .tf files found in %s", dir)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		content := string(data)
		parsed := parseHCLResources(content, file)
		resources = append(resources, parsed...)
	}

	return resources, nil
}

// parseHCLResources does basic regex-like parsing of resource blocks (not full HCL parser)
func parseHCLResources(content, filename string) []Resource {
	var resources []Resource
	lines := strings.Split(content, "\n")

	var currentType, currentName string
	var inResource bool
	var braceDepth int
	var attrs map[string]interface{}
	var tags map[string]string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect resource block start
		if strings.HasPrefix(trimmed, "resource ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				currentType = strings.Trim(parts[1], "\"")
				currentName = strings.Trim(parts[2], "\"")
				inResource = true
				braceDepth = 0
				attrs = make(map[string]interface{})
				tags = make(map[string]string)
			}
		}

		if inResource {
			braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

			// Parse simple key = value attributes
			if strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "//") {
				kv := strings.SplitN(trimmed, "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					val = strings.Trim(val, "\"")

					// Detect tags
					if key != "" && key != "tags" && key != "{" && key != "}" {
						attrs[key] = val
					}
				}
			}

			// Simple tag detection (tags = { key = "value" })
			if strings.Contains(trimmed, "=") && braceDepth > 1 {
				kv := strings.SplitN(trimmed, "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(strings.Trim(kv[1], "\""))
					if key != "{" && key != "}" && key != "" {
						tags[key] = val
					}
				}
			}

			if braceDepth <= 0 && inResource && currentType != "" {
				address := fmt.Sprintf("%s.%s", currentType, currentName)
				r := newResource(currentType, currentName, address, "", attrs)
				r.Tags = tags
				resources = append(resources, r)
				inResource = false
				currentType = ""
				currentName = ""
			}
		}
	}

	return resources
}

func newResource(tfType, name, address, provider string, values map[string]interface{}) Resource {
	r := Resource{
		Type:       tfType,
		Name:       name,
		Address:    address,
		Provider:   provider,
		SofeType:   mapTfTypeToSofe(tfType),
		Attributes: values,
		Tags:       extractTags(values),
	}
	return r
}

// extractTags pulls tags from resource values
func extractTags(values map[string]interface{}) map[string]string {
	tags := make(map[string]string)
	if t, ok := values["tags"]; ok {
		switch v := t.(type) {
		case map[string]interface{}:
			for k, val := range v {
				tags[k] = fmt.Sprintf("%v", val)
			}
		}
	}
	if t, ok := values["tags_all"]; ok {
		switch v := t.(type) {
		case map[string]interface{}:
			for k, val := range v {
				if _, exists := tags[k]; !exists {
					tags[k] = fmt.Sprintf("%v", val)
				}
			}
		}
	}
	return tags
}

// mapTfTypeToSofe converts Terraform resource types to SOFE types
func mapTfTypeToSofe(tfType string) string {
	mapping := map[string]string{
		"aws_instance":                    "aws.ec2",
		"aws_launch_template":             "aws.ec2",
		"aws_autoscaling_group":           "aws.ec2",
		"aws_s3_bucket":                   "aws.s3",
		"aws_s3_bucket_acl":               "aws.s3",
		"aws_lambda_function":             "aws.lambda",
		"aws_rds_cluster":                 "aws.rds",
		"aws_db_instance":                 "aws.rds",
		"aws_dynamodb_table":              "aws.dynamodb",
		"aws_ecs_service":                 "aws.ecs",
		"aws_ecs_cluster":                 "aws.ecs",
		"aws_eks_cluster":                 "aws.eks",
		"aws_elasticache_cluster":         "aws.elasticache",
		"aws_lb":                          "aws.elb",
		"aws_alb":                         "aws.elb",
		"aws_nat_gateway":                 "aws.natgateway",
		"aws_ebs_volume":                  "aws.ebs",
		"aws_cloudfront_distribution":     "aws.cloudfront",
		"aws_api_gateway_rest_api":        "aws.apigateway",
		"aws_apigatewayv2_api":            "aws.apigateway",
		"aws_route53_zone":                "aws.route53",
		"aws_secretsmanager_secret":       "aws.secretsmanager",
		"aws_redshift_cluster":            "aws.redshift",
		"aws_sagemaker_endpoint":          "aws.sagemaker",
		"aws_sns_topic":                   "aws.sns",
		"aws_sqs_queue":                   "aws.sqs",
	}

	if sofeType, ok := mapping[tfType]; ok {
		return sofeType
	}

	// Generic: aws_xxx → aws.xxx (strip aws_ prefix, take first part)
	if strings.HasPrefix(tfType, "aws_") {
		parts := strings.SplitN(tfType[4:], "_", 2)
		return "aws." + parts[0]
	}

	return tfType
}
