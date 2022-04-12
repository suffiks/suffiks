package docparser

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/suffiks/suffiks"
)

func TestParse(t *testing.T) {
	x, err := Parse(suffiks.DocFiles)
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(x)
}
