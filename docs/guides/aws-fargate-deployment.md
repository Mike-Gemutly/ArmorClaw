# AWS Fargate Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** AWS Fargate (https://aws.amazon.com/fargate/)
> **Best For:** Enterprise production, highly available deployments, AWS ecosystem integration
> **Difficulty Level:** Advanced
> **Estimated Time:** 90-120 minutes

---

## Executive Summary

**AWS Fargate** is a serverless compute engine for containers that works with Amazon ECS and EKS. Fargate removes the need to provision and manage servers, letting you deploy and scale containerized applications.

### Why AWS Fargate for ArmorClaw?

✅ **Serverless** - No EC2 instances to manage
✅ **Auto-Scaling** - Scale based on CPU, memory, or custom metrics
✅ **High Availability** - Multi-AZ deployment
✅ **AWS Integration** - RDS, EFS, CloudWatch, Secrets Manager
✅ **Spot Pricing** - Up to 70% discount with Fargate Spot
✅ **Enterprise-Grade** - Compliance, security, reliability

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      AWS Cloud Infrastructure                │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  Application Load Balancer            │   │
│  │                  (HTTPS: Port 443)                    │   │
│  └────────────────────────┬─────────────────────────────┘   │
│                           │                                 │
│  ┌────────────────────────┴─────────────────────────────┐   │
│  │                    Amazon ECS Cluster                 │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │   │
│  │  │  Fargate     │  │  Fargate     │  │ Fargate   │ │   │
│  │  │  Task 1      │  │  Task 2      │  │ Task 3    │ │   │
│  │  │  (Bridge)    │  │  (Agent)     │  │ (Matrix)  │ │   │
│  │  │  AZ: us-ea-1a│  │  AZ: us-ea-1b│  │ AZ: us-ea-1c│ │   │
│  │  └──────────────┘  └──────────────┘  └───────────┘ │   │
│  │                                                     │   │
│  │  Auto-Scaling Group (0-N tasks)                    │   │
│  │  Cloud Watch Alarms and Metrics                     │   │
│  └──────────────────────────────────────────────────────┘   │
│                          │                                   │
│  ┌───────────────────────┼───────────────────────────────┐   │
│  │                       │                               │   │
│  ▼                       ▼                               ▼   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  Amazon RDS  │  │  Amazon EFS  │  │  Secrets     │   │
│  │  (PostgreSQL)│  │  (Shared FS) │  │  Manager     │   │
│  │  Multi-AZ    │  │              │  │              │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              CloudWatch (Logging & Monitoring)        │   │
│  │  (Logs, Metrics, Alarms, Dashboards, X-Ray)           │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Prerequisites

- AWS Account with appropriate permissions
- AWS CLI v2 installed
- Docker installed locally
- Basic AWS knowledge (VPC, subnets, security groups)

### 1. Install AWS CLI

**macOS:**
```bash
brew install awscli
```

**Linux:**
```bash
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install
```

**Windows:**
- Download installer from https://aws.amazon.com/cli/

### 2. Configure AWS Credentials

```bash
aws configure
# Enter:
# - AWS Access Key ID
# - AWS Secret Access Key
# - Default region: us-east-1
# - Default output format: json
```

### 3. Create ECS Cluster

```bash
aws ecs create-cluster --cluster-name armorclaw-cluster
```

### 4. Build and Push Image

```bash
# Build image
docker build -t armorclaw-bridge:latest .

# Tag for ECR
docker tag armorclaw-bridge:latest \
  <account-id>.dkr.ecr.us-east-1.amazonaws.com/armorclaw:latest

# Push to ECR
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/armorclaw:latest
```

### 5. Deploy Fargate Service

```bash
aws ecs run-task \
  --cluster armorclaw-cluster \
  --task-definition armorclaw-bridge \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxxx],securityGroups=[sg-xxxx],assignPublicIp=ENABLED}"
```

---

## Detailed Deployment

### 1. VPC and Networking Setup

