package {{.Name}}

// +kubebuilder:validation:Pattern=`^[\w\.\-]+$`
type EnvName string

type ExtraEnv struct {
	Name  EnvName `json:"name"`
	Value string  `json:"value"`
}

type Spec struct {
	// ExtraEnv adds a single, extra argument to the default container if set.
	ExtraEnv *ExtraEnv `json:"extraEnv,omitempty"`
}

// My super awesome extension
// +suffiks:extension:Targets={{.Targets}}{{if .Validation}},Validation=true{{end}}{{if .Defaulting}},Defaulting=true{{end}}{{if .Always}},Always=true{{end}}
type {{.GoName}} struct {
	{{.GoName}} *Spec `json:"{{.Name}},omitempty"`
}
