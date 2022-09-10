//go:build tools
// +build tools

package tools

import (
	_ "k8s.io/code-generator/pkg/util"
	_ "k8s.io/gengo/args"
)
