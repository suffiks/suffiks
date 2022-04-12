---
category: Extensions
group: Create
title: Create extension
weight: 1
---

# Create extension

Although the platform has no requirements on the language or technology used to implement extensions, except that they must be GRPC services, the easiest way to implement an extension is to use the `extgen` tool.

## Install `extgen`

Install Go 1.18 or later by following [the official documentation](https://go.dev/doc/install).

Install `extgen` by running the following command:

```bash
go install github.com/suffiks/suffiks/cmd/extgen@latest
```

## Generate extension

Go to the directory where you want to create the extension.

```bash
# Create a minimal new extension:
extgen new github.com/myorg/myextension
```

There are a few options that can be passed to `extgen new`:

- `--validation` Add validation webhook support (default: `false`)
- `--defaulting` Add defaulting webhook support (default: `false`)
- `--kubernetes` Add kubernetes support (default: `false`)
- `--always` Always run the sync of this extension (default: `false`)
- `--target value` Types to target

After creating the extension, `cd` into it and run `go mod tidy` and `make generate`.

## Structure

The folder structure of an extension is as follows:

```
├── cmd
│   └── [extensionname]
│       └── main.go # The entrypoint for the extension
├── config
│   └── ... kustomize files
├── Dockerfile
├── [extensionname]
│   ├── config.go # The configuration for the extension
│   ├── [extensionname].go # The extension logic
│   ├── [extensionname]_test.go # The extension test
│   └── [extensionname]_type.go # The extension spec
├── go.mod
├── Makefile
└── README.md
```

## Markers

There's two markers supported by `extgen`:  
`// +suffiks:extension` and `// +kubebuilder:rbac`.

`// +suffiks:extension` is used to generate the Extension CRD and has the folloiwing options:

- `Targets`: One or more of the supported Kinds, separated by semicolons.
- `Validation`: Set to true to enable validation webhook support.
- `Defaulting`: Set to true to enable defaulting webhook support.
- `Always`: Set to true to always run the sync of the extension.

To use options in `// +suffiks:extension` you can use the following syntax:

```
// +suffiks:extension:Targets=Application;Work,Validation=true,Defaulting=true,Always=true
```

`// +kubebuilder:rbac` is documented in the [kubebuilder book](https://book.kubebuilder.io/reference/markers/rbac.html).
