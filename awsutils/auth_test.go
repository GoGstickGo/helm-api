package awsutils_test

import (
	"context"
	"errors"
	"helm-api/awsutils"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAWSConfigLoader is a mock implementation of AWSConfigLoader for testing.
type MockAWSConfigLoader struct {
	mock.Mock
}

// Load mocks the Load method of AWSConfigLoader.
func (m *MockAWSConfigLoader) Load(ctx context.Context, region string) (aws.Config, error) {
	args := m.Called(ctx, region)

	return args.Get(0).(aws.Config), args.Error(1)
}

func TestInitializeAWSClients_Success(t *testing.T) {
	t.Parallel()
	// Create a mock config loader.
	mockLoader := new(MockAWSConfigLoader)

	// Define expected aws.Config.
	expectedConfig := aws.Config{}

	// Setup expectations.
	mockLoader.On("Load", mock.Anything, "us-west-2").Return(expectedConfig, nil)

	// Call InitializeAWSClients.
	clients, err := awsutils.InitializeAWSClients(context.Background(), mockLoader, "us-west-2")

	// Assertions.
	require.NoError(t, err)
	assert.NotNil(t, clients)
	assert.IsType(t, &ssm.Client{}, clients.SSM)

	// Verify that all expectations were met.
	mockLoader.AssertExpectations(t)
}

func TestInitializeAWSClients_LoadConfigError(t *testing.T) {
	t.Parallel()
	// Create a mock config loader.
	mockLoader := new(MockAWSConfigLoader)

	// Setup expectations to return an error.
	mockLoader.On("Load", mock.Anything, "us-west-2").Return(aws.Config{}, errors.New("failed to load AWS config"))

	// Call InitializeAWSClients.
	clients, err := awsutils.InitializeAWSClients(context.Background(), mockLoader, "us-west-2")

	// Assertions
	require.Error(t, err)
	assert.Nil(t, clients)
	assert.Contains(t, err.Error(), "failed to load AWS config")

	// Verify that all expectations were met.
	mockLoader.AssertExpectations(t)
}
