# How to Secure Kubernetes: A Practical Defense-in-Depth Guide

Security in Kubernetes is often compared to security in the physical world. If you want to protect a valuable asset—say, a bank vault—you wouldn't rely on just one lock on the front door. You'd have a perimeter fence, security guards, ID scanners, cameras, and finally, the vault door itself.

This is **Defense in Depth**.

In the Cloud Native world, you can break this down into the "4Cs of Security":
1.  **Cloud**: The underlying infrastructure (AWS, GCP, Azure).
2.  **Cluster**: The Kubernetes control plane and node components.
3.  **Container**: The images running inside your pods.
4.  **Code**: The application logic itself.

This guide focuses on the **Cluster** and **Container** layers, using a practical, hands-on scenario that you can deploy and experiment with using the accompanying [new-security](./new-security) repository.

## Prerequisites

To follow this tutorial, you will need:
-   **Docker Desktop** (or a similar container runtime) installed and running.
-   **kubectl** installed for interacting with the cluster.
-   **Git** installed to clone the repository.
-   A local Kubernetes cluster (like **Kind**, **Minikube**, or Docker Desktop's built-in Kubernetes).
-   Basic familiarity with YAML manifests and terminal commands.

## Getting Started

First, clone the repository to your local machine:

```bash
git clone https://github.com/YOUR_USERNAME/new-security.git
cd new-security
```

All the files referenced in this guide are located in this directory.

---

## The Scenario: The "Secure E-Commerce Store"

To make these concepts concrete, you will secure a hypothetical "E-Commerce Store" consisting of three components:

1.  **`store-frontend`**: A Next.js web UI for customers.
2.  **`store-api`**: A Go API that handles requests and publishes events.
3.  **`order-processor`**: A Python worker that processes orders from a queue.
4.  **`postgres`**: A database storing sensitive customer and order data.
5.  **`redis`**: A message queue for decoupling services.

By default, an out-of-the-box Kubernetes cluster allows all these components to talk to each other freely, run as `root`, and access the Kubernetes API with broad permissions. **You function as the security team tasked with locking this down.**

---

## Layer 0: Code Security (Input Sanitization)
**The "What is Written"**

Before compiling your code or building your container, you must ensure the application logic itself is secure. A common vulnerability is **Injection Attacks** (SQL Injection, XSS), where malicious input causes the application to execute unintended commands.

**The Fix:** Sanitize all input at the API boundary, validating on both **Frontend** and **Backend**.

### Frontend Sanitization
In real-world applications, you would typically use libraries like **Zod**, **Yup**, or **React Hook Form** to handle complex validation schema easily. However, for this workshop, you implement a basic sanitization function in the `store-frontend` (React) to demonstrate the core concept of stripping dangerous characters before sending data to the API.

**File:** `new-security/src/store-frontend/app/page.js`
```javascript
const sanitizeInput = (input) => {
  return input.replace(/[<>]/g, '');
};
```

### Backend Sanitization
In the Go `store-api`, you implement a strict sanitization function to strip dangerous characters before processing any order. This is critical because attackers can bypass the frontend entirely.

**File:** `new-security/src/store-api/main.go`
```go
// SanitizeInput removes potentially dangerous characters
func SanitizeInput(input string) string {
    // Remove characters that could be used for XSS or Injection
    safe := strings.ReplaceAll(input, "<", "")
    safe = strings.ReplaceAll(safe, ">", "")
    safe = strings.ReplaceAll(safe, "'", "")
    return safe
}
```

This ensures that even if a hacker tries to send `<script>alert('hack')</script>` as a product ID, the application neutralizes it immediately.

---

## Layer 1: Container Security (The Runtime)
**The "What is running"**

Security starts before you even deploy to Kubernetes. If your container image is full of vulnerabilities or runs as root, your cluster is at risk.

**The Fix:** Use Multi-stage builds and Distroless images.

You use **Multi-stage builds** to compile the application in one stage and copy *only* the binary to a minimal runtime image in the second stage. This removes build tools, shells, and unnecessary packages that attackers could use.

**File:** `new-security/src/store-api/Dockerfile`
```dockerfile
# Build Stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o store-api main.go

# Run Stage
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /app/store-api .
# Run as non-root user
USER nonroot:nonroot
ENTRYPOINT ["/store-api"]
```

By using `gcr.io/distroless/static-debian12:nonroot`, you ensure the container has no shell (`/bin/sh` is missing), making it extremely hard for an attacker to execute commands even if they compromise the app.

### What about other options? (Alpine Linux)
If Distroless is too restrictive (e.g., you need a shell for debugging), **Alpine Linux** is an excellent alternative. We use `node:18-alpine` for our `store-frontend`.
Alpine is strictly stripped down, containing only a fraction of the packages found in full images. However, unlike Distroless, it includes a package manager (`apk`), allowing you to install specific tools (like `curl` or `busybox-extras`) if you need to debug a production issue. This offers a balance between security and operation.

### Can I use Distroless for Node.js?
Yes! Google maintains Distroless images for Node.js, Python, and Java. It provides the highest security but requires careful handling of dependencies.

```dockerfile
# Example Node.js Distroless
FROM gcr.io/distroless/nodejs20-debian12
COPY --from=builder /app /app
WORKDIR /app
CMD ["server.js"]
```

### What about other languages? (Manual Non-Root)
Not every app can use distroless easily. For interpreted languages like Python or Node.js, you should still ensure you don't run as root.

**File:** `new-security/src/order-processor/Dockerfile`
```dockerfile
# Create a non-root user for security
RUN useradd -m nonroot && chown -R nonroot /app
USER nonroot

ENTRYPOINT ["python", "app.py"]
```
This manually creates a user with fewer privileges, preventing the container process from modifying system files.

### Pro Tip: Pin Your Versions
**Never use the `:latest` tag in production.** It is unpredictable and can introduce breaking changes or new vulnerabilities at any time. always specify a version (like `node:18`) or, for maximum security, use the SHA256 digest (e.g., `image@sha256:abc...`).

---

## Layer 2: Identity & Access (RBAC)
**The "Who can do What"**

A common mistake is letting applications run with the `default` ServiceAccount. This gives your workload an identity that might be shared with other workloads and often has either too many or undefined permissions.

**The Fix:** Create dedicated ServiceAccounts and grant permissions using the Principle of Least Privilege.

### Step 1: Create a Dedicated Namespace
Deploying everything to the `default` namespace is insecure. It makes it impossible to isolate applications.
**Always create a dedicated namespace**.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: secure-store
```

### Example: Securing the `store-api`
Now, create a specific identity for the API service within that namespace.

**File:** `new-security/rbac/serviceaccount-store-api.yaml`
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: store-api-sa
  namespace: secure-store
```

Next, you define *exactly* what this API needs to do. It needs to read ConfigMaps and Secrets for its configuration, and discover Services. It does **not** need to delete pods or view nodes.

**File:** `new-security/rbac/role-store-api.yaml`
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: store-api-role
  namespace: secure-store
rules:
# Allow reading own ConfigMap and Secret
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  resourceNames: ["store-api-config", "store-api-secret"]
  verbs: ["get", "list"]
```

Finally, you bind the Identity (ServiceAccount) to the Permissions (Role) using a **RoleBinding**.

### Pro Tip: Disable Token Mounting
If your application (like our `order-processor`) does **not** need to talk to the Kubernetes API, you should prevent the token from being mounted inside the pod at all.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: order-processor-sa
automountServiceAccountToken: false
```
This prevents an attacker from stealing a credential they shouldn't have access to in the first place.

---

## Layer 3: Hardening the Workload (Pod Security)
**The "How it Runs"**

Container breakout attacks happen when a compromised application has direct access to the host's kernel or filesystem.

**The Fix:** Enforce Pod Security Standards (PSS).

You use Kubernetes' built-in Pod Security Admission controller. You can label a namespace to enforce the **Restricted** standard, which requires pods to:
- Drop all Linux capabilities.
- Run as a non-root user.
- Prevent privilege escalation.

**File:** `new-security/pod-security/namespace.yaml`
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: secure-store
  labels:
    # Enforce the restricted standard
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/enforce-version: latest
    # Warn on baseline violations
    pod-security.kubernetes.io/warn: baseline
```

If a developer tries to deploy a pod that runs as root in this namespace, the cluster will **reject the request** immediately.

---

## Layer 4: Network Segmentation (Network Policies)
**The "Who can talk to Whom"**

By default, Kubernetes is a flat network. A compromised frontend pod can scan and attack your database directly, even if it has no business talking to it.

**The Fix:** A Zero Trust Network using Network Policies.

You start with a "**Deny All**" policy. This acts as a firewall that blocks *all* traffic to and from every pod in the namespace.

**File:** `new-security/network-policies/deny-all.yaml`
```yaml
kind: NetworkPolicy
metadata:
  name: default-deny-all
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
```

Then, you selectively "punch holes" in the firewall. For example, you only allow traffic to the `store-api` if it comes from the Ingress Controller (or specific sources).

**File:** `new-security/network-policies/allow-store-api.yaml`
```yaml
kind: NetworkPolicy
spec:
  podSelector:
    matchLabels:
      app: store-api
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
```

---

## Layer 5: Governance (Policy as Code with Kyverno)
**The "Automated Guardrails"**

How do you ensure *every* team sets CPU limits? Or that no one uses the `latest` image tag? You can't review every YAML file manually.

**The Fix:** Policy Agents like Kyverno.

Kyverno lets you write policies as Kubernetes resources. Here is a policy that mandates resource limits to prevent "noisy neighbor" problems or DoS attacks.

**File:** `new-security/kyverno/require-resource-limits.yaml`
```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-resource-limits
spec:
  validationFailureAction: enforce
  rules:
  - name: check-resource-limits
    validate:
      message: "CPU and memory limits are required for all containers"
      pattern:
        spec:
          containers:
          - resources:
              limits:
                memory: "?*"
                cpu: "?*"
```

If you try to `kubectl apply` a pod without limits, Kyverno intervenes and blocks the request with a helpful error message.

---

## Layer 6: Visibility (Auditing)
**The "Flight Recorder"**

If an attack happens, you need to answer: *Who did it? When? and How?*

**The Fix:** Kubernetes Audit Logs.

You configure the API server to log detailed events. The policy below logs metadata for most requests but captures the full Request and Response body for critical actions like modifying Pods or Secrets.

**File:** `new-security/audit/audit-policy.yaml`
```yaml
rules:
  # Log pod changes with full details
  - level: RequestResponse
    resources:
    - group: ""
      resources: ["pods"]
    verbs: ["create", "update", "patch", "delete"]
```

---

## Conclusion

Securing Kubernetes is not a one-time task; it's a mindset. By implementing these layers:
1.  **Code Security**: Sanitize inputs to prevent Injection.
2.  **Container Security**: Build minimal, non-root images.
3.  **RBAC**: Lock down identities.
4.  **Pod Security**: Lock down the runtime.
5.  **Network Policies**: Lock down the network.
6.  **Kyverno**: Automate the rules.
7.  **Auditing**: Watch everything.

You create a robust defense-in-depth strategy.

### Next Steps
Explore the full [new-security](./new-security) folder in this repository to see the complete manifests and try applying them to your own testing cluster.
