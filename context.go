package statedb

import (
	// "encoding/gob"
	// "errors"
	"fmt"
	"log"
	// "time"
	// "os"
	"path"
	"sync"
)

// // Volume: Amazon bucket identifier or Volume
// // Path:   Path to file in bucket
// // Suffix:  Combination of Machine and application maybe?
// type ctx_directory struct {
// 	volume, dir, suffix string // required for committing and recovering state
// 	full                string // <bucket>/<dir>/<suffix>
// 	cpt_dir             string // <bucket>/<dir>/<suffix>/<cpt_id>
// }

// // the current state of the context
// // - cpt_id: 0 means no reference checkpoint has been committed, increases with every checkpoint
// // - delta_cnt:
// type ctx_status struct {
// 	restore   bool // does the system need to be restored before using it?
// 	cpt_id    int  // the id of the current reference checkpoint
// 	delta_cnt int  // the number of delta checkpoints since the last reference checkpoint
// }

// type ctx_stats struct {
// 	LastCpt          time.Time
// 	Emax, Emin, Eavg time.Duration // Encoding stats
// 	Dmax, Dmin, Davg time.Duration // Decoding stats
// 	Rmax, Rmin, Ravg time.Duration // Reading stats
// 	Wmax, Wmin, Wavg time.Duration // Writing stats
// 	TMax, TMin, TAvg time.Duration // Total stats
// }

// type ctx struct {
// 	dir    ctx_directory
// 	status ctx_status
// 	stats  ctx_stats
// 	// cpt_id     int    // the id of the last full update
// 	// delta_diff int    // number of delta checkpoints relative to full checkpoint
// 	// full       string // <bucket>/<dir>/<suffix>
// 	// cpt_dir    string // <bucket>/<dir>/<suffix>/<cpt_id>
// 	// restore    bool   // is true as long as a previous checkpoint has not been restored
// 	sync.RWMutex
// }

type Context struct {
	bucket, dir, suffix string
	// status ctx_status
	// stats  ctx_stats
	cpt_id     int    // the id of the last full update
	delta_diff int    // number of delta checkpoints relative to full checkpoint
	full       string // <bucket>/<dir>/<suffix>
	cpt_dir    string // <bucket>/<dir>/<suffix>/<cpt_id>
	restore    bool   // is true as long as a previous checkpoint has not been restored
	sync.RWMutex
}

// Bucket: Amazon bucket identifier
// Path:   Path to file in bucket
// AppID:  Combination of Machine and application maybe?
func NewContext(bucket, dir, suffix string) (*Context, bool, error) {

	if dir == "" {
		return nil, false, fmt.Errorf("NewContext: bucket='%s' path='%s' suffix='%s'", bucket, dir, suffix)
	}

	// clean the parameters
	dir = path.Clean(dir)
	full := path.Join(bucket, dir, suffix)

	ctx := &Context{
		dir:    dir,
		bucket: bucket,
		suffix: suffix,
		full:   full,
	}

	// Create checkpoint directories. If some already exist at this checkpoint, signal a restore..
	if err := ctx.initContext(); err != nil {
		fmt.Println("No reason for restore")
		return nil, false, err
	}

	fmt.Printf("signalling restore %v: id %d\n", ctx.restore, ctx.cpt_id)

	if ctx.cpt_id != 0 {
		fmt.Printf("Previous checkpoint id=%d detected\n", ctx.cpt_id)
	} else {
		log.Printf("No previous checkpoint detected\n")
	}

	return ctx, ctx.restore, nil
}

// If any existing checkpoints exist, a restore is signalled.
// If no previous checkpoints exist, the checkpoint dir is created.
func (ctx *Context) initContext() error {
	// make sure that the checkpoint dir exists
	if !IsDir(ctx.full) {
		log.Println("Creating Checkpoint dir at " + ctx.full)
		return CreateDir(ctx.full)
	}

	fmt.Println("Probing existing checkpoint..")

	// note that there there might be a previous backup
	// TODO: update to something that describes the difference
	//       in context after this call..
	id := ctx.probeCptId()
	if id == 0 {
		return nil
	}

	ctx.setId(id)

	fmt.Printf("Previous checkpoint id found: %d\n", id)

	valid := IsValidCheckpoint(ctx.cpt_dir)

	// if the previous cpt is valid, signal a restore

	if id != 0 && valid {
		ctx.restore = true
		ctx.cpt_id = id
	}

	fmt.Println("Previous checkpoint appears to be valid..")

	return nil
}

func (ctx *Context) setId(id int) {
	ctx.cpt_id = id
	ctx.cpt_dir = path.Join(ctx.full, fmt.Sprintf("%010d", id))
}

func (ctx *Context) newContextWithId(id int) *Context {
	tmp := *ctx
	tmp.setId(id)

	return &tmp
}

func (ctx *Context) prepareDirectories() error {
	err := CreateDir(ctx.cpt_dir)
	if err != nil {
		return err
	}
	return nil
}

// Increase the checkpoint id and create the necessary subfolder
func (ctx *Context) copyContext() *Context {
	// take lock to make sure that the updated cpt_id is
	// updates atomically
	tmp := *ctx
	return &tmp
}

func (ctx *Context) MutablePath() string {
	return ctx.cpt_dir + "/mutable.cpt"
}

func (ctx *Context) ImmutablePath() string {
	return ctx.cpt_dir + "/immutable.cpt"
}

func (ctx *Context) DeltaPath() string {
	return ctx.cpt_dir + "/delta.cpt"
}
