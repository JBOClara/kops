
<!--- This file is automatically generated by make gen-cli-docs; changes should be made in the go CLI command code (under cmd/kops) -->

## kops export kubeconfig

Export kubeconfig.

### Synopsis

Export a kubeconfig file for a cluster from the state store. By default the configuration will be saved into a users $HOME/.kube/config file.

```
kops export kubeconfig [CLUSTER | --all] [flags]
```

### Examples

```
  # export a kubeconfig file with the cluster admin user (make sure you keep this user safe!)
  kops export kubeconfig k8s-cluster.example.com --admin
  
  # export using a user already existing in the kubeconfig file
  kops export kubeconfig k8s-cluster.example.com --user my-oidc-user
  
  # export using the internal DNS name, bypassing the cloud load balancer
  kops export kubeconfig k8s-cluster.example.com --internal
```

### Options

```
      --admin duration[=18h0m0s]   Also export a cluster admin user credential with the specified lifetime and add it to the cluster context
      --all                        Export all clusters from the kOps state store
      --auth-plugin                Use the kOps authentication plugin
  -h, --help                       help for kubeconfig
      --internal                   Use the cluster's internal DNS name
      --kubeconfig string          Filename of the kubeconfig to create
      --user string                Existing user in kubeconfig file to use
```

### Options inherited from parent commands

```
      --config string   yaml config file (default is $HOME/.kops.yaml)
      --name string     Name of cluster. Overrides KOPS_CLUSTER_NAME environment variable
      --state string    Location of state storage (kops 'config' file). Overrides KOPS_STATE_STORE environment variable
  -v, --v Level         number for the log level verbosity
```

### SEE ALSO

* [kops export](kops_export.md)	 - Export configuration.
