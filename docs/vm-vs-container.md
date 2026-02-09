# Native VM vs Container Deployment

Summary of deployment strategy analysis for goloo.

## Recommendation: Native Installation + CNAME Swap

### Why Native Over Containers

| Factor | Native Wins Because |
|--------|---------------------|
| Simplicity | No container abstraction layer |
| Fit with goloo | Cloud-init + apt is natural pairing |
| Debugging | Direct SSH, familiar tools |
| Ubuntu-only target | Container portability not needed |

### The Key Insight: CNAME Swap Changes Everything

The CNAME swap strategy eliminates containers' main advantage (easy rollback):

```
1. Spin up new VM (green) with updated software
2. Test green VM
3. Swap CNAME: app.example.com → green IP
4. Keep blue VM 1 hour for rollback
5. Terminate blue
```

This gives you:
- Zero-downtime updates
- Instant rollback (point DNS back)
- Clean state every deploy (no drift)
- No complex update orchestration

### For Updates

- **Security patches**: Ubuntu's `unattended-upgrades` (automatic, already enabled)
- **App updates**: Either in-place `apt upgrade` or CNAME swap for zero-downtime
- **Major changes**: Always CNAME swap

### When Containers Make Sense

Choose Podman if:
- Services have conflicting dependencies
- Team already knows containers
- Software only available as images
- Planning Kubernetes migration

### Stateful Data

Externalize state to avoid migration complexity:
- Database → RDS (both VMs share it)
- Files → S3/EFS
- Cache → Rebuild (transient)

For dev VMs with local DB: backup/restore via S3 during swap.

## Full Analysis

See [DEPLOYMENT-ANALYSIS.md](./DEPLOYMENT-ANALYSIS.md) for the complete comparison including:
- Detailed pros/cons for each approach
- unattended-upgrades configuration
- Podman auto-update and Quadlets
- Blue-green DNS deployment diagrams
- Database migration strategies
- Security considerations
