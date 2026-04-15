# See https://developer.hashicorp.com/nomad/tutorials/access-control/access-control-policies for ACL Policy details

namespace "default" {
  capabilities = ["read-job", "submit-job", "alloc-lifecycle", "read-logs", "read-fs"]
}

node {
  policy = "read"
}

