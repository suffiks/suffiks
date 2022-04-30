# {{.Name}}

{{.Name}} is an extension for [suffiks](https://github.com/suffiks/suffiks).

## Development

After changing the type defined in `./{{.Name}}/{{.Name}}_type.go`, run `make generate` to regenerate the Extension CRD and optional RBAC config.

## Documentation

Documentation is generated from markdown files in `./docs/`.
These are shared with the suffiks operator and used to generate documentation for the entire cluster it is deployed to.
