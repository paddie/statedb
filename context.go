package statedb

import (
	// "encoding/gob"
	// "errors"
	"fmt"
	"log"
	// "time"
	// "os"
	"path"
	"path/filepath"
	"sync"
)

// // Volume: Amazon bucket identifier or Volume
// // Path:   Path to file in bucket
// // Suffix:  Combination of Machine and application maybe?
type Ctx_dir struct {
	volume, dir, suffix string // required for committing and recovering state
	full                string // <bucket>/<dir>/<suffix>
}

func NewCtx_dir(volume, dir, suffix string) (*ctx_dir, error) {

	// dir must be != ""
	if dir == "" {
		return nil, fmt.Errorf("ctx_dir: volume='%s' dir='%s' suffix='%s'", bucket, dir, suffix)
	}

	// Clean the paths
	dir = path.Clean(dir)
	volume = path.Clean(volume)
	suffix = path.Clean(suffix)

	// build the complete path
	full := path.Join(bucket, dir, suffix)

	return &ctx_dir{
		volume: bucket,
		dir:    dir,
		suffix: suffix,
		full:   full,
	}, nil
}

// type ctx_stats struct {
// 	LastCpt          time.Time
// 	Emax, Emin, Eavg time.Duration // Encoding stats
// 	Dmax, Dmin, Davg time.Duration // Decoding stats
// 	Rmax, Rmin, Ravg time.Duration // Reading stats
// 	Wmax, Wmin, Wavg time.Duration // Writing stats
// 	TMax, TMin, TAvg time.Duration // Total stats
// }

type Context struct {
	dir      ctx_dir
	restored bool // does the system need to be restored before using it?
	rcid     int  // reference checkpoint id - the id of the current reference checkpoint
	dcnt     int  // delta count - the number of delta checkpoints since the last reference checkpoint
	mcnt     int  // mutable checkpoints since the last reference checkpoint
	// stats  ctx_stats
	// cpt_id     int    // the id of the last full update
	// delta_diff int    // number of delta checkpoints relative to full checkpoint
	// full       string // <bucket>/<dir>/<suffix>
	// cpt_dir    string // <bucket>/<dir>/<suffix>/<cpt_id>
	// restore    bool   // is true as long as a previous checkpoint has not been restored
	sync.RWMutex
}

// Bucket: Amazon bucket identifier
// Path:   Path to file in bucket
// AppID:  Combination of Machine and application maybe?
func NewContext(dir ctx_dir) (*Context, bool, error) {

	if tmp, error := RebuildPreviousContext(dir); err == nil {
		return tmp, true, nil
	}
	// no previous checkpoints detected
	// rcid, dcnt and mcnt are zero at init
	// Create checkpoint directories. If some already exist at this checkpoint, signal a restore..
	if err := ctx.initContext(); err != nil {
		fmt.Println("No reason for restore")
		return nil, false, err
	}

	// fmt.Printf("signalling restore %v: id %d\n", ctx.restore, ctx.cpt_id)

	if ctx.rcid != 0 {
		fmt.Printf("Previous checkpoint id=%d detected\n", ctx.cpt_id)
	} else {
		log.Printf("No previous checkpoint detected\n")
	}

	return ctx, ctx.restored, nil
}

// If any existing checkpoints exist, a restore is signalled.
// If no previous checkpoints exist, the checkpoint dir is created.
func (ctx *Context) initContext() error {
	// make sure that the checkpoint dir exists
	if !IsDir(ctx.dir.full) {
		log.Println("Creating Checkpoint dir at " + ctx.dir.full)
		return CreateDir(ctx.dir.full)
	}

	fmt.Println("Probing for existing checkpoints..")

	// note that there there might be a previous backup
	// TODO: update to something that describes the difference
	//       in context after this call..
	id := ctx.probeCptId()
	if id == 0 {
		return nil
	}

	ctx.setId(id)

	valid := IsValidCheckpoint(ctx.cpt_dir)

	if id != 0 && valid {
		ctx.restore = true
		ctx.cpt_id = id
	}

	fmt.Println("Previous checkpoint appears to be valid..")

	return nil
}

