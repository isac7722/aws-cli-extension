package ssm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// mockSSMAPI implements SSMAPI for testing.
type mockSSMAPI struct {
	// getParametersByPathPages stores paginated responses keyed by "prefix:token".
	getParametersByPathPages map[string]*ssm.GetParametersByPathOutput
	// getParameterResponses stores GetParameter responses keyed by parameter name.
	getParameterResponses map[string]*ssm.GetParameterOutput
	// putParameterFn is a custom handler for PutParameter calls.
	// When set, it is called instead of the default behavior.
	putParameterFn func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error)
	// deleteParameterFn is a custom handler for DeleteParameter calls.
	deleteParameterFn func(ctx context.Context, params *ssm.DeleteParameterInput) (*ssm.DeleteParameterOutput, error)
	// deleteParametersFn is a custom handler for DeleteParameters (batch) calls.
	deleteParametersFn func(ctx context.Context, params *ssm.DeleteParametersInput) (*ssm.DeleteParametersOutput, error)
	// errors to return
	getParametersByPathErr error
	getParameterErr        error
	deleteParameterErr     error
	deleteParametersErr    error
	// callCount tracks how many times GetParametersByPath was called.
	callCount int
	// putCallCount tracks how many times PutParameter was called.
	putCallCount int
	// deleteCallCount tracks how many times DeleteParameter was called.
	deleteCallCount int
	// deleteParametersCallCount tracks how many times DeleteParameters was called.
	deleteParametersCallCount int
	// lastPutInput stores the most recent PutParameter input for assertions.
	lastPutInput *ssm.PutParameterInput
	// lastDeleteInput stores the most recent DeleteParameter input for assertions.
	lastDeleteInput *ssm.DeleteParameterInput
	// lastDeleteParametersInput stores the most recent DeleteParameters input for assertions.
	lastDeleteParametersInput *ssm.DeleteParametersInput
}

func (m *mockSSMAPI) GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	m.callCount++
	if m.getParametersByPathErr != nil {
		return nil, m.getParametersByPathErr
	}

	token := ""
	if params.NextToken != nil {
		token = *params.NextToken
	}
	key := aws.ToString(params.Path) + ":" + token

	if resp, ok := m.getParametersByPathPages[key]; ok {
		return resp, nil
	}
	// Return empty if no matching page.
	return &ssm.GetParametersByPathOutput{}, nil
}

func (m *mockSSMAPI) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	m.putCallCount++
	m.lastPutInput = params

	if m.putParameterFn != nil {
		return m.putParameterFn(ctx, params)
	}
	// Default: return version 1 with no error.
	return &ssm.PutParameterOutput{Version: 1}, nil
}

func (m *mockSSMAPI) DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	m.deleteCallCount++
	m.lastDeleteInput = params

	if m.deleteParameterFn != nil {
		return m.deleteParameterFn(ctx, params)
	}
	if m.deleteParameterErr != nil {
		return nil, m.deleteParameterErr
	}
	// Default: return success with no error.
	return &ssm.DeleteParameterOutput{}, nil
}

func (m *mockSSMAPI) DeleteParameters(ctx context.Context, params *ssm.DeleteParametersInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParametersOutput, error) {
	m.deleteParametersCallCount++
	m.lastDeleteParametersInput = params

	if m.deleteParametersFn != nil {
		return m.deleteParametersFn(ctx, params)
	}
	if m.deleteParametersErr != nil {
		return nil, m.deleteParametersErr
	}
	// Default: return all names as successfully deleted.
	return &ssm.DeleteParametersOutput{
		DeletedParameters: params.Names,
	}, nil
}

func (m *mockSSMAPI) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.getParameterErr != nil {
		return nil, m.getParameterErr
	}

	name := aws.ToString(params.Name)
	if resp, ok := m.getParameterResponses[name]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("parameter %q not found", name)
}

