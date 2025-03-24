package awsutils

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AWSConfigLoader defines the interface for loading AWS configurations.
type AWSConfigLoader interface {
	Load(ctx context.Context, region string) (aws.Config, error)
}

// RealAWSConfigLoader implements AWSConfigLoader using the actual AWS SDK.
type RealAWSConfigLoader struct{}

// Load loads the AWS configuration using the AWS SDK.
func (r *RealAWSConfigLoader) Load(ctx context.Context, region string) (aws.Config, error) {

	return config.LoadDefaultConfig(ctx, config.WithRegion(region))
}

type AWSClients struct {
	SSM SSM
	// Add other clients as needed.
}

func LoadAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	// Load AWS configuration with the specified region.
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {

		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, nil
}

func InitializeAWSClients(ctx context.Context, loader AWSConfigLoader, region string) (*AWSClients, error) {
	cfg, err := loader.Load(ctx, region)
	if err != nil {

		return nil, err
	}

	clients := &AWSClients{
		SSM: ssm.NewFromConfig(cfg),
		// Initialize other clients.
	}

	return clients, nil
}
