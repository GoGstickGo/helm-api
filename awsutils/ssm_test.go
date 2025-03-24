package awsutils_test

import (
	"context"
	"helm-api/awsutils"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSSM is a mock implementation of the SSM interface
type MockSSM struct {
	mock.Mock
}

func (m *MockSSM) GetParameters(ctx context.Context, input *ssm.GetParametersInput, opts ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(*ssm.GetParametersOutput), args.Error(1)
}

func TestGetSSMParameters(t *testing.T) {
	tests := []struct {
		name          string
		parametersMap map[string]string
		mockResponse  *ssm.GetParametersOutput
		mockError     error
		expectedEnv   map[string]string
		expectError   bool
	}{
		{
			name: "successful parameter fetch",
			parametersMap: map[string]string{
				"/test/param1": "ENV_PARAM1",
				"/test/param2": "ENV_PARAM2",
			},
			mockResponse: &ssm.GetParametersOutput{
				Parameters: []types.Parameter{
					{
						Name:  stringPtr("/test/param1"),
						Value: stringPtr("value1"),
					},
					{
						Name:  stringPtr("/test/param2"),
						Value: stringPtr("value2"),
					},
				},
			},
			mockError: nil,
			expectedEnv: map[string]string{
				"ENV_PARAM1": "value1",
				"ENV_PARAM2": "value2",
			},
			expectError: false,
		},
		{
			name: "AWS service error",
			parametersMap: map[string]string{
				"/test/param1": "ENV_PARAM1",
			},
			mockResponse: &ssm.GetParametersOutput{},
			mockError:    assert.AnError,
			expectedEnv:  map[string]string{},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSSM)
			ctx := context.Background()

			// Clear environment variables
			for _, envName := range tt.parametersMap {
				os.Unsetenv(envName)
			}

			// Create the input we expect based on the parameter map
			var expectedParamNames []string
			for name := range tt.parametersMap {
				expectedParamNames = append(expectedParamNames, name)
			}
			decryption := true
			expectedInput := &ssm.GetParametersInput{
				Names:          expectedParamNames,
				WithDecryption: &decryption,
			}

			// Set up the expected call
			mockClient.On("GetParameters", ctx, expectedInput, mock.Anything).Return(tt.mockResponse, tt.mockError)

			// Call the function
			err := awsutils.GetSSMParameters(ctx, mockClient, tt.parametersMap)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Check that environment variables were set correctly
				for envName, expectedValue := range tt.expectedEnv {
					actualValue, exists := os.LookupEnv(envName)
					assert.True(t, exists, "Environment variable %s should be set", envName)
					assert.Equal(t, expectedValue, actualValue, "Environment variable %s has incorrect value", envName)
				}
			}

			// Verify all expected calls were made
			mockClient.AssertExpectations(t)
		})
	}
}

// Helper function to get string pointer
func stringPtr(s string) *string {
	return &s
}
