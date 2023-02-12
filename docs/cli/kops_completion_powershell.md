
<!--- This file is automatically generated by make gen-cli-docs; changes should be made in the go CLI command code (under cmd/kops) -->

## kops completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	kops completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
kops completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   yaml config file (default is $HOME/.kops.yaml)
      --name string     Name of cluster. Overrides KOPS_CLUSTER_NAME environment variable
      --state string    Location of state storage (kops 'config' file). Overrides KOPS_STATE_STORE environment variable
  -v, --v Level         number for the log level verbosity
```

### SEE ALSO

* [kops completion](kops_completion.md)	 - Generate the autocompletion script for the specified shell
