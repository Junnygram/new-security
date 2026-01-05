# Kubernetes Security Workshop: Securing the E-Commerce Store

Welcome to the **Defense in Depth** workshop. In this session, you will take an insecure E-Commerce application and harden it layer by layer using Kubernetes security best practices.

## Architecture
The application consists of 5 components:
1.  **`store-frontend`**: Next.js UI (Port 3000)
2.  **`store-api`**: Go API (Port 8080)
3.  **`order-processor`**: Python worker
4.  **`postgres`**: Database (Port 5432)
5.  **`redis`**: Message Queue (Port 6379)

## Prerequisites
-   A Kubernetes cluster (Minikube, Kind, or Docker Desktop).
-   `kubectl` CLI installed.
-   Docker installed.
-   Git installed.

---

## Step 0: Build the Images
Security starts with the container image. We will build our apps using multi-stage Dockerfiles.

1.  **Build the Images:**
    ```bash
    # Build Store API
    cd src/store-api
    docker build -t store-api:v1 .

    # Build Order Processor
    cd ../order-processor
    docker build -t order-processor:v1 .

    # Build Store Frontend
    cd ../store-frontend
    docker build -t store-frontend:v1 .
    
    # Return to root
    cd ../..
    ```
    *(Note: If using Minikube/Kind, you may need to load these images into the cluster manually)*

---

## Step 1: Create Namespace & Deploy
We stop using the `default` namespace immediately.

1.  **Create the Dedicated Namespace:**
    ```bash
    kubectl create namespace secure-store
    ```

2.  **Deploy the Application:**
    We deploy all 5 components into this namespace.
    ```bash
    kubectl apply -f base-app/ -n secure-store
    ```

3.  **Verify Status:**
    ```bash
    kubectl get pods -n secure-store
    ```
    Wait until all pods are `Running`.

---

## Step 2: Identity & Access (RBAC)
By default, pods use the default ServiceAccount. We will give the **store-api** its own identity.

1.  **Apply the RBAC manifests:**
    ```bash
    kubectl apply -f rbac/
    ```
    *(These manifests use `namespace: secure-store` automatically)*

2.  **Update the Deployment:**
    Edit the `store-api` deployment to use the new `store-api-sa`.
    ```bash
    kubectl set serviceaccount deployment store-api store-api-sa -n secure-store
    ```

---

## Step 3: Pod Security
Now, we lock down the runtime. We will enforce the **Restricted** Pod Security Standard on our `secure-store` namespace.

1.  **Label the Namespace:**
    This updates our `secure-store` namespace with strict labels.
    ```bash
    kubectl apply -f pod-security/namespace.yaml
    ```

2.  **Verify Violation:**
    Try to deploy an insecure pod (like a root container) to confirm it is blocked.
    ```bash
    kubectl run root-test --image=nginx --restart=Never -n secure-store
    # You should see an error: "violates PodSecurity"
    ```

---

## Step 4: Network Policies
We implement a Zero Trust network to stop lateral movement.

1.  **Apply the Firewall (Deny All):**
    ```bash
    kubectl apply -f network-policies/deny-all.yaml
    ```
    *Test:* `kubectl exec -it deploy/store-frontend -n secure-store -- curl http://postgres:5432` should fail (timeout).

2.  **Punch Holes (Allow Specific Traffic):**
    ```bash
    kubectl apply -f network-policies/allow-store-api.yaml
    ```

---

## Step 5: Governance (Kyverno)
Let's automate our guardrails.

1.  **Install Kyverno:**
    ```bash
    kubectl create -f https://github.com/kyverno/kyverno/releases/latest/download/install.yaml
    ```

2.  **Apply Policies:**
    ```bash
    kubectl apply -f kyverno/require-resource-limits.yaml
    ```

3.  **Test Policy:**
    Try to apply a pod without CPU limits. It should be blocked.

---

## Step 6: Auditing
Enable the "Flight Recorder".

1.  **Review Policy:**
    Check `audit/audit-policy.yaml`.
    *(Enabling auditing varies by cluster provider, usually requiring API server flags)*.

---

## Conclusion
You have hardened the store by:
1.  Isolating it in `secure-store`.
2.  Stripping container privileges.
3.  Defining strict Identities (RBAC).
4.  Locking down Network traffic.
5.  Enforcing Policy as Code.
