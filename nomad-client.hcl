data_dir  = "/var/lib/nomad"

acl {
  enabled = true
}

client {
  enabled = true
  servers = ["nomad-server-01.holochain.org"]
}