**Create VPC:**
```bash
aws ec2 create-vpc \
  --cidr-block 10.0.0.0/16 \
  --tag-specifications 'ResourceType=vpc,Tags=[{Key=Name,Value=armorclaw-vpc}]'
```

**Create Subnets:**
```bash
# Public subnets (3 AZs for high availability)
aws ec2 create-subnet \
  --vpc-id vpc-xxxxx \
  --cidr-block 10.0.1.0/24 \
  --availability-zone us-east-1a \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=armorclaw-public-1a}]'

aws ec2 create-subnet \
  --vpc-id vpc-xxxxx \
  --cidr-block 10.0.2.0/24 \
  --availability-zone us-east-1b \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=armorclaw-public-1b}]'

aws ec2 create-subnet \
  --vpc-id vpc-xxxxx \
  --cidr-block 10.0.3.0/24 \
  --availability-zone us-east-1c \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=armorclaw-public-1c}]'
```

**Create Internet Gateway:**
```bash
aws ec2 create-internet-gateway \
  --tag-specifications 'ResourceType=internet-gateway,Tags=[{Key=Name,Value=armorclaw-igw}]'

# Attach to VPC
aws ec2 attach-internet-gateway \
  --vpc-id vpc-xxxxx \
  --internet-gateway-id igw-xxxxx
```

**Create Route Table:**
```bash
aws ec2 create-route-table \
  --vpc-id vpc-xxxxx \
  --tag-specifications 'ResourceType=route-table,Tags=[{Key=Name,Value=armorclaw-rt}]'

# Add route to Internet Gateway
aws ec2 create-route \
  --route-table-id rtb-xxxxx \
  --destination-cidr-block 0.0.0.0/0 \
  --gateway-id igw-xxxxx
```

### 2. Security Groups

**Create Security Group:**
```bash
aws ec2 create-security-group \
  --group-name armorclaw-sg \
  --description "ArmorClaw Fargate security group" \
  --vpc-id vpc-xxxxx

# Allow inbound HTTP (from ALB only)
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxx \
  --protocol tcp \
  --port 80 \
  --source-group sg-ALB-xxxxx

# Allow inbound HTTPS (from ALB only)
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxx \
  --protocol tcp \
  --port 443 \
  --source-group sg-ALB-xxxxx

# Allow all outbound
aws ec2 authorize-security-group-egress \
  --group-id sg-xxxxx \
  --protocol -1 \
  --port -1 \
  --cidr 0.0.0.0/0
```

### 3. ECR Repository

**Create Repository:**
```bash
aws ecr create-repository \
  --repository-name armorclaw \
  --image-tag-mutability IMMUTABLE
```

**Authenticate Docker:**
```bash
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  <account-id>.dkr.ecr.us-east-1.amazonaws.com
```

**Push Image:**
```bash
# Build
docker build -t armorclaw:latest .

# Tag
docker tag armorclaw:latest \
  <account-id>.dkr.ecr.us-east-1.amazonaws.com/armorclaw:latest

# Push
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/armorclaw:latest
```

### 4. ECS Task Definition

**Create Task Definition:**
```json
{
  "family": "armorclaw-bridge",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::ACCOUNT_ID:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::ACCOUNT_ID:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "armorclaw-bridge",
      "image": "<account-id>.dkr.ecr.us-east-1.amazonaws.com/armorclaw:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "ARMORCLAW_ENV",
          "value": "production"
        }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:ACCOUNT_ID:secret:armorclaw-db-url"
        },
        {
          "name": "API_KEY",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:ACCOUNT_ID:secret:armorclaw-api-key"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/armorclaw-bridge",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs",
          "awslogs-create-group": "true"
        }
      },
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "curl -f http://localhost:8080/health || exit 1"
        ],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

**Register Task Definition:**
```bash
aws ecs register-task-definition \
  --cli-input-json file://task-definition.json
