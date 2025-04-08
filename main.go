package main

import (
	"log"
	"os"
	"strconv"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	userData, err := os.ReadFile("cloud-init.yaml")
	if err != nil {
		log.Fatalf("failed to read droplet user data from file: %s", err)
	}

	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		// Add SSH key to be used by Pulumi as part of the setup to DigitalOcean
		_, err := digitalocean.NewSshKey(ctx, "nomad-server-access-key", &digitalocean.SshKeyArgs{
			Name:      pulumi.String("nomad-server-access-key"),
			PublicKey: pulumi.String(cfg.Require("serverAccessPublicKey")),
		})
		if err != nil {
			return err
		}

		region := digitalocean.RegionAMS3
		reservedIp, err := digitalocean.NewReservedIp(ctx, "nomad-server-01-reserved-ip", &digitalocean.ReservedIpArgs{
			Region: region,
		})
		if err != nil {
			return err
		}

		getSshKeysResult, err := digitalocean.GetSshKeys(ctx, &digitalocean.GetSshKeysArgs{}, nil)
		if err != nil {
			return err
		}

		var sshFingerprints []string
		for _, key := range getSshKeysResult.SshKeys {
			sshFingerprints = append(sshFingerprints, key.Fingerprint)
		}

		droplet, err := digitalocean.NewDroplet(ctx, "nomad-server-01", &digitalocean.DropletArgs{
			Image:    pulumi.String("ubuntu-24-04-x64"),
			Name:     pulumi.String("nomad-server-01"),
			Region:   region,
			Size:     digitalocean.DropletSlugDropletS1VCPU512MB10GB,
			Ipv6:     pulumi.Bool(true),
			Tags:     pulumi.StringArray{pulumi.String("nomad")},
			SshKeys:  pulumi.ToStringArray(sshFingerprints),
			UserData: pulumi.String(userData),
		})
		if err != nil {
			return err
		}
		_, err = digitalocean.NewReservedIpAssignment(ctx, "nomad-server-01-ip-assign", &digitalocean.ReservedIpAssignmentArgs{
			IpAddress: reservedIp.IpAddress,
			DropletId: droplet.ID().ApplyT(func(dropletId string) (int, error) {
				id, err := strconv.Atoi(dropletId)
				return id, err
			}).(pulumi.IntInput),
		}, pulumi.DependsOn([]pulumi.Resource{reservedIp, droplet}))
		if err != nil {
			return err
		}

		conn := remote.ConnectionArgs{
			Host:       droplet.Ipv4Address,
			User:       pulumi.String("root"),
			PrivateKey: cfg.RequireSecret("serverAccessPrivateKey"),
		}

		createEtcNomadDir, err := remote.NewCommand(ctx, "create-etc-nomad-dir", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("mkdir -p /etc/nomad.d"),
		}, pulumi.DependsOn([]pulumi.Resource{droplet}))
		if err != nil {
			return err
		}

		copyCaCert, err := remote.NewCopyToRemote(ctx, "copy-ca-cert", &remote.CopyToRemoteArgs{
			Connection: conn,
			RemotePath: pulumi.String("/etc/nomad.d/nomad-agent-ca.pem"),
			Source:     pulumi.NewFileAsset("./nomad-agent-ca.pem"),
		}, pulumi.DependsOn([]pulumi.Resource{createEtcNomadDir}))
		if err != nil {
			return err
		}

		copyNomadConfig, err := remote.NewCopyToRemote(ctx, "copy-nomad-config", &remote.CopyToRemoteArgs{
			Connection: conn,
			RemotePath: pulumi.String("/etc/nomad.d/nomad.hcl"),
			Source:     pulumi.NewFileAsset("./nomad.hcl"),
		}, pulumi.DependsOn([]pulumi.Resource{createEtcNomadDir}))
		if err != nil {
			return err
		}

		_, err = remote.NewCopyToRemote(ctx, "copy-nomad-service-config", &remote.CopyToRemoteArgs{
			Connection: conn,
			RemotePath: pulumi.String("/usr/lib/systemd/system/nomad.service"),
			Source:     pulumi.NewFileAsset("./nomad.service"),
		})
		if err != nil {
			return err
		}

		_, err = remote.NewCommand(ctx, "chown-etc-nomad-dir", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("chown -R nomad:nomad /etc/nomad.d"),
		}, pulumi.DependsOn([]pulumi.Resource{
			createEtcNomadDir,
			copyCaCert,
			copyNomadConfig,
		}))
		if err != nil {
			return err
		}

		return nil
	})
}
