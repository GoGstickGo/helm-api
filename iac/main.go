package main

import (
	"fmt"

	"github.com/dirien/pulumi-vultr/sdk/v2/go/vultr"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {

	pulumi.Run(func(ctx *pulumi.Context) error {
		ctx.Log.Debug("Activating power conduit", nil)
		_, err := vultr.NewContainerRegistry(ctx, "vcr1", &vultr.ContainerRegistryArgs{
			Name:   pulumi.String("activatedpowerconduit"),
			Plan:   pulumi.String("start_up"),
			Public: pulumi.Bool(false),
			Region: pulumi.String("fra"),
		})

		if err != nil {
			ctx.Log.Error(fmt.Sprintf("Error: %s", err), nil)
		}
		return err
	})
}
