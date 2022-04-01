package {{.Name}}

import "github.com/suffiks/suffiks/extension"


// Config contains all the configuration for the extension.
type Config struct {
	// ConfigSpec is the required configuration by the Suffiks extension framework.
	extension.ConfigSpec `json:",inline"`
}
