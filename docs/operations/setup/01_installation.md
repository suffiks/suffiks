---
category: Operations
group: Setup
title: Installation
weight: 1
---

!!!warning
As the platform is still in development, the installation process is not yet fully tested, but using helm should be possible.
!!!

## Requirements

Suffiks requires

- Kubernetes 1.20 or higher. Some extensions might require newer versions.
- [cert-manager](https://cert-manager.io) if you want to use Webhooks.

### Using helm

All Suffiks charts are available as OCI images.
List of available images can be found [here](https://github.com/orgs/suffiks/packages?tab=packages&q=charts).

You can install the chart with the following command:

```bash
helm install suffiks oci://ghcr.io/suffiks/charts/suffiks --version 0.1.0
```

And upgrade using:

```bash
helm upgrade suffiks oci://ghcr.io/suffiks/charts/suffiks --version 0.1.0
```

### Configuration

Suffiks has the following configuration options:

```yaml
# Tracing configuration
tracing:
	# OTLP GRPC tracing endpoint. If empty, tracing is disabled.
	otlpEndpoint: # tempo-eu-west-0.grafana.net:443 for grafana
	# Attributes to add to the spans. Key value string pairs.
	attributes: {}
# Default container runtime configuration
health:
	healthProbeBindAddress: :8091
leaderElection:
	leaderElect: true
	resourceName: 0ff08fbf.suffiks.com
metrics:
	bindAddress: :8090
webhook:
	port: 9443
```

### Values

The default `values.yaml` file can be [seen in the github repository](https://github.com/suffiks/charts/blob/main/suffiks/values.yaml).
