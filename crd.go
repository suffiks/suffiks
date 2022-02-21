package suffiks

import "embed"

//go:embed config/crd/bases/suffiks.com_applications.yaml config/crd/bases/suffiks.com_works.yaml
var CRDFiles embed.FS