func newMockParam(name, paramType, value string, version int64, lastMod time.Time) ssmtypes.Parameter {
	return ssmtypes.Parameter{
		Name:             aws.String(name),
		Type:             ssmtypes.ParameterType(paramType),
		Value:            aws.String(value),
		Version:          version,
		ARN:              aws.String(fmt.Sprintf("arn:aws:ssm:us-east-1:123456789:parameter%s", name)),
		DataType:         aws.String("text"),
		LastModifiedDate: &lastMod,
	}
}

func TestListParameters_SinglePage(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mock := &mockSSMAPI{
		getParametersByPathPages: map[string]*ssm.GetParametersByPathOutput{
			"/app:": {
				Parameters: []ssmtypes.Parameter{
					newMockParam("/app/config/db_host", "String", "localhost", 1, now),
					newMockParam("/app/config/db_pass", "SecureString", "****", 2, now),
					newMockParam("/app/api_key", "String", "key123", 1, now),
				},
			},
		},
	}

	client := NewClientWithAPI(mock)
	params, err := client.ListParameters(context.Background(), "/app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(params) != 3 {
		t.Fatalf("got %d params, want 3", len(params))
	}

	// Verify first param
	if params[0].Path != "/app/config/db_host" {
		t.Errorf("params[0].Path = %q, want %q", params[0].Path, "/app/config/db_host")
	}
	if params[0].Meta.Type != "String" {
		t.Errorf("params[0].Meta.Type = %q, want %q", params[0].Meta.Type, "String")
	}
	if params[0].Meta.Version != 1 {
		t.Errorf("params[0].Meta.Version = %d, want 1", params[0].Meta.Version)
	}
	if !params[0].Meta.LastModified.Equal(now) {
		t.Errorf("params[0].Meta.LastModified = %v, want %v", params[0].Meta.LastModified, now)
	}

	// Verify SecureString
	if params[1].Meta.Type != "SecureString" {
		t.Errorf("params[1].Meta.Type = %q, want %q", params[1].Meta.Type, "SecureString")
	}

	// Verify single API call (no pagination)
	if mock.callCount != 1 {
		t.Errorf("API called %d times, want 1", mock.callCount)
	}
}

func TestListParameters_Paginated(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mock := &mockSSMAPI{
		getParametersByPathPages: map[string]*ssm.GetParametersByPathOutput{
			"/app:": {
				Parameters: []ssmtypes.Parameter{
					newMockParam("/app/param1", "String", "val1", 1, now),
					newMockParam("/app/param2", "String", "val2", 1, now),
				},
				NextToken: aws.String("page2"),
			},
			"/app:page2": {
				Parameters: []ssmtypes.Parameter{
					newMockParam("/app/param3", "String", "val3", 1, now),
				},
				// No NextToken = last page
			},
		},
	}

	client := NewClientWithAPI(mock)
	params, err := client.ListParameters(context.Background(), "/app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(params) != 3 {
		t.Fatalf("got %d params, want 3", len(params))
	}

	// Verify all pages were fetched
	if mock.callCount != 2 {
		t.Errorf("API called %d times, want 2 (paginated)", mock.callCount)
	}

	// Verify third param from second page
	if params[2].Path != "/app/param3" {
		t.Errorf("params[2].Path = %q, want %q", params[2].Path, "/app/param3")
	}
}

func TestListParameters_Empty(t *testing.T) {
	mock := &mockSSMAPI{
		getParametersByPathPages: map[string]*ssm.GetParametersByPathOutput{
			"/:": {},
		},
	}

	client := NewClientWithAPI(mock)
	params, err := client.ListParameters(context.Background(), "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(params) != 0 {
		t.Fatalf("got %d params, want 0", len(params))
	}
}

