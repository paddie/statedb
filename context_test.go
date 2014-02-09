package statedb

import (
	"fmt"
	"os"
	"path"
	"testing"
)

var ctx_dir string

func init() {

	ctx_dir = "ctx_test"

}

// Make sure that the checkpoint directory is created with the context
func TestContextInit(t *testing.T) {

	t.Skip("skipping for now..")

	exp := 4

	cptid := path.Join(ctx_dir, fmt.Sprintf("%010d", exp))

	// make sure there is no existing ctx_test dir
	_ = os.RemoveAll(cptid)

	// Now create it again
	err := CreateDir(cptid)
	if err != nil {
		t.Error(err)
	}

	ctx, restore, err := NewContext("", ctx_dir, "")
	if err != nil {
		t.Error(err)
	}

	if restore {
		t.Error("Restore should not have been signaled at this point")
	}

	if ctx.cpt_id != 0 {
		t.Errorf("Incorrect cpt_id: %d", ctx.cpt_id)
	}

}
