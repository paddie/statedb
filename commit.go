package statedb

import (
	"fmt"
	"time"
)

// Interface to checkpoint data to non-volatile memory
type Persistence interface {
	List(prefix string) ([]string, error) // list items in dir
	Put(name string, data []byte) error   // create/overwrite file
	Get(name string) ([]byte, error)      // get file
	Delete(path string) error             // delete file
	Init() error                          // ensure that directory/bucket exists
}

type CommitReq struct {
	cpt_type int
	ctx      *Context
	imm      []byte
	mut      []byte
	del      []byte
}

type CommitResp struct {
	cpt_type int
	ctx      *Context
	imm_err  error
	mut_err  error
	del_err  error
	ctx_err  error
	imm_dur  time.Duration
	mut_dur  time.Duration
	del_dur  time.Duration
}

func (r *CommitResp) Err() error {
	if r.cpt_type == ZEROCPT {
		return fmt.Errorf("ZEROCPT:\n\timmutable: %s\n\tmutable: %s", r.imm_err.Error(), r.mut_err.Error())
	} else {
		return fmt.Errorf("∆CPT:\n\t∆: %s\n\tmut: %s", r.del_err.Error(), r.mut_err.Error())
	}
}

func (c *CommitResp) Success() bool {
	return c.del_err == nil && c.mut_err == nil && c.imm_err == nil && c.ctx_err == nil
}

type TimedCommit struct {
	dur time.Duration
	err error
}

type CommitNexus struct {
	comReqChan  chan *CommitReq
	comRespChan chan *CommitResp
}

func NewCommitNexus() *CommitNexus {
	return &CommitNexus{
		comReqChan:  make(chan *CommitReq),
		comRespChan: make(chan *CommitResp),
	}
}

func (c *CommitNexus) Quit() {
	close(c.comReqChan)
}

func commitLoop(fs Persistence, cnx *CommitNexus) {

	t_comm := make(chan *TimedCommit)
	// for will loop until the channel is closed
	// by cnx.Close() which is called during the cleanup
	for r := range cnx.comReqChan {
		// note the start time for statistics purposes
		start := time.Now()
		c := &CommitResp{
			cpt_type: r.cpt_type,
			ctx:      r.ctx,
		}
		// send the values on different goroutines
		// to parallelize the writes
		go async_commit(fs, r.ctx.MutPath(), r.mut, t_comm)
		// commit either the immutable or the delta
		// depending on the type of checkpoint
		if r.cpt_type == ZEROCPT {
			fmt.Println("Received encoded ZEROCPT")
			c.imm_dur, c.imm_err = commit_t(fs, r.ctx.ImmPath(), r.imm)
		} else {
			fmt.Println("Received encoded ∆CPT")
			if r.cpt_type == DELTACPT {

				if r.del != nil {
					c.del_dur, c.del_err = commit_t(fs, r.ctx.DelPath(), r.del)
				}
			}
		}
		// wait for the mutable checkpoint to complete
		tm := <-t_comm
		c.mut_dur, c.mut_err = tm.dur, tm.err

		// ctx_err is nil now
		if !c.Success() {
			cnx.comRespChan <- c
			continue
		}

		// encode the context and flip-flop to disk
		c.ctx_err = commitContext(fs, r.ctx)
		// note the checkpoint time with the timeline
		timeline.Commit(start)
		fmt.Println("registered commit time")
		// send the durations back to statistics module
		cnx.comRespChan <- c
	}

	fmt.Println("Committer has shut down")
	// once the cnx.comReqCHan is closed
	// close the response channel
	close(cnx.comRespChan)
}

func commitContext(fs Persistence, ctx *Context) error {
	data, err := encode(ctx)
	if err != nil {
		return err
	}
	return commit(fs, ctx.CtxPath(), data)
}

func commit_t(fs Persistence, path string, data []byte) (time.Duration, error) {
	now := time.Now()

	err := fs.Put(path, data)

	return time.Now().Sub(now), err
}

func async_commit(fs Persistence, path string, data []byte, tc chan<- *TimedCommit) {
	dur, err := commit_t(fs, path, data)
	tc <- &TimedCommit{
		dur: dur,
		err: err,
	}
}

func commit(fs Persistence, path string, data []byte) error {
	return fs.Put(path, data)
}
