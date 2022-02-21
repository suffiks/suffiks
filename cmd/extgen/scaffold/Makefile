generate:
	../../suffiks/cmd/extgen/extgen crd -type {{ .GoName }}
	../../suffiks/cmd/extgen/extgen rbac -name {{ .Name }}-extension

docker:
	go mod vendor
	docker build -t github.com/suffiks/extensions/{{ .Name }}:latest .

kind: docker
	kind load docker-image github.com/suffiks/extensions/{{ .Name }}:latest
	kubectl apply -k config
