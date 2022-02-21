package controllers

type {{.GoName}}Spec struct {
	Super bool `json:"super"`
}

// My super awesome extension
// +suffiks:extension:Targets={{.Targets}}{{if .Validation}},Validation=true{{end}}{{if .Defaulting}},Defaulting=true{{end}}{{if .Always}},Always=true{{end}}
type {{.GoName}} struct {
	{{.GoName}} *{{.GoName}}Spec `json:"{{.Name}},omitempty"`
}
