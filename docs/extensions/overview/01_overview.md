---
category: Extensions
group: Overview
title: Overview
weight: 1
---

# What is an extension?

A Suffiks extension is a GRPC service that can be used to extend the functionality of the platform.
It hooks into the platform and can be used to add new features to the platform.

Usually you will want to extens the specification of one or more of the available kinds provided by Suffiks.

## Features

The GRPC service must be built from the [`extension.proto`](https://github.com/suffiks/suffiks/blob/main/extension/proto/extension.proto) file.

### Sync

All extensions must implement the `Sync` method, which is invoked when the extension is defined in any of the kind specs.

This method should be used to create resources or modify the resources that will be managed by Suffiks.

### Delete

All extensions must implement the `Delete` method, which is invoked when the extension either no longer in use by any kind specs, or the kind is deleted.

### Default

The `Default` method is an optional method that is invoked when a kind is created, modified or deleted.
This will be called regardless of whether the extension is used by any kind specs.

`Default` should be used to modify the stored spec with default values for the fields managed by the extension.

### Validate

It can optionally implement `Validate` which is invoked whenever the extension is used by a kind spec.

## CRD

The CRD is used to add the extension to the platform.

The following example is:

- An extension named `extension-sample`.
- running a service named `suffiks-ingress` in the `system` namespace.
- extens `Application` with an `ingress` field.

```yaml
apiVersion: suffiks.com/v1
kind: Extension
metadata:
  name: extension-sample
spec:
  targets:
    - Application
  controller:
    service: suffiks-ingress
    namespace: system
  openAPIV3Schema:
    type: object
    properties:
      spec:
        properties:
          ingresses:
            description:
              List of URLs that will route HTTPS traffic to the application.
              All URLs must start with `https://`. Domain availability differs
              according to which environment your application is running in.
            items:
              pattern: ^https:\/\/.+$
              type: string
            type: array
        type: object
```

### Spec

`spec.always`: Always call the extension, even if the extension schema isn't used.

`spec.controller.service`: Name of the extension service.  
`spec.controller.namespace`: Namespace the extension service is running in.  
`spec.controller.port`: Port the extension service is running on.

`spec.openAPIV3Schema`: The extension schema.

`spec.targets`: A list of kinds that the extension is used for.

`spec.webhooks.defaulting`: Enable the defaulting webhook for this extension.  
`spec.webhooks.validation`: Enable the validation webhook for this extension.
