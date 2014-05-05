package statedb

import (
	"fmt"
	"time"
)

const (
	ZEROCPT  = 10
	DELTACPT = 5
)

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

func (c *CommitResp) Success() bool {
	return c.del_err == nil && c.mut_err == nil && c.imm_err == nil && c.ctx_err == nil
}

func (r *CommitResp) Error() string {
	if r.cpt_type == ZEROCPT {
		return fmt.Sprintf("ZEROCPT:\n\timmutable: %s\n\tmutable: %s", r.imm_err.Error(), r.mut_err.Error())
	} else {
		return fmt.Sprintf("∆CPT:\n\t∆: %s\n\tmut: %s", r.del_err.Error(), r.mut_err.Error())
	}
}

type TimedCommit struct {
	dur time.Duration
	err error
}

func commitLoop(fs Persistence, req chan *CommitReq, comResp chan *CommitResp) {

	t_comm := make(chan *TimedCommit)
	for r := range req {

		c := &CommitResp{
			cpt_type: r.cpt_type,
			ctx:      r.ctx,
		}

		// send the values on different goroutines
		// to minimize blocking
		// errChan := make(chan error)
		go async_commit(fs, r.ctx.MutPath(), r.mut, t_comm)
		// commit either the immutable or the delta depending on the type of
		// checkpoint
		if r.cpt_type == ZEROCPT {
			fmt.Println("Received ZEROCPT")
			c.imm_dur, c.imm_err = commit_t(fs, r.ctx.ImmPath(), r.imm)
		} else if r.del != nil {
			fmt.Println("Received ∆CPT")
			c.del_dur, c.del_err = commit_t(fs, r.ctx.DelPath(), r.del)
		}
		// retrieve any error from committing the mutable
		tm := <-t_comm
		c.mut_dur, c.mut_err = tm.dur, tm.err

		// ctx_err is nil now
		if !c.Success() {
			comResp <- c
			continue
		}
		// encode the context and flip-flop to disk
		c.ctx_err = commitContext(fs, r.ctx)

		// send back commit errors if any
		fmt.Println("Comitted!")
		comResp <- c
	}

	fmt.Println("commitLoop exiting..")

	close(comResp)
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
