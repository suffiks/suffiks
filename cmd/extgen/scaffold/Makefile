generate:
	extgen crd --source ./{{.Name}} -type {{ .GoName }}
	extgen rbac --source ./{{.Name}} -name {{ .Name }}-extension

docker-build:
	docker build -t github.com/suffiks/extension-{{ .Name }}:latest .

kind: docker
	kind load docker-image github.com/suffiks/extension-{{ .Name }}:latest
	kubectl apply -k config
