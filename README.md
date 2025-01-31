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

Finally, select the Pulumi stack that you want to use, for this repo it is `nomad-server`.
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

## Chaning the DigitalOcean token

Pulumi requires a Personal Access Token (PAT) for DigitalOcean to make calls to the API.

Currently the PAT is linked to the `cdunster` DigitalOcean user account. To
change the token, run the following command:
```sh
pulumi config set --secret digitalocean:token <new-token>
```

This value is encrypted by Pulumi and stored in [Pulumi.nomad-server.yaml].
