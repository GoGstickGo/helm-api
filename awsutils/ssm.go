package awsutils

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SSMPutParameterAPI defines the interface for the PutParameter function.
// We use this interface to test the function using a mock.
type SSM interface {
	GetParameters(ctx context.Context, params *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
}

// Gett SSM Parameters values.
func GetSSMParameters(ctx context.Context, client SSM, parametersMap map[string]string) error {

	var paramNames []string
	for ssmName := range parametersMap {
		paramNames = append(paramNames, ssmName)
	}
	decryption := true
	input := &ssm.GetParametersInput{
		Names:          paramNames,
		WithDecryption: &decryption,
	}

	result, err := client.GetParameters(ctx, input)
	if err != nil {

		return fmt.Errorf("%w", err)
	}

	// Set environment variables using mapping.
	for _, param := range result.Parameters {
		if envName, exists := parametersMap[*param.Name]; exists {
			if err := os.Setenv(envName, *param.Value); err != nil {
				return err
			}
		}
	}

	return nil
}
