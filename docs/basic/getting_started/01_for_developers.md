---
category: Basic
group: Getting started
title: For developers
weight: 1
---

# Suffiks for developers

Suffiks is a platform that lets you create and manage your applications and jobs in a simple and easy way.

!!! info
The features available in the platform vary among clusters, so it's strongly recommended to read the documentation provided by each cluster.
!!!

This guide will show you how to use the platform without any added extensions.
To learn more about how to use the available extensions in your cluster, see the left sidebar.

## Your first Suffiks Application

A bare bone suffiks application lets you run a long lived service and expose it to the cluster.
It's designed to run a single container with a single port exposed.
If the port is set, the application will be accessible from within the cluster on `[name].[namespace]:80`.

```yaml
apiVersion: suffiks.com/v1
kind: Application
metadata:
	# Name of the application
  name: application-sample
	# Namespace to deploy it to
  namespace: default
spec:
	# The image to run
  image: nginxdemos/nginx-hello:0.3-plain-text
	# The port to expose the service on
  port: 8080
	# The command to run
	command: ["nginx", "-g", "daemon off"]
	# Environment variables
	env:
		# Static environment variable
		- name: MY_ENV_VAR
		  value: "my-value"
	# Mount all keys from a configmap as environment variables
	# Requires that the config map is created in the same namespace
	# envFrom:
	# 	configmap: "my-configmap"
```
