# nomad-server

A Pulumi definition for deploying a cluster of Nomad servers as DigitalOcean droplets.

## Development

### Installation

First, make sure that you are in the Nix development shell or that you have
installed `pulumi`, `pulumi-language-go`, and `go`.

Then, log into Pulumi with:

```sh
pulumi login
```

Next, set the default organisation to `holochain` with:

```sh
pulumi org set-default holochain
```

Finally, select the Pulumi stack that you want to use. For this repo it is `nomad-server`.

```sh
pulumi stack select nomad-server
```

### Making changes

Modify the [Go code](main.go) to make changes to the workflow using any
available Pulumi providers.

The Pulumi DigitalOcean provider can create and manage droplets, SSH keys, and
reserved IP addresses on DigitalOcean.
The documentation for it can be found at <https://www.pulumi.com/registry/packages/digitalocean>
with the most important function being [digitalocean.Droplet](https://www.pulumi.com/registry/packages/digitalocean/api-docs/droplet)
which allows the creation and management of droplets.

You can also use the SSH key created and managed by this Pulumi stack to send
commands and copy files to the droplet using the `pulumi-remote` provider, see
<https://www.pulumi.com/registry/packages/command/api-docs/remote>.

Then preview the changes with:

```sh
pulumi preview
```

### Applying changes

Simply open a PR to see the preview of the changes in the CI. Then, once the PR
is reviewed and merged into the `main` branch, a new workflow will push the
changes to Pulumi.

## Changing the DigitalOcean token

Pulumi requires a Personal Access Token (PAT) for DigitalOcean to make calls to
the API.

Currently the PAT is linked to the `cdunster` DigitalOcean user account. To
change the token, run the following command:

```sh
pulumi config set --secret digitalocean:token <new-token>
```

This value is encrypted by Pulumi and stored in
[Pulumi.nomad-server.yaml](Pulumi.nomad-server.yaml).

Remember to open a PR with the new token and allow the CI/Actions to apply the
changes to Pulumi.

## Changing the ACL bootstrap token

Nomad requires a token for bootstrapping ACL. This token should be shared
across all servers on the cluster, thus it is managed by Pulumi.

This token should also be stored in the Holochain shared vault of the password
manager service so that all developers can access the admin token.

Update the token with:

```sh
pulumi config set --secret aclBootstrapToken <new-token>
```

This value is encrypted by Pulumi and stored in
[Pulumi.nomad-server.yaml](Pulumi.nomad-server.yaml).

Remember to open a PR with the new token and allow the CI/Actions to apply the
changes to Pulumi.

## Changing the TLS certificate authority certificate

Nomad uses TLS for encrypted communications with the server(s) for this to work
there needs to be a main CA certificate that is used to generate the
certificates used by the server(s).

A new certificate-key pair can be generated with:

```sh
nomad tls ca create
```

The public certificate should be stored as
[nomad-agent-ca.pem](nomad-agent-ca.pem) which will then be coppied over to the
servers as part of this workflow. The key should also be saved in this stack
but as a secret.

To update the key from the generated file, use:

```sh
cat nomad-agent-ca-key.pem | pulumi config set --secret caCertKey
```

This value is encrypted by Pulumi and stored in
[Pulumi.nomad-server.yaml](Pulumi.nomad-server.yaml).

Remember to open a PR with the new token and allow the CI/Actions to apply the
changes to Pulumi.

## Changing the SSH key-pair

Pulumi uses the SSH key stored in this stack to send commands and files to the
droplet after it has been created. This allows the automated setup of Nomad
with the required configuration.

This is just a standard SSH key that can be generated with `ssh-keygen`.

Both the public and private parts of the key are stored in [this stack's
config](Pulumi.nomad-server.yaml), the public part is stored as plain text and
the private is encrypted.

To replace the SSH key, generate a new key with:

```sh
ssh-keygen
```

Then, set the public part in the config with:

```sh
cat <key-name>.pub | pulumi config set serverAccessPublicKey
```

Finally, set the private part in the config with:

```sh
cat <key-name> | pulumi config set --secret serverAccessPrivateKey
```

Remember to open a PR with the new token and allow the CI/Actions to apply the
changes to Pulumi.

## Deployment

Some changes to the definition of the DigitalOcean droplet will require the
droplet to be destroyed and re-created.

If this happens, or if we want to deliberately re-create the server(s), then
then this workflow should handle the full setup of the Nomad server(s).
However, any users of Nomad will need to re-generate a new ACL token using the
policies that they need. This can be done through Nomad's web UI.

If a new ACL token is created and used then it must be updated in the Holochain
shared vault of the password manager.

To test the admin token in the password manager, navigate to
<https://nomad-server-01.holochain.org:4646/ui/settings/tokens> and use the
token to login. You should now have full access to the new nomad server.