func (ctx *Context) newContextWithId(id int) *Context {
	tmp := *ctx
	tmp.setId(id)
	return &tmp
}

func (ctx *Context) newIncrementalContext(db *StateDB) *Context {
	tmp := *ctx
	// increase DCNT if there is something in the delta log
	if len(db.delta) > 0 {
		tmp.dcnt++
	}
	tmp.mcnt++
	return &tmp
}

func (ctx *Context) newRelativeContext(db *StateDB) *Context {

	tmp := *ctx
	// increase DCNT if there is something in the delta log
	tmp.rcid++
	tmp.mcnt = 0
	tmp.dcnt = 0
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
func (ctx *Context) Copy() *Context {
	// take lock to make sure that the updated cpt_id is
	// updates atomically
	tmp := *ctx
	return &tmp
}

// TODO: cache the checkpoint dir
func (ctx *Context) CheckpointDir() string {
	return path.Join(ctx.dir.full, fmt.Sprintf("%010d", ctx.rcid))
}

// Filename is generated by the MCNT (Mutable CouNT)
func (ctx *Context) MutablePath() string {
	return ctx.CheckpointDir() + fmt.Sprintf("/mutable_%d.cpt", ctx.mcnt)
}

func probeMCNT(dir string) int {

	files := ListGlob(`mutable_*.cpt`)
	if len(files) == 0 {
		return 0
	}

	var mcnts []int
	max := 0
	// path_max := ""
	for _, f := range files {
		if mcnt, err := strconv.Atoi(f[8 : len(str)-4]); err == nil {
			if mcnt > max {
				max = mcnt
				// path_max = f
			}
			// mcnts = append(mcnts, mcnt)
		}
	}

	if max == 0 {
		return 0
	}

	return max
}

func ListGlob(pattern string) []string {

	files, err := filepath.Glob(pattern)
	if err != nil {
		return []string{}
	}
	return files
}

func ListDeltas(path string) []string {

	files, err := filepath.Glob(`delta_*.cpt`)
	if err != nil {
		return []string{}
	}
	return files
}

func (ctx *Context) ImmutablePath() string {
	return ctx.CheckpointDir() + "/immutable.cpt"
}

// Filename is generated by the DCNT (Delta CouNT)
func (ctx *Context) DeltaPath() string {
	return ctx.CheckpointDir() + fmt.Sprintf("/delta_%d.cpt", ctx.dcnt)
}

func RebuildPreviousContext(dir Ctx_dir) (*Context, error) {
	rcid := probeRCID(dir.full)
	if id == 0 {
		return nil, fmt.Errorf("No previous checkpoint detected at %s", dir.full)
	}

	tmp := &Context{
		dir:  dir,
		rcid: rcid,
	}

	// 1. decode reference checkpoint
	dcnt := tmp.LatestMutable()
	// 2. Get mutable chackpoint to get the max DCNT

	// 3. build list of delta checkpoints and replay the changes in the database

	tmp.restored = true

	return nil
}

// When restoring, this helps identify the type of the final checkpoint.
func (ctx *Context) Type() string {
	if ctx.rcid > 0 {
		if ctx.mcnt > 0 {
			return "IncrementalCPT"
		}
		return "ReferenceCPT"
	}
	return "ReferenceCPT"
}

// Get the ID of the most recent full commit in the ctx.full folder.
// Returns 0 if no valid full commit exists in the context.
func ProbeRCID(path string) int {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return 0
	}

	if len(files) == 0 {
		return 0
	}

	max := -1
	folders := 0
	for _, f := range files {
		if f.IsDir() {
			folders++
			if id, err := strconv.Atoi(f.Name()); err == nil {
				if id > max {
					max = id
				}
			}
		}
	}
	// if none of the files in ctx.dir were cpt folders
	if folders == 0 || max <= 0 {
		return 0
	}

	// set current cpt id in context
	return max
}
