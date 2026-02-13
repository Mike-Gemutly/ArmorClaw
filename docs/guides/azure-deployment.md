# Azure Container Instances Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Microsoft Azure (https://azure.microsoft.com)
> **Best For:** Azure ecosystem integration, burstable workloads
> **Difficulty Level:** Intermediate to Advanced
> **Estimated Time:** 45-60 minutes

---

## Executive Summary

**Azure Container Instances (ACI)** is a serverless container service that provides fast, isolated container execution. ACI is ideal for burstable workloads and scenarios where you don't need full orchestration.

### Why Azure for ArmorClaw?

✅ **Serverless Containers** - No infrastructure management
✅ **Per-Second Billing** - Pay only for what you use
✅ **Azure Integration** - Cosmos DB, Key Vault, Monitor
✅ **Global Network** - 60+ regions worldwide
✅ **Fast Startup** - Containers start in seconds
✅ **Flexible Sizing** - Up to 48 GB RAM

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Microsoft Azure Cloud                     │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Container Group (ACI)                    │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │   │
│  │  │   Secure    │  │   Matrix    │  │  Sidecar  │  │   │
│  │  │   Claw      │  │   Service   │  │  (Logs)    │  │   │
│  │  │   Bridge    │  │              │  │            │  │   │
│  │  └──────────────┘  └──────────────┘  └───────────┘  │   │
│  │                                                     │   │
│  │  - Shared virtual network                           │   │
│  │  - Sidecar containers for logs                       │   │
│  │  - Azure File Share for persistent storage            │   │
│  └──────────────────────────────────────────────────────┘   │
│                          │                                   │
│  ┌───────────────────────┼───────────────────────────────┐   │
│  │                       │                               │   │
│  ▼                       ▼                               ▼   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  Azure Key   │  │  Cosmos DB  │  │  Azure File  │   │
│  │  Vault       │  │  (NoSQL)     │  │  Share       │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │          Azure Monitor (Logs, Metrics, Alerts)       │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### 1. Install Azure CLI

**macOS:**
```bash
brew install azure-cli
```

**Linux:**
```bash
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
```

**Windows:**
- Download installer from https://aka.ms/installazurecliwindows

### 2. Authenticate

```bash
az login
```

### 3. Create Resource Group

```bash
az group create \
  --name armorclaw-rg \
  --location eastus
```

### 4. Create Container

```bash
az container create \
  --resource-group armorclaw-rg \
  --name armorclaw-bridge \
  --image armorclaw/bridge:latest \
  --cpu 1 \
  --memory 1 \
  --ports 8080 \
  --dns-name-label armorclaw-bridge-unique
```

---

## Detailed Deployment

### 1. Create Container Registry

```bash
az acr create \
  --resource-group armorclaw-rg \
  --name armorclawRegistry \
  --sku Basic \
  --location eastus
```

### 2. Build and Push Image

```bash
# Login to ACR
az acr login --name armorclawRegistry

# Build image
az acr build \
  --registry armorclawRegistry \
  --image armorclaw-bridge:latest \
  .

# Or build locally and push
docker build -t armorclaw-bridge:latest .
docker tag armorclaw-bridge:latest armorclawRegistry.azurecr.io/armorclaw-bridge:latest
docker push armorclawRegistry.azurecr.io/armorclaw-bridge:latest
```

### 3. Create Container Group

```bash
az container create \
  --resource-group armorclaw-rg \
  --name armorclaw-bridge \
  --image armorclawRegistry.azurecr.io/armorclaw-bridge:latest \
  --registry-login-server armorclawRegistry.azurecr.io \
  --cpu 1 \
  --memory 2 \
  --ports 8080 \
  --dns-name-label armorclaw-bridge-unique \
  --environment-variables \
    ARMORCLAW_ENV=production \
    LOG_LEVEL=info \
  --secure-environment-variables \
    DATABASE_URL=... \
    API_KEY=...
```

### 4. Configure Persistent Storage

**Create Azure File Share:**

```bash
# Create storage account
az storage account create \
  --resource-group armorclaw-rg \
  --name armorclawstorage \
  --location eastus \
  --sku Standard_LRS

# Create file share
az storage share create \
  --account-name armorclawstorage \
  --name armorclaw-data

# Get storage key
STORAGE_KEY=$(az storage account keys list \
  --resource-group armorclaw-rg \
  --account-name armorclawstorage \
  --query '[0].value' -o tsv)
```

