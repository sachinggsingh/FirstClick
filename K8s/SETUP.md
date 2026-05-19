# Kubernetes Setup & Troubleshooting Guide

## Issues Fixed

### 1. ✓ Pod Labels Don't Match
**Problem**: Deployment metadata had `app: firstclick-deployment` but Service selector expected `app: firstclick`

**Fix**: Updated [deployment.yaml](./apps/firstclick/deployment.yaml) to use consistent label `app: firstclick`

```yaml
metadata:
  labels:
    app: firstclick  # Now matches service selector
```

---

### 2. ✓ Container Listening on Port 8080
**Status**: Container is properly configured to listen on port 8080

**Current Configuration**:
- Container port: 8080
- Service targetPort: 8080
- Service port: 80 (external)
- Ingress maps port 80 to service port 80

**Verify Application**: Ensure your Go application is listening on port 8080. In `cmd/firstclick/main.go`, check:
```go
// Should listen on :8080
http.ListenAndServe(":8080", handler)
```

---

### 3. ✓ Ingress Controller Not Running
**Solution**: Created NGINX Ingress Controller manifests in [K8s/ingress-controller/](./ingress-controller/)

**Deploy the ingress controller**:
```bash
kubectl apply -f K8s/ingress-controller/
```

**Files created**:
- `namespace.yaml` - ingress-nginx namespace
- `deployment.yaml` - NGINX controller deployment
- `service.yaml` - LoadBalancer service
- `serviceaccount.yaml` - ServiceAccount
- `clusterrole.yaml` - ClusterRole
- `clusterrolebinding.yaml` - ClusterRoleBinding
- `configmap.yaml` - NGINX configuration

**Verify deployment**:
```bash
kubectl get pods -n ingress-nginx
kubectl get svc -n ingress-nginx
```

---

### 4. ✓ /etc/hosts Not Configured
**Solution**: Configure local machine to resolve `firstclick.local`

**Option A: Using the provided script**:
```bash
chmod +x K8s/setup-hosts.sh
./K8s/setup-hosts.sh
```

**Option B: Manual configuration**:
Edit `/etc/hosts` and add:
```
127.0.0.1 firstclick.local
```

**Verify**:
```bash
ping firstclick.local
# Should resolve to 127.0.0.1
```

---

## Deployment Steps

1. **Deploy the Ingress Controller** (one-time):
   ```bash
   kubectl apply -f K8s/ingress-controller/
   ```

2. **Configure /etc/hosts**:
   ```bash
   ./K8s/setup-hosts.sh
   # or manually add: 127.0.0.1 firstclick.local
   ```

3. **Deploy FirstClick application**:
   ```bash
   kubectl apply -f K8s/apps/firstclick/
   kubectl apply -f K8s/base/
   ```

4. **Verify all components**:
   ```bash
   kubectl get pods -n firstclick
   kubectl get svc -n firstclick
   kubectl get ingress -n firstclick
   kubectl get pods -n ingress-nginx
   ```

5. **Access the application**:
   ```bash
   http://firstclick.local
   ```

---

## Troubleshooting

### Check if ingress controller is running:
```bash
kubectl get pods -n ingress-nginx
kubectl logs -n ingress-nginx deployment/nginx-ingress-controller
```

### Check service endpoints:
```bash
kubectl get endpoints firstclick-service -n firstclick
kubectl describe svc firstclick-service -n firstclick
```

### Check ingress status:
```bash
kubectl describe ingress firstclick-ingress -n firstclick
kubectl get ingress -n firstclick -o wide
```

### Test pod connectivity:
```bash
kubectl port-forward -n firstclick svc/firstclick-service 8080:80
# Then visit http://localhost:8080
```

### Verify /etc/hosts:
```bash
cat /etc/hosts | grep firstclick
ping firstclick.local
nslookup firstclick.local
```
