package statedb

import (
	"fmt"
	"os"
	"path"
	"testing"
)

var ctxDir string

func init() {

	ctxDir = "ctx_test"

}

// Make sure that the checkpoint directory is created with the context
func TestContextInit(t *testing.T) {

	// t.Skip("skipping for now..")

	exp := 4

	cptid := path.Join(ctxDir, fmt.Sprintf("%010d", exp))

	// make sure there is no existing ctx_test dir
	_ = os.RemoveAll(cptid)

	// Now create it again
	err := CreateDir(cptid)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := NewContext("", ctxDir, "")
	if err != nil {
		t.Fatal(err)

	}

	if !ctx.previous {
		t.Fatal("Restore should not have been signaled at this point")
	}

	if ctx.rcid != 4 {
		t.Errorf("Incorrect cpt_id: %d", ctx.rcid)
	}

}