```

### 5. ECS Service

**Create Service:**
```bash
aws ecs create-service \
  --cluster armorclaw-cluster \
  --service-name armorclaw-bridge \
  --task-definition armorclaw-bridge:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --deployment-configuration "maximumPercent=200,minimumHealthyPercent=100" \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxxx,subnet-yyyy,subnet-zzzz],securityGroups=[sg-xxxx],assignPublicIp=DISABLED}" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:us-east-1:ACCOUNT_ID:targetgroup/armorclaw-tg/xxxxx,containerName=armorclaw-bridge,containerPort=8080"
```

### 6. Application Load Balancer

**Create Load Balancer:**
```bash
aws elbv2 create-load-balancer \
  --name armorclaw-alb \
  --subnets subnet-xxxx subnet-yyyy subnet-zzzz \
  --security-groups sg-xxxxx \
  --scheme internet-facing \
  --type application \
  --ip-address-type ipv4
```

**Create Target Group:**
```bash
aws elbv2 create-target-group \
  --name armorclaw-tg \
  --protocol HTTP \
  --port 8080 \
  --target-type ip \
  --vpc-id vpc-xxxxx \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 3 \
  --unhealthy-threshold-count 2
```

**Create Listener:**
```bash
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:ACCOUNT_ID:loadbalancer/net/armorclaw-alb/xxxxx \
  --protocol HTTPS \
  --port 443 \
  --certificates CertificateArn=arn:aws:acm:us-east-1:ACCOUNT_ID:certificate/xxxxx \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:ACCOUNT_ID:targetgroup/armorclaw-tg/xxxxx
```

### 7. Auto Scaling

**Create Target Tracking Policy:**
```bash
aws application-autoscaling put-scaling-policy \
  --policy-name armorclaw-cpu-tracking \
  --policy-type TargetTrackingScaling \
  --resource-id service/armorclaw-cluster/armorclaw-bridge \
  --scalable-dimension ecs:service:DesiredCount \
  --service-namespaces ecs \
  --target-tracking-scaling-policy-configuration file://scaling-policy.json
```

**scaling-policy.json:**
```json
{
  "TargetValue": 70.0,
  "PredefinedMetricSpecification": {
    "PredefinedMetricType": "ECSServiceAverageCPUUtilization"
  },
  "ScaleOutCooldown": 300,
  "ScaleInCooldown": 300,
  "DisableScaleIn": false
}
```

---

## Pricing Details

### Fargate Pricing (2026)

**On-Demand Pricing (us-east-1):**
- **CPU:** $0.04048 per vCPU-hour
- **Memory:** $0.0044 per GB-hour

**Example Configurations:**

| Configuration | vCPU | Memory | Hourly Cost | Monthly Cost (24/7) |
|--------------|------|--------|-------------|---------------------|
| **Micro** | 0.25 vCPU | 0.5 GB | $0.012 | ~$9 |
| **Small** | 0.5 vCPU | 1 GB | $0.024 | ~$18 |
| **Medium** | 1 vCPU | 2 GB | $0.049 | ~$36 |
| **Large** | 2 vCPU | 4 GB | $0.098 | ~$72 |

**Fargate Spot Pricing:**
- **Discount:** Up to 70% off on-demand
- **Example:** Medium config ~$10-12/month (with 2 tasks)
- **Trade-off:** Can be interrupted with 2-minute notice

### Total Cost Estimation

**Small Production Deployment:**
```
- ECS Fargate (2 tasks, 0.5 vCPU, 1 GB): $36/month
- ALB: $20/month
- RDS (db.t3.micro): $15/month
- EFS (1 GB): $0.33/month
- CloudWatch: $5/month
- Data Transfer: $5/month
Total: ~$80/month
```

**With Fargate Spot:**
```
- ECS Fargate Spot (2 tasks): ~$10/month
Total: ~$55/month
```

---

## Limitations

### Platform Limitations

| Limitation | Default | Maximum |
|------------|---------|---------|
| **CPU per Task** | 0.25 vCPU | 4 vCPU (can increase to 8) |
| **Memory per Task** | 0.5 GB | 30 GB (can increase to 120 GB) |
| **Task Definition Size** | - | 64 KB |
| **Containers per Task** | - | 10 |
| **Network Interfaces** | - | 4 (awsvpc network mode) |

### ArmorClaw Considerations

✅ **Supported:**
- Long-running containers (no timeout)
- Full Docker control
- Multi-AZ high availability
- Auto-scaling
- AWS services integration

⚠️ **Considerations:**
- More complex than serverless alternatives
- Requires VPC, subnets, security groups
- Fargate Spot interruptions (if using Spot)

---

## Monitoring and Logging

### CloudWatch Logs

**View Logs:**
```bash
aws logs tail /ecs/armorclaw-bridge --follow
```

**Create Log Group:**
```bash
aws logs create-log-group \
  --log-group-name /ecs/armorclaw-bridge \
  --retention-in-days 7
