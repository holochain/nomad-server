package main

import (
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
)

func main() {
	userData, err := os.ReadFile("cloud-init.yaml")
	if err != nil {
		return
	}

	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := digitalocean.NewDroplet(ctx, "nomad-server", &digitalocean.DropletArgs{
			Image:    pulumi.String("ubuntu-24-04-x64"),
			Name:     pulumi.String("nomad-server-01"),
			Region:   pulumi.String(digitalocean.RegionAMS3),
			Size:     pulumi.String(digitalocean.DropletSlugDropletS1VCPU512MB10GB),
			Ipv6:     pulumi.Bool(true),
			Tags:     pulumi.StringArray{pulumi.String("nomad")},
			UserData: pulumi.String(userData),
		})
		if err != nil {
			return err
		}

		return nil
	})
}

