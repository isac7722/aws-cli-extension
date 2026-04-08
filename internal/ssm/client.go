package ssm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMAPI defines the subset of AWS SSM client methods used by Client.
// This interface enables testing with mock implementations.
type SSMAPI interface {
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
	DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
	DeleteParameters(ctx context.Context, params *ssm.DeleteParametersInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParametersOutput, error)
}

// ClientOptions configures the SSM client.
type ClientOptions struct {
	// Profile is the AWS profile name. If empty, default credential chain is used.
	Profile string
	// Region is the AWS region. If empty, default region resolution is used.
	Region string
	// AccessKeyID and SecretAccessKey allow direct credential injection (e.g. from awse's profile store).
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// Client wraps the AWS SSM API for parameter browsing operations.
type Client struct {
	api SSMAPI
}

// NewClient creates a new SSM client configured with the given options.
// It loads AWS credentials using the standard SDK config chain, with optional
// profile and region overrides.
func NewClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	var cfgOpts []func(*awsconfig.LoadOptions) error

	if opts.Region != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithRegion(opts.Region))
	}

	if opts.Profile != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithSharedConfigProfile(opts.Profile))
	}

	// If explicit credentials are provided, use them directly.
	if opts.AccessKeyID != "" && opts.SecretAccessKey != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				opts.AccessKeyID,
				opts.SecretAccessKey,
				opts.SessionToken,
			),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{
		api: ssm.NewFromConfig(cfg),
	}, nil
}

// NewClientWithAPI creates a Client with a custom SSM API implementation.
// This is intended for testing.
func NewClientWithAPI(api SSMAPI) *Client {
	return &Client{api: api}
}

// ListParameters recursively fetches all parameters under the given path prefix.
// It handles pagination automatically and returns a flat list of FlatParam suitable
// for BuildTree. Parameters are fetched without decryption by default.
func (c *Client) ListParameters(ctx context.Context, prefix string) ([]FlatParam, error) {
	prefix = normalizePath(prefix)

	var params []FlatParam
	var nextToken *string

	for {
		input := &ssm.GetParametersByPathInput{
			Path:           aws.String(prefix),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(false),
			MaxResults:     aws.Int32(10), // AWS maximum per page
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		output, err := c.api.GetParametersByPath(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("GetParametersByPath failed for %q: %w", prefix, err)
		}

		for _, p := range output.Parameters {
			params = append(params, flatParamFromAWS(p))
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}

	return params, nil
}

// GetParameterValue fetches a single parameter's value by name/path.
// If decrypt is true, SecureString values are decrypted.
func (c *Client) GetParameterValue(ctx context.Context, name string, decrypt bool) (string, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(decrypt),
	}

	output, err := c.api.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("GetParameter failed for %q: %w", name, err)
	}

	if output.Parameter == nil || output.Parameter.Value == nil {
		return "", nil
	}

	return *output.Parameter.Value, nil
}

// flatParamFromAWS converts an AWS SSM Parameter to our FlatParam type.
func flatParamFromAWS(p ssmtypes.Parameter) FlatParam {
	fp := FlatParam{
		Path: aws.ToString(p.Name),
		Meta: &ParameterMeta{
			Type:     string(p.Type),
			Version:  p.Version,
			DataType: aws.ToString(p.DataType),
			ARN:      aws.ToString(p.ARN),
		},
	}

	if p.LastModifiedDate != nil {
		fp.Meta.LastModified = *p.LastModifiedDate
	}

	return fp
}

// normalizePath ensures a path starts with "/" and does not end with "/" (except root).
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// Root "/" stays as-is; otherwise trim trailing slash.
	if path != "/" {
		path = strings.TrimRight(path, "/")
	}
	return path
}

