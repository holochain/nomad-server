data_dir = "/opt/nomad/data"

acl {
  enabled = true
}

advertise {
  http = "{{ GetPublicIP }}"
  rpc  = "{{ GetPublicIP }}"
}

server {
  enabled          = true
  bootstrap_expect = 1 # should increase this after testing
  job_gc_threshold = "24h"
}

tls {
  http = true
  rpc  = true

  rpc_upgrade_mode = true # Allows clients to connect without TLS

  ca_file   = "/etc/nomad.d/nomad-agent-ca.pem"
  cert_file = "/etc/nomad.d/global-server-nomad.pem"
  key_file  = "/etc/nomad.d/global-server-nomad-key.pem"
}

