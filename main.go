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
		sshAccessKey, err := digitalocean.NewSshKey(ctx, "nomad-server-access-key", &digitalocean.SshKeyArgs{
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

		getSshKeysResult, err := digitalocean.GetSshKeys(ctx, &digitalocean.GetSshKeysArgs{}, pulumi.DependsOn([]pulumi.Resource{sshAccessKey}))
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
		reservedIpAssign, err := digitalocean.NewReservedIpAssignment(ctx, "nomad-server-01-ip-assign", &digitalocean.ReservedIpAssignmentArgs{
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
			Host:           reservedIpAssign.IpAddress,
			User:           pulumi.String("root"),
			PrivateKey:     cfg.RequireSecret("serverAccessPrivateKey"),
			DialErrorLimit: pulumi.Int(60),
		}

		waitForNomadUser, err := remote.NewCommand(ctx, "wait-for-nomad-user", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("until getent passwd nomad; do sleep 0.5; done"),
		}, pulumi.DependsOn([]pulumi.Resource{reservedIpAssign}))
		if err != nil {
			return err
		}

		createEtcNomadDir, err := remote.NewCommand(ctx, "create-etc-nomad-dir", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("mkdir -p /etc/nomad.d"),
		}, pulumi.DependsOn([]pulumi.Resource{reservedIpAssign}))
		if err != nil {
			return err
		}

		createOptNomadDataDir, err := remote.NewCommand(ctx, "create-opt-nomad-data-dir", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("mkdir -p /opt/nomad/data && chown -R nomad:nomad /opt/nomad/data"),
		}, pulumi.DependsOn([]pulumi.Resource{
			reservedIpAssign,
			waitForNomadUser,
		}))
		if err != nil {
			return err
		}

		caCertFile := pulumi.NewFileAsset("./nomad-agent-ca.pem")
		copyCaCert, err := remote.NewCopyToRemote(ctx, "copy-ca-cert", &remote.CopyToRemoteArgs{
			Connection: conn,
			RemotePath: pulumi.String("/etc/nomad.d/nomad-agent-ca.pem"),
			Source:     caCertFile,
			Triggers:   pulumi.Array{caCertFile},
		}, pulumi.DependsOn([]pulumi.Resource{createEtcNomadDir}))
		if err != nil {
			return err
		}

		nomadConfigFile := pulumi.NewFileAsset("./nomad.hcl")
		copyNomadConfig, err := remote.NewCopyToRemote(ctx, "copy-nomad-config", &remote.CopyToRemoteArgs{
			Connection: conn,
			RemotePath: pulumi.String("/etc/nomad.d/nomad.hcl"),
			Source:     nomadConfigFile,
			Triggers:   pulumi.Array{nomadConfigFile},
		}, pulumi.DependsOn([]pulumi.Resource{createEtcNomadDir}))
		if err != nil {
			return err
		}

		nomadServiceConfigFile := pulumi.NewFileAsset("./nomad.service")
		copyNomadServiceConfig, err := remote.NewCopyToRemote(ctx, "copy-nomad-service-config", &remote.CopyToRemoteArgs{
			Connection: conn,
			RemotePath: pulumi.String("/usr/lib/systemd/system/nomad.service"),
			Source:     nomadServiceConfigFile,
			Triggers:   pulumi.Array{nomadServiceConfigFile},
		})
		if err != nil {
			return err
		}

		caCertKeySecret := cfg.RequireSecret("caCertKey")
		copyCaCertKey, err := remote.NewCommand(ctx, "copy-ca-key", &remote.CommandArgs{
			Connection: conn,
			Environment: pulumi.StringMap{
				"LC_CA_KEY": caCertKeySecret,
			},
			Create:   pulumi.String("echo \"$LC_CA_KEY\" > /etc/nomad.d/nomad-agent-ca-key.pem"),
			Triggers: pulumi.Array{caCertKeySecret},
		}, pulumi.DependsOn([]pulumi.Resource{createEtcNomadDir}))
		if err != nil {
			return err
		}

		// Need to chown before creating certificate for the server
		chownEtcNomadDir, err := remote.NewCommand(ctx, "chown-etc-nomad-dir-before-server-cert", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("chown -R nomad:nomad /etc/nomad.d"),
			Triggers: pulumi.Array{
				createEtcNomadDir,
				copyCaCert,
				copyCaCertKey,
				copyNomadConfig,
			},
		}, pulumi.DependsOn([]pulumi.Resource{
			waitForNomadUser,
			createEtcNomadDir,
			copyCaCert,
			copyCaCertKey,
			copyNomadConfig,
		}))
		if err != nil {
			return err
		}

		createServerCert, err := remote.NewCommand(ctx, "create-server-cert", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("cd /etc/nomad.d && rm -f global-server-nomad*.pem && nomad tls cert create -server -additional-dnsname=nomad-server-01.holochain.org"),
			Triggers: pulumi.Array{
				createEtcNomadDir,
				copyCaCert,
				copyCaCertKey,
			},
		}, pulumi.DependsOn([]pulumi.Resource{chownEtcNomadDir}))
		if err != nil {
			return err
		}

		jobRunnerPolicyFile := pulumi.NewFileAsset("./job-runner.policy.hcl")
		copyJobRunnerPolicy, err := remote.NewCopyToRemote(ctx, "copy-job-runner-policy", &remote.CopyToRemoteArgs{
			Connection: conn,
			RemotePath: pulumi.String("/etc/nomad.d/job-runner.policy.hcl"),
			Source:     jobRunnerPolicyFile,
		})
		if err != nil {
			return err
		}

		chownEtcNomadDir, err = remote.NewCommand(ctx, "chown-etc-nomad-dir", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("chown -R nomad:nomad /etc/nomad.d"),
			Triggers: pulumi.Array{
				createEtcNomadDir,
				copyCaCert,
				copyCaCertKey,
				createServerCert,
				copyNomadConfig,
				copyJobRunnerPolicy,
			},
		}, pulumi.DependsOn([]pulumi.Resource{
			waitForNomadUser,
			createEtcNomadDir,
			copyCaCert,
			copyCaCertKey,
			createServerCert,
			copyNomadConfig,
			copyJobRunnerPolicy,
		}))
		if err != nil {
			return err
		}

		enableNomadService, err := remote.NewCommand(ctx, "enable-nomad-service", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("systemctl enable nomad.service"),
		}, pulumi.DependsOn([]pulumi.Resource{
			copyNomadServiceConfig,
			chownEtcNomadDir,
			createOptNomadDataDir,
		}))
		if err != nil {
			return err
		}

		startNomadService, err := remote.NewCommand(ctx, "start-nomad-service", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("systemctl start nomad.service"),
		}, pulumi.DependsOn([]pulumi.Resource{enableNomadService}))
		if err != nil {
			return err
		}

		aclTokenSecret := cfg.RequireSecret("aclBootstrapToken")
		aclBootstrap, err := remote.NewCommand(ctx, "acl-bootstrap", &remote.CommandArgs{
			Connection: conn,
			Environment: pulumi.StringMap{
				"LC_ACL_TOKEN": aclTokenSecret,
			},
			Create:   pulumi.String("echo \"$LC_ACL_TOKEN\" | nomad acl bootstrap -address=https://localhost:4646 -ca-cert=/etc/nomad.d/nomad-agent-ca.pem -"),
			Logging:  remote.LoggingStderr, // Don't log stdout as it contains the token
			Triggers: pulumi.Array{aclTokenSecret},
		},
			pulumi.DependsOn([]pulumi.Resource{startNomadService}),
			pulumi.AdditionalSecretOutputs([]string{"stdout"}), // Hide stdout as it conatins the token
		)
		if err != nil {
			return err
		}

		_, err = remote.NewCommand(ctx, "apply-job-runner-policy", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String("nomad acl policy apply -address=https://localhost:4646 -ca-cert=/etc/nomad.d/nomad-agent-ca.pem -description \"For running jobs and reading Node status in CI workflows\" job-runner /etc/nomad.d/job-runner.policy.hcl"),
			Triggers: pulumi.Array{
				copyJobRunnerPolicy,
				aclBootstrap,
			},
		}, pulumi.DependsOn([]pulumi.Resource{
			copyJobRunnerPolicy,
			aclBootstrap,
		}))
		if err != nil {
			return err
		}

		return nil
	})
}