// ListParametersFiltered fetches parameters under prefix and filters by an optional
// name substring. This is useful for search functionality in the TUI.
func (c *Client) ListParametersFiltered(ctx context.Context, prefix, filter string) ([]FlatParam, error) {
	all, err := c.ListParameters(ctx, prefix)
	if err != nil {
		return nil, err
	}

	if filter == "" {
		return all, nil
	}

	filter = strings.ToLower(filter)
	var filtered []FlatParam
	for _, p := range all {
		if strings.Contains(strings.ToLower(p.Path), filter) {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

// ParameterDetail holds full parameter information including its value.
type ParameterDetail struct {
	FlatParam
	// Value is the parameter value. For SecureString, only populated when decrypted.
	Value string
}

// GetParameterDetail fetches a single parameter with full metadata and value.
func (c *Client) GetParameterDetail(ctx context.Context, name string, decrypt bool) (*ParameterDetail, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(decrypt),
	}

	output, err := c.api.GetParameter(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("GetParameter failed for %q: %w", name, err)
	}

	if output.Parameter == nil {
		return nil, fmt.Errorf("parameter %q not found", name)
	}

	p := output.Parameter
	detail := &ParameterDetail{
		FlatParam: FlatParam{
			Path: aws.ToString(p.Name),
			Meta: &ParameterMeta{
				Type:     string(p.Type),
				Version:  p.Version,
				DataType: aws.ToString(p.DataType),
				ARN:      aws.ToString(p.ARN),
			},
		},
	}
	if p.LastModifiedDate != nil {
		detail.Meta.LastModified = *p.LastModifiedDate
	}
	if p.Value != nil {
		detail.Value = *p.Value
	}

	return detail, nil
}

// ParameterResult holds the result of a GetParameter call with full metadata.
type ParameterResult struct {
	// Name is the parameter name/path.
	Name string
	// Value is the parameter value. For SecureString, populated only when WithDecryption is true.
	Value string
	// Type is the SSM parameter type: String, StringList, or SecureString.
	Type string
	// Version is the parameter version number.
	Version int64
	// ARN is the full Amazon Resource Name for the parameter.
	ARN string
	// LastModified is the timestamp of the last parameter update.
	LastModified time.Time
}

// GetParameter fetches a single SSM parameter by name with full metadata.
// The decrypt flag controls whether SecureString values are decrypted via the
// AWS KMS WithDecryption option. Returns the parameter value, type, version,
// ARN, and last-modified timestamp.
func (c *Client) GetParameter(ctx context.Context, name string, decrypt bool) (*ParameterResult, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(decrypt),
	}

	output, err := c.api.GetParameter(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("GetParameter failed for %q: %w", name, err)
	}

	if output.Parameter == nil {
		return nil, fmt.Errorf("parameter %q not found", name)
	}

	p := output.Parameter
	result := &ParameterResult{
		Name:    aws.ToString(p.Name),
		Type:    string(p.Type),
		Version: p.Version,
		ARN:     aws.ToString(p.ARN),
	}

	if p.Value != nil {
		result.Value = *p.Value
	}
	if p.LastModifiedDate != nil {
		result.LastModified = *p.LastModifiedDate
	}

	return result, nil
}

// PutParameterInput holds the input for creating or updating an SSM parameter.
type PutParameterInput struct {
	// Name is the fully qualified parameter name (e.g. "/app/config/db_host").
	Name string
	// Value is the parameter value.
	Value string
	// Type is the parameter type: String, StringList, or SecureString.
	Type string
	// Description is an optional human-readable description.
	Description string
	// Overwrite allows updating an existing parameter. When false, PutParameter
	// returns an error if the parameter already exists.
	Overwrite bool
}

// PutParameter creates or updates an SSM parameter.
// Returns the version number of the created/updated parameter.
func (c *Client) PutParameter(ctx context.Context, input PutParameterInput) (int64, error) {
	paramType, err := parseParameterType(input.Type)
	if err != nil {
		return 0, err
	}

	putInput := &ssm.PutParameterInput{
		Name:      aws.String(input.Name),
		Value:     aws.String(input.Value),
		Type:      paramType,
		Overwrite: aws.Bool(input.Overwrite),
	}

	if input.Description != "" {
		putInput.Description = aws.String(input.Description)
	}

	result, err := c.api.PutParameter(ctx, putInput)
	if err != nil {
		return 0, fmt.Errorf("PutParameter failed for %q: %w", input.Name, err)
	}

	return result.Version, nil
}

// parseParameterType converts a string to an SSM ParameterType enum value.
func parseParameterType(t string) (ssmtypes.ParameterType, error) {
	switch t {
	case "String":
		return ssmtypes.ParameterTypeString, nil
	case "StringList":
		return ssmtypes.ParameterTypeStringList, nil
	case "SecureString":
		return ssmtypes.ParameterTypeSecureString, nil
	default:
		return "", fmt.Errorf("invalid parameter type %q: must be String, StringList, or SecureString", t)
	}
}

// UpdateParameterInput holds the input for updating an existing SSM parameter.
// Unlike PutParameterInput, Overwrite is always true and cannot be disabled.
type UpdateParameterInput struct {
	// Name is the fully qualified parameter name (e.g. "/app/config/db_host").
	Name string
	// Value is the new parameter value.
	Value string
	// Type is the parameter type: String, StringList, or SecureString.
	// If empty, the caller should resolve the existing type before calling.
	Type string
	// Description is an optional human-readable description.
	Description string
}

// UpdateParameter updates an existing SSM parameter with Overwrite enabled.
// This is a convenience wrapper around PutParameter that enforces Overwrite=true,
// which is the standard behavior for parameter updates.
// Returns the new version number of the updated parameter.
func (c *Client) UpdateParameter(ctx context.Context, input UpdateParameterInput) (int64, error) {
	if input.Name == "" {
		return 0, fmt.Errorf("parameter name is required")
	}
	if input.Value == "" {
		return 0, fmt.Errorf("parameter value is required")
	}
	if input.Type == "" {
		return 0, fmt.Errorf("parameter type is required (resolve from existing parameter if not specified)")
	}

	return c.PutParameter(ctx, PutParameterInput{
		Name:        input.Name,
		Value:       input.Value,
		Type:        input.Type,
		Description: input.Description,
		Overwrite:   true,
	})
}

// DeleteParameter deletes a single SSM parameter by name/path.
// Returns an error if the parameter does not exist or if the API call fails.
func (c *Client) DeleteParameter(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("parameter name is required")
	}

	input := &ssm.DeleteParameterInput{
		Name: aws.String(name),
	}

	_, err := c.api.DeleteParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("DeleteParameter failed for %q: %w", name, err)
	}

	return nil
}

