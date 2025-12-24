# Quick Start Guide

This guide will help you quickly deploy and test the PFX to PEM converter operator.

## Prerequisites

- A Kubernetes cluster (version 1.24+)
- `kubectl` configured to access your cluster
- A PFX certificate file for testing

## Step 1: Deploy the Operator

```bash
# Clone the repository
git clone https://github.com/hellices/kubernetes-pfx-tls.git
cd kubernetes-pfx-tls

# Deploy the operator
kubectl apply -f deploy/
```

## Step 2: Verify the Operator is Running

```bash
# Check the operator pod status
kubectl get pods -n pfx-tls-system

# Check the logs
kubectl logs -n pfx-tls-system -l app=pfx-tls-operator
```

Expected output:
```
I1224 14:00:00.000000       1 controller.go:45] Setting up event handlers
I1224 14:00:00.000000       1 main.go:60] Starting Secret controller
I1224 14:00:00.000000       1 main.go:62] Waiting for informer caches to sync
I1224 14:00:00.000000       1 main.go:67] Starting workers
```

## Step 3: Create a Test Secret with PFX Certificate

First, encode your PFX certificate to base64:

```bash
# Encode your PFX file
base64 -w 0 your-certificate.pfx > pfx-base64.txt
```

Create a Secret manifest:

```yaml
# test-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
    # If your PFX has a password, uncomment and set it:
    # pfx-tls.kubernetes.io/password: "your-password"
type: Opaque
data:
  pfx: <PASTE_BASE64_DATA_HERE>
```

Apply the secret:

```bash
kubectl apply -f test-secret.yaml
```

## Step 4: Verify the Conversion

Check if the secret was converted:

```bash
# View the secret
kubectl get secret test-pfx-cert -o yaml
```

You should see:
- The secret type changed to `kubernetes.io/tls`
- New keys: `tls.crt`, `tls.key`, and `ca.crt` (if CA certs were in the PFX)
- Annotation `pfx-tls.kubernetes.io/converted: "true"`

Decode and view the certificate:

```bash
# View the certificate
kubectl get secret test-pfx-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout

# View the private key
kubectl get secret test-pfx-cert -o jsonpath='{.data.tls\.key}' | base64 -d
```

## Step 5: Use with Gateway API

Create a Gateway that uses the converted certificate:

```yaml
# gateway.yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: test-gateway
  namespace: default
spec:
  gatewayClassName: <your-gateway-class>
  listeners:
    - name: https
      protocol: HTTPS
      port: 443
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: test-pfx-cert
            namespace: default
```

Apply the gateway:

```bash
kubectl apply -f gateway.yaml
```

## Troubleshooting

### Operator Pod Not Running

```bash
# Check pod events
kubectl describe pod -n pfx-tls-system -l app=pfx-tls-operator

# Check RBAC permissions
kubectl get clusterrole pfx-tls-operator -o yaml
kubectl get clusterrolebinding pfx-tls-operator -o yaml
```

### Secret Not Converting

```bash
# Check operator logs for errors
kubectl logs -n pfx-tls-system -l app=pfx-tls-operator

# Verify the annotation is correct
kubectl get secret test-pfx-cert -o jsonpath='{.metadata.annotations}'

# Check if the PFX key exists in the secret
kubectl get secret test-pfx-cert -o jsonpath='{.data}' | jq 'keys'
```

### Password Issues

If your PFX is password-protected and you're getting conversion errors:

1. Store the password in a separate secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pfx-password
  namespace: default
type: Opaque
stringData:
  password: "your-password"
```

2. Reference it in your PFX secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
    pfx-tls.kubernetes.io/password-secret-name: "pfx-password"
    pfx-tls.kubernetes.io/password-secret-key: "password"
type: Opaque
data:
  pfx: <BASE64_PFX_DATA>
```

## Clean Up

To remove the operator and test resources:

```bash
# Delete test resources
kubectl delete secret test-pfx-cert
kubectl delete gateway test-gateway

# Remove the operator
kubectl delete -f deploy/
```

## Next Steps

- Check the [README.md](README.md) for complete documentation
- See [examples/](examples/) for more usage examples
- Review [README.ko.md](README.ko.md) for Korean documentation
