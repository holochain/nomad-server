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

Use the Pulumi DigitalOcean provider to create and manage droplets on DigitalOcean.
The provider documentation can be found at <https://www.pulumi.com/registry/packages/digitalocean>
with the most important function being
[digitalocean.Droplet](https://www.pulumi.com/registry/packages/digitalocean/api-docs/droplet)
which allows the creation and management of droplets.

Then preview the changes with:
```sh
pulumi preview
```

### Applying changes

Simply open a PR to see the preview of the changes in the CI. Then, once the PR
is reviewed and merged into the `main` branch, a new workflow will push the
changes to Pulumi.

## Changing the DigitalOcean token

Pulumi requires a Personal Access Token (PAT) for DigitalOcean to make calls to the API.

Currently the PAT is linked to the `cdunster` DigitalOcean user account. To
change the token, run the following command:
```sh
pulumi config set --secret digitalocean:token <new-token>
```

This value is encrypted by Pulumi and stored in [Pulumi.nomad-server.yaml].

Remember to open a PR with the new token and allow the CI/Actions to apply the
changes to Pulumi.

## Deployment

Some changes to the definition of the DigitalOcean droplet will require the
droplet to be destroyed and re-created.

If this happens or if we want to deliberately re-create the server then the TLS
certificates, TLS keys, and the ACL bootstrap token will all need to be
re-created on the new server.

Most of this is automated with Pulumi, however, there is one manual step to
bootstrap the ACL support and update the token in BitWarden used by developers
to access and manage the server.

First, SSH into the Nomad server, this should be accessible at
`nomad-server-01.holochain.org` and if your SSH key is in DigitalOcean then you
should also have access to is.

```sh
ssh root@nomad-server-01.holochain.org
```

Then to bootstrap ACL run the following:
```console
nomad acl bootstrap -ca-cert=/etc/nomad.d/nomad-agent-ca.pem -address=https://127.0.0.1:4646
```

This should print the ACL bootstrap token details, copy the `Secret ID` into
the `Nomad Server Bootstrap Token` login item in the Holo BitWarden vault.

Now, navigate to <https://nomad-server-01.holochain.org:4646/ui/settings/tokens>
and use the token to login. You should now have full access to the new nomad server.