// DeleteParametersResult holds the result of a batch delete operation.
type DeleteParametersResult struct {
	// DeletedParameters contains the names of parameters that were successfully deleted.
	DeletedParameters []string
	// InvalidParameters contains the names of parameters that could not be found or deleted.
	InvalidParameters []string
}

// DeleteParameters deletes multiple SSM parameters in a single API call.
// AWS allows up to 10 parameters per batch call. If more than 10 names are
// provided, they are automatically chunked into multiple API calls.
// Returns the successfully deleted and invalid parameter names.
func (c *Client) DeleteParameters(ctx context.Context, names []string) (*DeleteParametersResult, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one parameter name is required")
	}

	result := &DeleteParametersResult{}

	// AWS DeleteParameters supports max 10 names per call.
	const maxBatchSize = 10

	for i := 0; i < len(names); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(names) {
			end = len(names)
		}
		chunk := names[i:end]

		input := &ssm.DeleteParametersInput{
			Names: chunk,
		}

		output, err := c.api.DeleteParameters(ctx, input)
		if err != nil {
			return result, fmt.Errorf("DeleteParameters failed: %w", err)
		}

		result.DeletedParameters = append(result.DeletedParameters, output.DeletedParameters...)
		result.InvalidParameters = append(result.InvalidParameters, output.InvalidParameters...)
	}

	return result, nil
}

// Ensure time import is used (via ParameterMeta.LastModified in flatParamFromAWS).
var _ = time.Time{}
