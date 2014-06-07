package statedb

type stateOperation struct {
	kt     *KeyType
	imm    []byte
	mut    *MutState
	action int
	err    chan error
}

// When called, all checkpointing is shut down,
// and a final, full checkpoint is written to disk.
func (db *StateDB) FinalCommit() error {

	// force a zero checkpoint and wait
	// till the checkpoint has been fully committed
	err := db.forceZeroCPTBlock()
	if err != nil {
		return err
	}
	fmt.Println("sending quit signal..")
	// send an error chan on which to respond in the
	// case of shut down issues
	errChan := make(chan error)
	db.quit <- errChan

	return <-errChan
}

func (db *StateDB) forceZeroCPTBlock() error {

	errChan := make(chan error)
	commitBlock := make(chan error)

	c := timeline.Tick()
	c.SyncStart()
	m := &msg{
		time:     time.Now(),
		err:      errChan,
		forceCPT: true,
		cptType:  ZEROCPT,
		t:        c,
		waitChan: commitBlock,
	}
	db.sync_chan <- m

	if err := <-errChan; err != nil {
		if err == ActiveCommitError {
			// if we sent a final commit while another commit
			// was being checkpointed, the commitBlock
			// will be signalled once *any* commit returns
			<-commitBlock
			db.sync_chan <- m
			err = <-errChan
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	c.SyncEnd()
	fmt.Println("waiting for commit")
	err := <-commitBlock
	fmt.Println("Complete was final!")
	// block until the final checkpoint is completed
	return err
}

func (db *StateDB) forceZeroCPT() error {
	errChan := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:     time.Now(),
		err:      errChan,
		forceCPT: true,    // don't query schedular
		cptType:  ZEROCPT, // force a zero checkpoint
		t:        c,
	}
	c.SyncEnd()
	return <-errChan
}

func (db *StateDB) ForceDeltaCPT() error {
	err := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:     time.Now(),
		err:      err,
		forceCPT: true,
		cptType:  DELTACPT,
		t:        c,
	}
	c.SyncEnd()
	return <-err
}

func (db *StateDB) ForceCheckpoint() error {
	err := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:     time.Now(),
		err:      err,
		forceCPT: true,
		cptType:  NONDETERMCPT,
		t:        c,
	}
	e := <-err
	c.SyncEnd()
	if e == ActiveCommitError {
		return nil
	}
	return e
}

// Sync is a call for consistency; if the monitor has signalled a checkpoint
// a checkpoint will be committed. During this time, we cannot allow any processes to
// write or delete objects in the database.
func (db *StateDB) PointOfConsistency() error {
	// response channel
	err := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:    time.Now(),
		err:     err,
		t:       c,
		cptType: NONDETERMCPT,
	}
	e := <-err
	c.SyncEnd()
	if e == ActiveCommitError {
		return nil
	}
	return e
}
