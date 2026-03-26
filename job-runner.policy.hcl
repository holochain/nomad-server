# See https://developer.hashicorp.com/nomad/tutorials/access-control/access-control-policies for ACL Policy details

namespace "default" {
  capabilities = ["list-jobs", "read-job", "submit-job", "alloc-lifecycle"]
}

node {
  policy = "read"
}