**Mount in Container:**

```bash
az container create \
  --resource-group armorclaw-rg \
  --name armorclaw-bridge \
  ... \
  --azure-file-volume-account-name armorclawstorage \
  --azure-file-volume-share-name armorclaw-data \
  --azure-file-volume-mount-path /data
```

### 5. Configure Key Vault

**Create Key Vault:**

```bash
az keyvault create \
  --name armorclaw-kv \
  --resource-group armorclaw-rg \
  --location eastus
```

**Store Secrets:**

```bash
az keyvault secret set \
  --vault-name armorclaw-kv \
  --name DatabaseUrl \
  --value "postgresql://..."

az keyvault secret set \
  --vault-name armorclaw-kv \
  --name ApiKey \
  --value "sk-proj-..."
```

**Enable Managed Identity:**

```bash
az container create \
  ... \
  --assign-identity \
  --vault-secret-name DatabaseUrl \
  --vault-secret-name ApiKey
```

---

## Pricing Details

### ACI Pricing (2026)

**Per-Second Billing:**

| vCPU | Memory per GB | Linux Price per Second |
|------|---------------|------------------------|
| 0.1 | 0.5 GB | $0.000011 |
| 0.25 | 1 GB | $0.000024 |
| 0.5 | 2 GB | $0.000047 |
| 1 | 4 GB | $0.000093 |
| 2 | 8 GB | $0.000186 |
| 4 | 16 GB | $0.000372 |

**Example Calculations:**

**Small Container (0.5 vCPU, 1 GB):**
```
Per second: $0.000024
Per minute: $0.00144
Per hour: $0.0864
Per day (24 hours): $2.07
Per month (30 days): $62.20
```

**Large Container (2 vCPU, 8 GB):**
```
Per second: $0.000186
Per minute: $0.01116
Per hour: $0.6696
Per day (24 hours): $16.07
Per month (30 days): $482.10
```

**Additional Costs:**
- **Data Transfer:** $0.087 per GB (North America)
- **Azure Files:** $0.06 per GB/month
- **Key Vault:** $0.03 per 10,000 operations
- **ACR:** $0.167 per day/month

---

## Limitations

| Limitation | Details |
|------------|---------|
| **No Background Processes** | Not suitable for persistent Bridge |
| **Execution Time** | No hard limit, but per-second billing adds up |
| **Cold Starts** | ~10-30 seconds to start |
| **No Auto-Scaling** | Use Azure Container Apps |
| **CPU Max** | 48 vCPU |
| **Memory Max** | 192 GB |

### ArmorClaw Considerations

⚠️ **NOT Recommended for:**
- Persistent Bridge (too expensive with per-second billing)
- Long-running services (use Azure Container Apps or AKS instead)

✅ **Good For:**
- Burstable agent workloads
- CI/CD pipelines
- Batch processing
- Temporary testing

**Alternative:** Consider **Azure Container Apps** for production deployments with background processes.

---

## Quick Reference

```bash
# Create container
az container create --resource-group RG --name NAME --image IMAGE

# List containers
az container list --resource-group RG

# Get logs
az container logs --resource-group RG --name NAME

# Restart container
az container restart --resource-group RG --name NAME

# Delete container
az container delete --resource-group RG --name NAME
```

---

## Conclusion

Azure Container Instances is best suited for burstable workloads, not persistent services.

**Best For:**
- Temporary agent containers
- Batch processing jobs
- CI/CD pipelines
- Development/testing

**NOT Recommended For:**
- Persistent Bridge (use Azure Container Apps or AKS)

**Alternative:** See [Azure Container Apps](https://azure.microsoft.com/services/container-apps/) for production.

**Related Documentation:**
- [Azure Container Instances Docs](https://docs.microsoft.com/azure/container-instances/)
- [Azure Container Apps](https://docs.microsoft.com/azure/container-apps/)

---

**Document Last Updated:** 2026-02-07
**Azure Version:** Based on 2026 pricing
**ArmorClaw Version:** 1.2.0
