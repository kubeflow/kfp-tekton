# Helper Functions for Python Kubernetes Client

Below are the list of helper functions for applying common Kubernetes resources such as reading environment variables from secret.
All the helper functions for Kubernetes client are under `kfp_tekton.k8s_client_helper`.

General Usage:
```
from kfp_tekton import k8s_client_helper

containerOp(
    ...
).apply(k8s_client_helper.env_from_secret(env_name, secret_name, secret_key))
```

## env_from_secret

`env_from_secret` let the containerOp to load a secret value into its environment variable.

Args:
- **env_name**: The environment variable name for the ContainerOp
- **secret_name**: The Kubernetes secret name that holds the secret values.
- **secret_key**: The key for loading the specific secret value from the above Kubernetes secret.

Usage
```
from kfp_tekton.k8s_client_helper import env_from_secret

containerOp(
    ...
).apply(k8s_client_helper.env_from_secret(env_name, secret_name, secret_key))
```