```

### CloudWatch Metrics

**Enable Container Insights:**
```bash
aws ecs update-cluster-settings \
  --cluster armorclaw-cluster \
  --settings name=containerInsights,value=enabled
```

### CloudWatch Alarms

**Create CPU Alarm:**
```bash
aws cloudwatch put-metric-alarm \
  --alarm-name armorclaw-high-cpu \
  --alarm-description "Alert when CPU > 80%" \
  --metric-name CPUUtilization \
  --namespace AWS/ECS \
  --statistic Average \
  --period 300 \
  --evaluation-periods 2 \
  --threshold 80 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=ServiceName,Value=armorclaw-bridge
```

---

## Troubleshooting

### Common Issues

**Task Not Starting:**
```bash
# Check task events
aws ecs describe-tasks \
  --cluster armorclaw-cluster \
  --tasks xxxxxxxxxxxx

# Check logs
aws logs tail /ecs/armorclaw-bridge
```

**Connection Refused:**
```bash
# Check security group rules
aws ec2 describe-security-groups --group-ids sg-xxxxx

# Check target group health
aws elbv2 describe-target-health \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:ACCOUNT_ID:targetgroup/armorclaw-tg/xxxxx
```

---

## Quick Reference

### Essential Commands

```bash
# Create cluster
aws ecs create-cluster --cluster-name armorclaw-cluster

# Register task definition
aws ecs register-task-definition --cli-input-json file://task-definition.json

# Create service
aws ecs create-service --cluster armorclaw-cluster --service-name armorclaw-bridge

# Update service
aws ecs update-service --cluster armorclaw-cluster --service armorclaw-bridge

# Scale service
aws ecs update-service --cluster armorclaw-cluster --service armorclaw-bridge --desired-count 3

# View logs
aws logs tail /ecs/armorclaw-bridge --follow

# Delete service
aws ecs delete-service --cluster armorclaw-cluster --service armorclaw-bridge --force
```

---

## Conclusion

AWS Fargate provides enterprise-grade serverless container hosting for ArmorClaw with:

✅ **High Availability** - Multi-AZ deployment
✅ **Auto-Scaling** - Automatic scaling based on metrics
✅ **Fargate Spot** - Up to 70% cost savings
✅ **AWS Integration** - Full ecosystem integration
✅ **Monitoring** - CloudWatch comprehensive monitoring

**Best For:**
- Enterprise production deployments
- High availability requirements
- Existing AWS infrastructure
- Compliance and security requirements

**Next Steps:**
1. Set up VPC and networking
2. Create ECS cluster
3. Deploy task definitions
4. Configure ALB and target groups
5. Set up auto-scaling and monitoring

**Related Documentation:**
- [AWS Fargate Documentation](https://docs.aws.amazon.com/AmazonECS/latest/userguide/AWS_Fargate.html)
- [Amazon ECS Developer Guide](https://docs.aws.amazon.com/AmazonECS/latest/developerguide)
- [Hostinger VPS Deployment](docs/guides/hostinger-vps-deployment.md)

---

**Document Last Updated:** 2026-02-07
**AWS Fargate Version:** Based on 2026 pricing and features
**ArmorClaw Version:** 1.2.0