func TestListParameters_APIError(t *testing.T) {
	mock := &mockSSMAPI{
		getParametersByPathErr: fmt.Errorf("access denied"),
	}

	client := NewClientWithAPI(mock)
	_, err := client.ListParameters(context.Background(), "/app")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if want := `GetParametersByPath failed for "/app": access denied`; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestGetParameterValue(t *testing.T) {
	mock := &mockSSMAPI{
		getParameterResponses: map[string]*ssm.GetParameterOutput{
			"/app/secret": {
				Parameter: &ssmtypes.Parameter{
					Name:  aws.String("/app/secret"),
					Type:  ssmtypes.ParameterTypeSecureString,
					Value: aws.String("my-secret-value"),
				},
			},
		},
	}

	client := NewClientWithAPI(mock)
	val, err := client.GetParameterValue(context.Background(), "/app/secret", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "my-secret-value" {
		t.Errorf("value = %q, want %q", val, "my-secret-value")
	}
}

func TestGetParameterValue_Error(t *testing.T) {
	mock := &mockSSMAPI{
		getParameterErr: fmt.Errorf("not found"),
	}

	client := NewClientWithAPI(mock)
	_, err := client.GetParameterValue(context.Background(), "/missing", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetParameter(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	t.Run("returns full metadata for String parameter", func(t *testing.T) {
		mock := &mockSSMAPI{
			getParameterResponses: map[string]*ssm.GetParameterOutput{
				"/app/db_host": {
					Parameter: &ssmtypes.Parameter{
						Name:             aws.String("/app/db_host"),
						Type:             ssmtypes.ParameterTypeString,
						Value:            aws.String("prod-db.example.com"),
						Version:          5,
						ARN:              aws.String("arn:aws:ssm:us-east-1:123456789:parameter/app/db_host"),
						DataType:         aws.String("text"),
						LastModifiedDate: &now,
					},
				},
			},
		}

		client := NewClientWithAPI(mock)
		result, err := client.GetParameter(context.Background(), "/app/db_host", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Name != "/app/db_host" {
			t.Errorf("Name = %q, want %q", result.Name, "/app/db_host")
		}
		if result.Value != "prod-db.example.com" {
			t.Errorf("Value = %q, want %q", result.Value, "prod-db.example.com")
		}
		if result.Type != "String" {
			t.Errorf("Type = %q, want %q", result.Type, "String")
		}
		if result.Version != 5 {
			t.Errorf("Version = %d, want 5", result.Version)
		}
		if result.ARN != "arn:aws:ssm:us-east-1:123456789:parameter/app/db_host" {
			t.Errorf("ARN = %q, want %q", result.ARN, "arn:aws:ssm:us-east-1:123456789:parameter/app/db_host")
		}
		if !result.LastModified.Equal(now) {
			t.Errorf("LastModified = %v, want %v", result.LastModified, now)
		}
	})

	t.Run("returns decrypted SecureString value", func(t *testing.T) {
		mock := &mockSSMAPI{
			getParameterResponses: map[string]*ssm.GetParameterOutput{
				"/app/secret": {
					Parameter: &ssmtypes.Parameter{
						Name:             aws.String("/app/secret"),
						Type:             ssmtypes.ParameterTypeSecureString,
						Value:            aws.String("decrypted-secret"),
						Version:          3,
						ARN:              aws.String("arn:aws:ssm:us-east-1:123456789:parameter/app/secret"),
						LastModifiedDate: &now,
					},
				},
			},
		}

		client := NewClientWithAPI(mock)
		result, err := client.GetParameter(context.Background(), "/app/secret", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Type != "SecureString" {
			t.Errorf("Type = %q, want %q", result.Type, "SecureString")
		}
		if result.Value != "decrypted-secret" {
			t.Errorf("Value = %q, want %q", result.Value, "decrypted-secret")
		}
	})

	t.Run("returns error for missing parameter", func(t *testing.T) {
		mock := &mockSSMAPI{
			getParameterErr: fmt.Errorf("ParameterNotFound: parameter /missing not found"),
		}

		client := NewClientWithAPI(mock)
		_, err := client.GetParameter(context.Background(), "/missing", false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := `GetParameter failed for "/missing": ParameterNotFound: parameter /missing not found`; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("returns error for nil parameter in response", func(t *testing.T) {
		mock := &mockSSMAPI{
			getParameterResponses: map[string]*ssm.GetParameterOutput{
				"/app/empty": {
					Parameter: nil,
				},
			},
		}

		client := NewClientWithAPI(mock)
		_, err := client.GetParameter(context.Background(), "/app/empty", false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles nil value and lastModified gracefully", func(t *testing.T) {
		mock := &mockSSMAPI{
			getParameterResponses: map[string]*ssm.GetParameterOutput{
				"/app/nil_value": {
					Parameter: &ssmtypes.Parameter{
						Name:    aws.String("/app/nil_value"),
						Type:    ssmtypes.ParameterTypeString,
						Value:   nil,
						Version: 1,
						ARN:     aws.String("arn:aws:ssm:us-east-1:123456789:parameter/app/nil_value"),
					},
				},
			},
		}

		client := NewClientWithAPI(mock)
		result, err := client.GetParameter(context.Background(), "/app/nil_value", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Value != "" {
			t.Errorf("Value = %q, want empty string", result.Value)
		}
		if !result.LastModified.IsZero() {
			t.Errorf("LastModified = %v, want zero time", result.LastModified)
		}
	})
}

func TestGetParameterDetail(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mock := &mockSSMAPI{
		getParameterResponses: map[string]*ssm.GetParameterOutput{
			"/app/db_host": {
				Parameter: &ssmtypes.Parameter{
					Name:             aws.String("/app/db_host"),
					Type:             ssmtypes.ParameterTypeString,
					Value:            aws.String("prod-db.example.com"),
					Version:          5,
					ARN:              aws.String("arn:aws:ssm:us-east-1:123456789:parameter/app/db_host"),
					DataType:         aws.String("text"),
					LastModifiedDate: &now,
				},
			},
		},
	}

	client := NewClientWithAPI(mock)
	detail, err := client.GetParameterDetail(context.Background(), "/app/db_host", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.Path != "/app/db_host" {
		t.Errorf("Path = %q, want %q", detail.Path, "/app/db_host")
	}
	if detail.Value != "prod-db.example.com" {
		t.Errorf("Value = %q, want %q", detail.Value, "prod-db.example.com")
	}
	if detail.Meta.Type != "String" {
		t.Errorf("Meta.Type = %q, want %q", detail.Meta.Type, "String")
	}
	if detail.Meta.Version != 5 {
		t.Errorf("Meta.Version = %d, want 5", detail.Meta.Version)
	}
	if !detail.Meta.LastModified.Equal(now) {
		t.Errorf("Meta.LastModified = %v, want %v", detail.Meta.LastModified, now)
	}
}

func TestListParametersFiltered(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mock := &mockSSMAPI{
		getParametersByPathPages: map[string]*ssm.GetParametersByPathOutput{
			"/:": {
				Parameters: []ssmtypes.Parameter{
					newMockParam("/app/config/db_host", "String", "localhost", 1, now),
					newMockParam("/app/config/db_pass", "SecureString", "****", 2, now),
					newMockParam("/infra/vpc_id", "String", "vpc-123", 1, now),
				},
			},
		},
	}

	client := NewClientWithAPI(mock)

	// Filter by "db"
	params, err := client.ListParametersFiltered(context.Background(), "/", "db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params) != 2 {
		t.Fatalf("got %d params, want 2 (filtered by 'db')", len(params))
	}

	// Filter by "vpc" (case-insensitive)
	params, err = client.ListParametersFiltered(context.Background(), "/", "VPC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params) != 1 {
		t.Fatalf("got %d params, want 1 (filtered by 'VPC')", len(params))
	}

	// Empty filter returns all
	params, err = client.ListParametersFiltered(context.Background(), "/", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params) != 3 {
		t.Fatalf("got %d params, want 3 (no filter)", len(params))
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "/"},
		{"/", "/"},
		{"/app", "/app"},
		{"/app/", "/app"},
		{"app", "/app"},
		{"app/config/", "/app/config"},
		{"/app/config", "/app/config"},
	}
	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListParameters_IntegrationWithBuildTree(t *testing.T) {
	// Verify that ListParameters output feeds directly into BuildTree.
	now := time.Now().Truncate(time.Second)
	mock := &mockSSMAPI{
		getParametersByPathPages: map[string]*ssm.GetParametersByPathOutput{
			"/:": {
				Parameters: []ssmtypes.Parameter{
					newMockParam("/app/config/db_host", "String", "localhost", 1, now),
					newMockParam("/app/config/db_pass", "SecureString", "****", 2, now),
					newMockParam("/app/api_key", "String", "key123", 1, now),
					newMockParam("/infra/vpc_id", "String", "vpc-123", 1, now),
				},
			},
		},
	}

	client := NewClientWithAPI(mock)
	params, err := client.ListParameters(context.Background(), "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Feed into BuildTree
	root := BuildTree(params)

	if root.ParameterCount() != 4 {
		t.Errorf("tree has %d parameters, want 4", root.ParameterCount())
	}
	if root.ChildCount() != 2 {
		t.Errorf("root has %d children, want 2 (app, infra)", root.ChildCount())
	}

	// Verify hierarchy
	app := root.FindChild("app")
	if app == nil {
		t.Fatal("expected 'app' folder in tree")
	}
	configFolder := app.FindChild("config")
	if configFolder == nil {
		t.Fatal("expected 'config' folder under 'app'")
	}
	if configFolder.ChildCount() != 2 {
		t.Errorf("config has %d children, want 2", configFolder.ChildCount())
	}

	// Verify SecureString metadata preserved through the pipeline
	dbPass := configFolder.FindChild("db_pass")
	if dbPass == nil {
		t.Fatal("expected 'db_pass' parameter")
	}
	if !dbPass.IsSecureString() {
		t.Error("db_pass should be SecureString")
	}
}

func TestPutParameter(t *testing.T) {
	t.Run("creates String parameter and returns version", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				return &ssm.PutParameterOutput{Version: 1}, nil
			},
		}

		client := NewClientWithAPI(mock)
		version, err := client.PutParameter(context.Background(), PutParameterInput{
			Name:      "/app/config/db_host",
			Value:     "mydb.example.com",
			Type:      "String",
			Overwrite: false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != 1 {
			t.Errorf("version = %d, want 1", version)
		}
		if mock.putCallCount != 1 {
			t.Errorf("PutParameter called %d times, want 1", mock.putCallCount)
		}

		// Verify the input was passed correctly to the SDK.
		if aws.ToString(mock.lastPutInput.Name) != "/app/config/db_host" {
			t.Errorf("Name = %q, want %q", aws.ToString(mock.lastPutInput.Name), "/app/config/db_host")
		}
		if aws.ToString(mock.lastPutInput.Value) != "mydb.example.com" {
			t.Errorf("Value = %q, want %q", aws.ToString(mock.lastPutInput.Value), "mydb.example.com")
		}
		if mock.lastPutInput.Type != ssmtypes.ParameterTypeString {
			t.Errorf("Type = %v, want %v", mock.lastPutInput.Type, ssmtypes.ParameterTypeString)
		}
		if aws.ToBool(mock.lastPutInput.Overwrite) != false {
			t.Errorf("Overwrite = %v, want false", aws.ToBool(mock.lastPutInput.Overwrite))
		}
	})

	t.Run("overwrites existing parameter and returns new version", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				// Simulate a version bump on overwrite.
				return &ssm.PutParameterOutput{Version: 5}, nil
			},
		}

		client := NewClientWithAPI(mock)
		version, err := client.PutParameter(context.Background(), PutParameterInput{
			Name:        "/app/config/db_host",
			Value:       "newhost.example.com",
			Type:        "String",
			Description: "Database hostname",
			Overwrite:   true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != 5 {
			t.Errorf("version = %d, want 5", version)
		}

		// Verify overwrite flag was set.
		if aws.ToBool(mock.lastPutInput.Overwrite) != true {
			t.Errorf("Overwrite = %v, want true", aws.ToBool(mock.lastPutInput.Overwrite))
		}
		// Verify description was passed.
		if aws.ToString(mock.lastPutInput.Description) != "Database hostname" {
			t.Errorf("Description = %q, want %q", aws.ToString(mock.lastPutInput.Description), "Database hostname")
		}
	})

	t.Run("creates SecureString parameter", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				return &ssm.PutParameterOutput{Version: 1}, nil
			},
		}

		client := NewClientWithAPI(mock)
		version, err := client.PutParameter(context.Background(), PutParameterInput{
			Name:      "/app/secrets/api_key",
			Value:     "super-secret",
			Type:      "SecureString",
			Overwrite: false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != 1 {
			t.Errorf("version = %d, want 1", version)
		}
		if mock.lastPutInput.Type != ssmtypes.ParameterTypeSecureString {
			t.Errorf("Type = %v, want %v", mock.lastPutInput.Type, ssmtypes.ParameterTypeSecureString)
		}
	})

	t.Run("returns error for invalid parameter type", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)
		_, err := client.PutParameter(context.Background(), PutParameterInput{
			Name:  "/app/config/key",
			Value: "val",
			Type:  "InvalidType",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := `invalid parameter type "InvalidType"`; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
		// Ensure PutParameter was never called on the API.
		if mock.putCallCount != 0 {
			t.Errorf("PutParameter called %d times, want 0 (should fail before API call)", mock.putCallCount)
		}
	})

	t.Run("returns API error with context", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				return nil, fmt.Errorf("ParameterAlreadyExists: The parameter already exists")
			},
		}

		client := NewClientWithAPI(mock)
		_, err := client.PutParameter(context.Background(), PutParameterInput{
			Name:      "/app/config/existing",
			Value:     "val",
			Type:      "String",
			Overwrite: false,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := `PutParameter failed for "/app/config/existing"`; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
		if want := "ParameterAlreadyExists"; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain original error %q", err.Error(), want)
		}
	})

	t.Run("returns access denied error", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				return nil, fmt.Errorf("AccessDeniedException: User is not authorized to perform ssm:PutParameter")
			},
		}

		client := NewClientWithAPI(mock)
		_, err := client.PutParameter(context.Background(), PutParameterInput{
			Name:      "/app/config/key",
			Value:     "val",
			Type:      "String",
			Overwrite: true,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "AccessDeniedException"; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
	})

	t.Run("omits description when empty", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				return &ssm.PutParameterOutput{Version: 1}, nil
			},
		}

		client := NewClientWithAPI(mock)
		_, err := client.PutParameter(context.Background(), PutParameterInput{
			Name:        "/app/config/key",
			Value:       "val",
			Type:        "String",
			Description: "",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.lastPutInput.Description != nil {
			t.Errorf("Description should be nil when empty, got %q", aws.ToString(mock.lastPutInput.Description))
		}
	})
}

func TestUpdateParameter(t *testing.T) {
	t.Run("updates parameter with overwrite enabled", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				// Verify overwrite is always true for updates.
				if !aws.ToBool(params.Overwrite) {
					t.Error("UpdateParameter must always set Overwrite=true")
				}
				return &ssm.PutParameterOutput{Version: 3}, nil
			},
		}

		client := NewClientWithAPI(mock)
		version, err := client.UpdateParameter(context.Background(), UpdateParameterInput{
			Name:  "/app/config/db_host",
			Value: "updated-host.example.com",
			Type:  "String",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != 3 {
			t.Errorf("version = %d, want 3", version)
		}
		if mock.putCallCount != 1 {
			t.Errorf("PutParameter called %d times, want 1", mock.putCallCount)
		}
	})

	t.Run("updates SecureString parameter and returns new version", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				return &ssm.PutParameterOutput{Version: 7}, nil
			},
		}

		client := NewClientWithAPI(mock)
		version, err := client.UpdateParameter(context.Background(), UpdateParameterInput{
			Name:        "/app/secrets/api_key",
			Value:       "new-secret-value",
			Type:        "SecureString",
			Description: "Updated API key",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != 7 {
			t.Errorf("version = %d, want 7", version)
		}
		if mock.lastPutInput.Type != ssmtypes.ParameterTypeSecureString {
			t.Errorf("Type = %v, want %v", mock.lastPutInput.Type, ssmtypes.ParameterTypeSecureString)
		}
		if aws.ToString(mock.lastPutInput.Description) != "Updated API key" {
			t.Errorf("Description = %q, want %q", aws.ToString(mock.lastPutInput.Description), "Updated API key")
		}
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)
		_, err := client.UpdateParameter(context.Background(), UpdateParameterInput{
			Name:  "",
			Value: "val",
			Type:  "String",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "parameter name is required"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
		if mock.putCallCount != 0 {
			t.Errorf("PutParameter should not be called, got %d calls", mock.putCallCount)
		}
	})

	t.Run("returns error when value is empty", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)
		_, err := client.UpdateParameter(context.Background(), UpdateParameterInput{
			Name:  "/app/key",
			Value: "",
			Type:  "String",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "parameter value is required"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("returns error when type is empty", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)
		_, err := client.UpdateParameter(context.Background(), UpdateParameterInput{
			Name:  "/app/key",
			Value: "val",
			Type:  "",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "parameter type is required"; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
	})

	t.Run("propagates API errors", func(t *testing.T) {
		mock := &mockSSMAPI{
			putParameterFn: func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
				return nil, fmt.Errorf("ThrottlingException: Rate exceeded")
			},
		}

		client := NewClientWithAPI(mock)
		_, err := client.UpdateParameter(context.Background(), UpdateParameterInput{
			Name:  "/app/config/key",
			Value: "val",
			Type:  "String",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "ThrottlingException"; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
	})
}

func TestDeleteParameter(t *testing.T) {
	t.Run("deletes parameter successfully", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)

		err := client.DeleteParameter(context.Background(), "/app/config/old_key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.deleteCallCount != 1 {
			t.Errorf("DeleteParameter called %d times, want 1", mock.deleteCallCount)
		}
		if aws.ToString(mock.lastDeleteInput.Name) != "/app/config/old_key" {
			t.Errorf("Name = %q, want %q", aws.ToString(mock.lastDeleteInput.Name), "/app/config/old_key")
		}
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)

		err := client.DeleteParameter(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "parameter name is required"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
		if mock.deleteCallCount != 0 {
			t.Errorf("DeleteParameter should not be called, got %d calls", mock.deleteCallCount)
		}
	})

	t.Run("propagates API errors", func(t *testing.T) {
		mock := &mockSSMAPI{
			deleteParameterErr: fmt.Errorf("ParameterNotFound: parameter not found"),
		}
		client := NewClientWithAPI(mock)

		err := client.DeleteParameter(context.Background(), "/app/config/missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "ParameterNotFound"; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
	})

	t.Run("uses custom delete function", func(t *testing.T) {
		var calledWith string
		mock := &mockSSMAPI{
			deleteParameterFn: func(ctx context.Context, params *ssm.DeleteParameterInput) (*ssm.DeleteParameterOutput, error) {
				calledWith = aws.ToString(params.Name)
				return &ssm.DeleteParameterOutput{}, nil
			},
		}
		client := NewClientWithAPI(mock)

		err := client.DeleteParameter(context.Background(), "/app/test/param")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledWith != "/app/test/param" {
			t.Errorf("delete called with %q, want %q", calledWith, "/app/test/param")
		}
	})
}

func TestDeleteParameters(t *testing.T) {
	t.Run("deletes multiple parameters successfully", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)

		names := []string{"/app/config/key1", "/app/config/key2", "/app/config/key3"}
		result, err := client.DeleteParameters(context.Background(), names)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.DeletedParameters) != 3 {
			t.Errorf("DeletedParameters count = %d, want 3", len(result.DeletedParameters))
		}
		if len(result.InvalidParameters) != 0 {
			t.Errorf("InvalidParameters count = %d, want 0", len(result.InvalidParameters))
		}
		if mock.deleteParametersCallCount != 1 {
			t.Errorf("DeleteParameters called %d times, want 1", mock.deleteParametersCallCount)
		}
	})

	t.Run("returns error when no names provided", func(t *testing.T) {
		mock := &mockSSMAPI{}
		client := NewClientWithAPI(mock)

		_, err := client.DeleteParameters(context.Background(), []string{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "at least one parameter name is required"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
		if mock.deleteParametersCallCount != 0 {
			t.Errorf("DeleteParameters should not be called, got %d calls", mock.deleteParametersCallCount)
		}
	})

	t.Run("reports invalid parameters", func(t *testing.T) {
		mock := &mockSSMAPI{
			deleteParametersFn: func(ctx context.Context, params *ssm.DeleteParametersInput) (*ssm.DeleteParametersOutput, error) {
				return &ssm.DeleteParametersOutput{
					DeletedParameters: []string{"/app/config/key1"},
					InvalidParameters: []string{"/app/config/missing"},
				}, nil
			},
		}
		client := NewClientWithAPI(mock)

		result, err := client.DeleteParameters(context.Background(), []string{"/app/config/key1", "/app/config/missing"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.DeletedParameters) != 1 {
			t.Errorf("DeletedParameters count = %d, want 1", len(result.DeletedParameters))
		}
		if len(result.InvalidParameters) != 1 {
			t.Errorf("InvalidParameters count = %d, want 1", len(result.InvalidParameters))
		}
		if result.InvalidParameters[0] != "/app/config/missing" {
			t.Errorf("InvalidParameters[0] = %q, want %q", result.InvalidParameters[0], "/app/config/missing")
		}
	})

	t.Run("chunks more than 10 parameters into multiple calls", func(t *testing.T) {
		callCount := 0
		mock := &mockSSMAPI{
			deleteParametersFn: func(ctx context.Context, params *ssm.DeleteParametersInput) (*ssm.DeleteParametersOutput, error) {
				callCount++
				return &ssm.DeleteParametersOutput{
					DeletedParameters: params.Names,
				}, nil
			},
		}
		client := NewClientWithAPI(mock)

		// Create 15 parameter names (should result in 2 API calls: 10 + 5)
		names := make([]string, 15)
		for i := range names {
			names[i] = fmt.Sprintf("/app/param%d", i)
		}

		result, err := client.DeleteParameters(context.Background(), names)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 2 {
			t.Errorf("DeleteParameters called %d times, want 2 (chunks of 10+5)", callCount)
		}
		if len(result.DeletedParameters) != 15 {
			t.Errorf("DeletedParameters count = %d, want 15", len(result.DeletedParameters))
		}
	})

	t.Run("propagates API errors", func(t *testing.T) {
		mock := &mockSSMAPI{
			deleteParametersErr: fmt.Errorf("AccessDeniedException: not authorized"),
		}
		client := NewClientWithAPI(mock)

		_, err := client.DeleteParameters(context.Background(), []string{"/app/key"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "AccessDeniedException"; !contains(err.Error(), want) {
			t.Errorf("error = %q, want to contain %q", err.Error(), want)
		}
	})
}

func TestParseParameterType(t *testing.T) {
	tests := []struct {
		input   string
		want    ssmtypes.ParameterType
		wantErr bool
	}{
		{"String", ssmtypes.ParameterTypeString, false},
		{"StringList", ssmtypes.ParameterTypeStringList, false},
		{"SecureString", ssmtypes.ParameterTypeSecureString, false},
		{"string", "", true},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseParameterType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseParameterType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseParameterType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// contains checks if s contains substr (helper for error message assertions).
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
