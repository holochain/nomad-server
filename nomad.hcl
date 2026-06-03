data_dir = "/opt/nomad/data"

acl {
  enabled = true
}

advertise {
  http = "nomad-server-01.holochain.org"
  rpc  = "nomad-server-01.holochain.org"
}

server {
  enabled          = true
  bootstrap_expect = 1 # should increase this after testing

  # Reduce node garbage collection threshold to 1 hour,
  # as we expect signficant node churn from threefold deployments.
  node_gc_threshold = "1h"

  # Garbage collect allocations after 8 hours.
  # This is longer than the time required to run all scenarios (~6h) and begin
  # checking for successful/failed allocations.
  batch_eval_gc_threshold = "8h"
}

tls {
  http = true
  rpc  = true

  rpc_upgrade_mode = true # Allows clients to connect without TLS

  ca_file   = "/etc/nomad.d/nomad-agent-ca.pem"
  cert_file = "/etc/nomad.d/global-server-nomad.pem"
  key_file  = "/etc/nomad.d/global-server-nomad-key.pem"
}

