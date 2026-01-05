# Kubernetes Security: Defense in Depth (Workshop)

This repository accompanies the article **"How to Secure Kubernetes: A Practical Defense-in-Depth Guide"**. It demonstrates how to secure a microservices application layer-by-layer.

## Prerequisites
- **Docker Desktop** (running)
- **Kind** (Kubernetes in Docker)
- **kubectl**
- **Make**

## The Application
A simple E-Commerce store with 5 components:
1. `store-frontend` (Next.js)
2. `store-api` (Go)
3. `order-processor` (Python)
4. `postgres` (Database)
5. `redis` (Queue)

## How to Run the Workshop
We have automated the workshop steps using a `Makefile`.

### 1. Setup Infrastructure
Initialize the cluster, build images, and deploy the **INSECURE** application.
```bash
make cluster
make build
make load
make setup-base
```
*Wait for pods to be ready.*

---

### 2. Demo Layer 3: Pod Security
**The Vulnerability:** Prove that a hacker can run a root container.
```bash
make check-root
```
*(Output: "SUCCESS: The pod is Running... ROOT IS ALLOWED")*

**The Fix:** Enforce the Restricted Pod Security Standard.
```bash
make fix-root-policy
```

**The Verification:** Prove that root is now blocked.
```bash
make verify-root-blocked
```
*(Output: "violates PodSecurity")*

**The App Fix:** Update the store apps to run as non-root so they keep working.
```bash
make fix-deployments
```

---

### 3. Demo Layer 4: Network Policies
**The Vulnerability:** Prove the Frontend can talk directly to the Database (it shouldn't!).
```bash
make check-network
```
*(Output: "Connected... NO FIREWALL exists")*

**The Fix:** Apply a "Deny All" policy and whitelisting.
```bash
make fix-network
```

**The Verification:** Prove the connection is now blocked.
```bash
make verify-network-blocked
```
*(Output: "Timeout/Error. Access is BLOCKED.")*

---

### 4. Cleanup
To delete the cluster:
```bash
make clean
```
