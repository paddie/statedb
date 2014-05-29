package statedb

import (
	// "encoding/gob"
	"errors"
	"fmt"
	// "log"
	// "time"
	// "os"
	// "github.com/paddie/goamz/s3"
	// "io/ioutil"
	// "path"
	// "path/filepath"
	// "strconv"
	// "strings"
	// "sync"
)

var (
	NoInfo = errors.New("No Info file exists")
)

// Every value is zeroed, so no need to set them
func NewContext() *Context {
	return &Context{}
}

type Context struct {
	RCID  int // reference checkpoint id - the id of the current reference checkpoint
	DCNT  int // delta count - the number of delta checkpoints since the last reference checkpoint
	MCNT  int // mutable checkpoints since the last reference checkpoint
	CtxID int //  0 or 1
	Type  int
}

func (ctx *Context) newDeltaContext() *Context {
	if ctx.RCID == 0 {
		panic("Attempted to commit incremental checkpoint without a reference checkpoint")
	}
	tmp := ctx.Copy()
	tmp.Type = DELTACPT
	tmp.MCNT += 1
	tmp.FlipCtxID()
	// tmp.info.mcnt++
	// the copied value from the old ctx
	// has this value to true (apart from in the initial case)
	// tmp.committed = false
	return tmp
}

func (ctx *Context) newZeroContext() *Context {
	tmp := ctx.Copy()
	tmp.Type = ZEROCPT
	tmp.FlipCtxID()
	tmp.RCID += 1
	tmp.MCNT = 1
	tmp.DCNT = 0
	return tmp
}

func (ctx *Context) Copy() *Context {
	// take lock to make sure that the updated cpt_id is
	// updates atomically
	tmp := *ctx
	return &tmp
}

// Returns the most recent of the two provided contexts
// by comparing first RCID and then MCNT
func MostRecent(c1, c2 *Context) *Context {
	if c1.RCID > c2.RCID {
		return c1
	}

	if c1.RCID < c2.RCID {
		return c2
	}

	if c1.MCNT > c2.MCNT {
		return c1
	}

	return c2
}

func (ctx *Context) ID() string {
	return fmt.Sprintf("%d.%d", ctx.RCID, ctx.MCNT)
}

func (ctx *Context) ImmPath() string {
	return fmt.Sprintf("%d/imm.cpt", ctx.RCID)
}

func (ctx *Context) MutPath() string {
	return fmt.Sprintf("%d/mut_%d.cpt", ctx.RCID, ctx.MCNT)
}

func (ctx *Context) DelPath() string {
	return fmt.Sprintf("%d/del_%d.cpt", ctx.RCID, ctx.DCNT)
}

func (ctx *Context) CtxPath() string {
	return fmt.Sprintf("cpt%d.nfo", ctx.CtxID)
}

func (ctx *Context) DeltaPaths() []string {
	paths := make([]string, 0, ctx.DCNT)
	for i := 1; i <= ctx.DCNT; i++ {
		paths = append(paths, fmt.Sprintf("%d/del_%d.cpt", ctx.RCID, i))
	}
	return paths
}

func (c *Context) FlipCtxID() {
	if c.CtxID == 1 {
		c.CtxID = 0
	} else {
		c.CtxID = 1
	}
}
