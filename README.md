# Kubernetes PFX to PEM Converter Operator

A Kubernetes operator that automatically converts PFX (PKCS#12) certificates to PEM format for use with Kubernetes TLS secrets and Gateway API.

## Overview

When using Kubernetes Gateway API with TLS certificates, the certificates must be in PEM format. However, many organizations use PFX certificates from sources like Azure Key Vault. This operator automatically converts PFX certificates stored in Kubernetes Secrets to PEM format, making them compatible with Gateway API and other Kubernetes resources that require TLS certificates.

## Features

- **Automatic Conversion**: Converts PFX certificates to PEM format automatically when Secrets are created or updated
- **Annotation-Based**: Uses annotations to control conversion behavior
- **Password Support**: Supports password-protected PFX files with passwords from annotations or other Secrets
- **CA Certificate Handling**: Extracts and stores CA certificates from PFX bundles
- **Gateway API Compatible**: Outputs standard `kubernetes.io/tls` type Secrets

## Installation

### Deploy the Operator

1. Apply the manifests to your cluster:

```bash
kubectl apply -f deploy/
```

This will:
- Create the `pfx-tls-system` namespace
- Create necessary RBAC resources (ServiceAccount, ClusterRole, ClusterRoleBinding)
- Deploy the operator

### Build from Source

```bash
# Build locally
make build

# Build Docker image
make docker-build

# Push Docker image
make docker-push
```

## Usage

### Basic Example (No Password)

Create a Secret with a PFX certificate and enable conversion:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
type: Opaque
data:
  pfx: <BASE64_ENCODED_PFX_DATA>
```

The operator will:
1. Detect the annotation
2. Convert the PFX to PEM format
3. Update the Secret with:
   - `tls.crt`: Certificate in PEM format
   - `tls.key`: Private key in PEM format
   - `ca.crt`: CA certificates (if present in PFX)
4. Change the Secret type to `kubernetes.io/tls`
5. Add the `pfx-tls.kubernetes.io/converted: "true"` annotation

### Example with Password in Annotation

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
    pfx-tls.kubernetes.io/password: "mypassword"
type: Opaque
data:
  pfx: <BASE64_ENCODED_PFX_DATA>
```

### Example with Password in Another Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pfx-password
  namespace: default
type: Opaque
data:
  password: <BASE64_ENCODED_PASSWORD>
---
apiVersion: v1
kind: Secret
metadata:
  name: my-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
    pfx-tls.kubernetes.io/password-secret-name: "pfx-password"
    pfx-tls.kubernetes.io/password-secret-key: "password"
type: Opaque
data:
  pfx: <BASE64_ENCODED_PFX_DATA>
```

### Using with Gateway API

After conversion, use the Secret with Gateway API:

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: my-gateway
  namespace: default
spec:
  gatewayClassName: my-gateway-class
  listeners:
    - name: https
      protocol: HTTPS
      port: 443
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: my-pfx-cert
            namespace: default
```

## Annotations

| Annotation | Required | Description | Default |
|------------|----------|-------------|---------|
| `pfx-tls.kubernetes.io/convert` | Yes | Enable conversion (must be `"true"`) | - |
| `pfx-tls.kubernetes.io/pfx-key` | No | Key in Secret containing PFX data | `"pfx"` |
| `pfx-tls.kubernetes.io/password` | No | Password for PFX file | `""` (empty) |
| `pfx-tls.kubernetes.io/password-secret-name` | No | Name of Secret containing password | - |
| `pfx-tls.kubernetes.io/password-secret-key` | No | Key in password Secret | - |
| `pfx-tls.kubernetes.io/converted` | No | Marker annotation (added by operator) | - |

## Architecture

The operator consists of three main components:

1. **Controller** (`pkg/controller/controller.go`): Watches Secrets and orchestrates conversion
2. **Converter** (`pkg/converter/converter.go`): Handles PFX to PEM conversion logic
3. **Main** (`main.go`): Entry point and Kubernetes client setup

### How It Works

1. The operator watches all Secrets in the cluster
2. When a Secret with `pfx-tls.kubernetes.io/convert: "true"` is detected:
   - Extracts PFX data from the specified key
   - Retrieves password (if specified)
   - Converts PFX to PEM format
   - Updates Secret with PEM data
   - Marks Secret as converted

## Development

### Prerequisites

- Go 1.21 or later
- Docker (for building container images)
- kubectl configured with a Kubernetes cluster

### Build and Test

```bash
# Format code
make fmt

# Run linter
make vet

# Run tests
make test

# Build binary
make build

# Clean build artifacts
make clean
```

### Local Development

Run the operator locally against a Kubernetes cluster:

```bash
go run main.go -kubeconfig=$HOME/.kube/config
```

## Security Considerations

- **Password Storage**: Avoid storing passwords in annotations for production use. Use Secret references instead.
- **RBAC**: The operator requires permissions to read and update Secrets across all namespaces.
- **Secret Types**: After conversion, the original PFX data remains in the Secret alongside PEM data.

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.