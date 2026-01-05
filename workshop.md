# Kubernetes Security Workshop: Securing the E-Commerce Store

Welcome to the **Defense in Depth** workshop. In this session, you will take an insecure E-Commerce application and harden it layer by layer using Kubernetes security best practices.

## Prerequisites
- A Kubernetes cluster (Minikube, Kind, or Docker Desktop).
- `kubectl` CLI installed.
- Docker installed (to build images).

## Step 0: Build the Images
Security starts with the container image. We will build our apps using multi-stage Dockerfiles to ensure they are minimal and don't run as root.

1.  **Build the Store API:**
    ```bash
    cd src/store-api
    docker build -t store-api:v1 .
    # (If using Minikube/Kind, verify how to load images locally)
    ```

2.  **Build the Order Processor:**
    ```bash
    cd src/order-processor
    docker build -t order-processor:v1 .
    ```

3.  **Build the Store Frontend:**
    ```bash
    cd src/store-frontend
    docker build -t store-frontend:v1 .
    ```

## Step 1: Deploy the Insecure Application
We will start by deploying our "E-Commerce Store," which consists of:
- `store-api`: The frontend API.
- `order-processor`: A background worker.
- `customer-db`: The database.

1.  **Deploy the manifests:**
    ```bash
    kubectl apply -f base-app/
    ```

2.  **Verify the pods are running:**
    ```bash
    kubectl get pods
    ```
    Wait until all pods are `Running`.

At this stage, the application is insecure:
- It uses the `default` service account.
- It runs as `root`.
- All pods can talk to each other freely.

---

## Step 2: Identity & Access (RBAC)
Let's fix the identity problem. We will give the **store-api** its own identity.

1.  **Apply the RBAC manifests:**
    ```bash
    kubectl apply -f rbac/
    ```

2.  **Update the deployment to use the new identity:**
    Edit the `store-api` deployment to use `serviceAccountName: store-api-sa`.
    ```bash
    kubectl set serviceaccount deployment store-api store-api-sa
    ```

---

## Step 3: Pod Security
Now, let's stop the pods from running as root. We will enforce the **Restricted** Pod Security Standard on our namespace.

1.  **Label the namespace:**
    ```bash
    kubectl apply -f pod-security/namespace.yaml
    ```

2.  **Try to restart the insecure pods:**
    ```bash
    kubectl delete pod -l app=order-processor
    ```
    Running `kubectl get pods` will show that the new pod is **blocked** (ReplicaSet cannot create it) because it violates the policy.

---

## Step 4: Network Policies
We need to stop lateral movement. We'll implement a Zero Trust network.

1.  **Apply the "Deny All" firewall:**
    ```bash
    kubectl apply -f network-policies/deny-all.yaml
    ```
    *Test it:* Try to curl the database from the api pod. It should fail (timeout).

2.  **Allow necessary traffic:**
    ```bash
    kubectl apply -f network-policies/allow-store-api.yaml
    # (Note: In a real lab you would also apply allow-customer-db.yaml)
    ```

---

## Step 5: Governance (Kyverno)
Let's automate our checks.

1.  **Install Kyverno (if not already installed):**
    *(Follow official docs for installation if needed)*

2.  **Apply the Resource Limit Policy:**
    ```bash
    kubectl apply -f kyverno/require-resource-limits.yaml
    ```

3.  **Test the policy:**
    Try to create a pod without resource limits. Kyverno should block it immediately.

---

## Step 6: Auditing
Finally, we enable the "Flight Recorder".

1.  **Review the Audit Policy:**
    Open `audit/audit-policy.yaml` to see how we configure logging for critical events.
    *(Note: Applying audit policies requires control plane access, often done via startup flags on the API server).*

---

## Conclusion
Congratulations! You have successfully transformed an insecure application into a hardened, defense-in-depth deployment.
