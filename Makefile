
# Variables
CLUSTER_NAME := k8s-security-demo
NS := secure-store

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# --- Infrastructure ---
.PHONY: cluster
cluster: ## Create a Kind cluster
	kind create cluster --name $(CLUSTER_NAME) || echo "Cluster might already exist"

.PHONY: build
build: ## Build docker images
	@echo "Building component images..."
	docker build -t store-api:v1 src/store-api
	docker build -t order-processor:v1 src/order-processor
	docker build -t store-frontend:v1 src/store-frontend

.PHONY: load
load: ## Load images into Kind
	kind load docker-image store-api:v1 --name $(CLUSTER_NAME)
	kind load docker-image order-processor:v1 --name $(CLUSTER_NAME)
	kind load docker-image store-frontend:v1 --name $(CLUSTER_NAME)

.PHONY: setup-base
setup-base: ## Setup Namespace and Deploy Insecure App
	kubectl create ns $(NS) || echo "Namespace existing"
	kubectl apply -f base-app/ -n $(NS)
	@echo "Waiting for base app..."
	kubectl wait --for=condition=ready pod -l app=store-api -n $(NS) --timeout=120s

# --- Layer 3: Pod Security (Root) ---
.PHONY: check-root
check-root: ## Layer 3 Check: Prove we can run as Root (VULNERABILITY)
	@echo "ATTEMPT: Running a privileged root pod..."
	kubectl run root-exploit --image=nginx:alpine --restart=Never -n $(NS) -- rm -rf /tmp/exploit
	@echo "Wait for it to start..."
	@sleep 5
	@kubectl get pod root-exploit -n $(NS)
	@echo "SUCCESS: The pod is Running. This means ROOT IS ALLOWED (Bad)."
	@kubectl delete pod root-exploit -n $(NS) --ignore-not-found

.PHONY: fix-root-policy
fix-root-policy: ## Layer 3 Fix: Enforce Restricted Policy
	@echo "APPLYING: Pod Security Policy..."
	kubectl apply -f pod-security/namespace.yaml

.PHONY: verify-root-blocked
verify-root-blocked: ## Layer 3 Verify: Prove Root is now Blocked
	@echo "ATTEMPT: Running a privileged root pod again..."
	-kubectl run root-blocked --image=nginx:alpine --restart=Never -n $(NS)
	@echo "SUCCESS: You should see 'violates PodSecurity' error above."

.PHONY: fix-deployments
fix-deployments: ## Layer 3 App Fix: Update Deployments to be Compliant
	@echo "UPDATING: Apps to run as non-root (SecurityContext)..."
	kubectl apply -f secure-app/ -n $(NS)
	kubectl rollout status deploy/store-api -n $(NS)

# --- Layer 4: Network Policies ---
.PHONY: check-network
check-network: ## Layer 4 Check: Prove Frontend can access DB (VULNERABILITY)
	@echo "ATTEMPT: Frontend connecting directly to Postgres..."
	kubectl exec -it deploy/store-frontend -n $(NS) -- curl -v telnet://postgres:5432
	@echo "SUCCESS: Connected. This means NO FIREWALL exists (Bad)."

.PHONY: fix-network
fix-network: ## Layer 4 Fix: Apply Deny-All and Allow-Api
	@echo "APPLYING: Network Policies..."
	kubectl apply -f network-policies/

.PHONY: verify-network-blocked
verify-network-blocked: ## Layer 4 Verify: Prove Frontend is Blocked
	@echo "ATTEMPT: Frontend connecting to Postgres again..."
	-kubectl exec -it deploy/store-frontend -n $(NS) -- curl --connect-timeout 2 telnet://postgres:5432
	@echo "SUCCESS: Timeout/Error. Access is BLOCKED."

# --- Clean Up ---
.PHONY: clean
clean: ## Delete the cluster
	kind delete cluster --name $(CLUSTER_NAME)
