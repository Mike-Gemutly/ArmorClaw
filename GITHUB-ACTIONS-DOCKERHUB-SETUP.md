# GitHub Actions Docker Hub Setup Guide

> **Purpose:** Configure GitHub Actions to automatically build and push Docker images to Docker Hub
> **Time:** 10-15 minutes
> **Difficulty:** Easy

---

## Prerequisites

1. **GitHub Account** with repository access
2. **Docker Hub Account** with a repository created
3. **Docker Desktop** installed (for local testing)

---

## Step 1: Create Docker Hub Repository

1. **Login to Docker Hub**
   - Visit: https://hub.docker.com
   - Login or create account

2. **Create repository**
   - Click **Create Repository**
   - Repository name: `agent`
   - Visibility: Public (recommended)
   - Click **Create**

3. **Get your repository URL**
   - Format: `docker.io/YOUR_USERNAME/agent`
   - Example: `docker.io/mikegemutly/agent`

---

## Step 2: Generate Docker Hub Access Token

**Important:** Use an access token instead of your password for security.

1. **Go to Docker Hub → Account Settings**
   - Click on your profile → **Account Settings**
   - Go to **Security** → **Access Tokens**

2. **Create new access token**
   - Click **New Access Token**
   - Description: `GitHub Actions - ArmorClaw`
   - Access permissions: Read, Write, Delete
   - Click **Generate**

3. **Copy the token**
   - ⚠️ **Copy and save immediately** - you won't see it again!

---

## Step 3: Configure GitHub Secrets

1. **Navigate to your GitHub repository**
   - Go to **Settings** → **Secrets and variables** → **Actions**

2. **Add repository secrets**
   Click **New repository secret** for each:

   **Secret 1: DOCKER_USERNAME**
   - Name: `DOCKER_USERNAME`
   - Value: Your Docker Hub username (e.g., `mikegemutly`)
   - Click **Add secret**

   **Secret 2: DOCKER_PASSWORD**
   - Name: `DOCKER_PASSWORD`
   - Value: Your Docker Hub access token (from Step 2)
   - Click **Add secret**

3. **Verify secrets**
   - You should see both secrets listed:
     - `DOCKER_USERNAME` ✓
     - `DOCKER_PASSWORD` ✓

---

## Step 4: Test Locally with Docker Desktop

**Before pushing, test the build locally:**

```bash
# Clone repository (if not already)
cd /path/to/ArmorClaw

# Build Docker image locally
docker build -t mikegemutly/agent:latest .

# Test run
docker run --rm mikegemutly/agent:latest --help

# Verify image exists
docker images | grep mikegemutly/agent
```

---

## Step 5: Trigger GitHub Action

**Option A: Push to main branch (automatic)**

```bash
# Make a commit and push
git add .
git commit -m "Test Docker Hub push"
git push origin main
```

The workflow will automatically start:
1. Build Docker image for linux/amd64 and linux/arm64
2. Push to Docker Hub
3. Scan for vulnerabilities
4. Test the image
5. Generate deployment summary

**Option B: Manually trigger**

1. Go to **Actions** tab in GitHub
2. Select **Docker Hub Build & Push** workflow
3. Click **Run workflow** → **Run workflow**
4. Choose branch: `main`
5. Click **Run workflow**

---

## Step 6: Monitor Build Progress

1. **Go to Actions tab**
   - You'll see the workflow running
   - Wait for completion (~5-10 minutes)

2. **View build logs**
   - Click on the running workflow
   - Expand steps to see progress:
     - ✅ Checkout repository
     - ✅ Set up QEMU
     - ✅ Set up Docker Buildx
     - ✅ Log in to Docker Hub
     - ✅ Extract Docker metadata
     - ✅ Build and push Docker image
     - ✅ Scan image for vulnerabilities
     - ✅ Test Docker Image

3. **Check summary**
   - Scroll to bottom of workflow run
   - You'll see a deployment summary with:
     - Image name and tags
     - Platform information
     - Pull command
     - Test results

---

## Step 7: Verify in Docker Hub

1. **Login to Docker Hub**
   - Go to your repository: `YOUR_USERNAME/agent`

2. **Check for new tags**
   - You should see:
     - `latest` tag
     - `main` tag (if push was to main)
     - Version tags (if release created)

3. **Verify image**
   - Click on **Tags** tab
   - Image details should show:
     - Size
     - Last pushed
     - Architecture: linux/amd64, linux/arm64

---

## Step 8: Test Pull from Docker Hub

```bash
# Pull the image
docker pull mikegemutly/agent:latest

# Verify image
docker images | grep mikegemutly/agent

# Test run
docker run --rm mikegemutly/agent:latest --help
```

---

## Workflow Details

### Triggers

The workflow runs automatically when:

- ✅ **Push to main branch** - with changes to Dockerfile, container/, or workflow file
- ✅ **Pull request to main** - for testing (builds but doesn't push)
- ✅ **Release created** - when you create a GitHub release
- ✅ **Manual trigger** - from Actions tab

### Build Matrix

The workflow builds for multiple platforms:
- `linux/amd64` - Standard Linux servers
- `linux/arm64` - ARM64 servers (Raspberry Pi, AWS Graviton, Apple Silicon)

### Security Features

- ✅ **Vulnerability scanning** with Trivy
- ✅ **SARIF reports** uploaded to GitHub
- ✅ **Secrets management** using GitHub Secrets
- ✅ **Multi-platform caching** for faster builds

### Workflow Jobs

1. **Build & Push Docker Image**
   - Builds image for all platforms
   - Pushes to Docker Hub
   - Scans for vulnerabilities
   - Generates deployment summary

2. **Test Docker Image**
   - Pulls image from Docker Hub
   - Verifies image runs
   - Reports test results

3. **Notify on Failure**
   - Sends error notification if build fails

---

## Environment Variables

The workflow uses these environment variables:

| Variable | Value | Description |
|-----------|---------|-------------|
| `DOCKER_IMAGE` | `{repo_owner}/agent` | Docker Hub image name |
| `PLATFORMS` | `linux/amd64,linux/arm64` | Build platforms |
| `BUILD_DATE` | Commit timestamp | Image metadata |
| `VCS_REF` | Git SHA | Image metadata |

---

## Troubleshooting

### Issue: "Authentication failed"

**Cause:** Invalid Docker Hub credentials

**Solution:**
```bash
# Update GitHub secrets
1. Go to Settings → Secrets
2. Delete DOCKER_USERNAME and DOCKER_PASSWORD
3. Regenerate access token in Docker Hub
4. Add secrets again
```

### Issue: "Repository not found"

**Cause:** Repository doesn't exist in Docker Hub

**Solution:**
```bash
# Create repository first
1. Login to Docker Hub
2. Create repository named "agent"
3. Re-run workflow
```

### Issue: "Build failed"

**Cause:** Dockerfile errors or missing files

**Solution:**
```bash
# Test build locally first
docker build -t test:latest .

# Check workflow logs for specific error
# Look at "Build and push Docker image" step
```

### Issue: "Image push failed"

**Cause:** Insufficient permissions or network issues

**Solution:**
```bash
# Verify access token permissions
1. Go to Docker Hub → Access Tokens
2. Ensure token has Read, Write, Delete permissions

# Test push manually
docker login
docker push YOUR_USERNAME/agent:latest
```

### Issue: "Vulnerability scan failed"

**Cause:** Security vulnerabilities found in image

**Solution:**
```bash
# View scan results
1. Go to workflow run
2. Scroll to "Scan image for vulnerabilities"
3. Click on "trivy-results.sarif"
4. Review and fix vulnerabilities
```

---

## Local Testing with Docker Desktop

### Build Image Locally

```bash
# Navigate to repository
cd /path/to/ArmorClaw

# Build image
docker build -t test-agent:latest .

# Test run
docker run --rm test-agent:latest

# Tag for Docker Hub
docker tag test-agent:latest YOUR_USERNAME/agent:latest

# Push to Docker Hub
docker login
docker push YOUR_USERNAME/agent:latest
```

### Multi-Platform Build Locally

```bash
# Build for both platforms (requires Buildx)
docker buildx create --name mybuilder --use

# Build amd64
docker buildx build --platform linux/amd64 -t YOUR_USERNAME/agent:amd64 .

# Build arm64
docker buildx build --platform linux/arm64 -t YOUR_USERNAME/agent:arm64 .
```

---

## Best Practices

1. **Use access tokens**, not passwords
   - Tokens can be rotated without changing password
   - Tokens have limited scope

2. **Test locally first**
   - Catch Dockerfile errors before pushing
   - Save CI/CD time and costs

3. **Review scan results**
   - Fix vulnerabilities before pushing to production
   - Keep base images up to date

4. **Use semantic versioning**
   - Create releases with tags like `v1.0.0`
   - Workflow automatically tags images

5. **Monitor build times**
   - Use caching to speed up builds
   - Workflow includes GitHub Actions cache

---

## Next Steps

1. ✅ Configure GitHub Secrets
2. ✅ Trigger workflow (push or manual)
3. ✅ Verify image in Docker Hub
4. ✅ Test pull and run locally
5. ✅ Deploy to Hostinger VPS
6. ✅ Set up automated updates

---

## Resources

- [Docker Hub Documentation](https://docs.docker.com/docker-hub/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [Trivy Security Scanner](https://aquasecurity.github.io/trivy/)
- [ArmorClaw Repository](https://github.com/Mike-Gemutly/ArmorClaw)
