package raft

import (
	"time"
	"math/rand"
	"fmt"
)

const (
	FOLLOWER  = iota
	CANDIDATE = iota
	LEADER    = iota
)

func (rf *Raft) createStateHandler() func() {
	return func() {
		switch rf.state {
		case FOLLOWER:
			rf.stateHandler = rf.createFollowerHandler()
		case CANDIDATE:
			rf.stateHandler = rf.createCandidateHandler()
		case LEADER:
			rf.stateHandler = rf.createLeaderHandler()
		}
	}
}

func (rf *Raft) createFollowerHandler() func() {
	return func() {
		fmt.Printf("Initialize follower{%v} \n", rf.me)
		rf.resetTimer(electionTime())
		select {
		case <-rf.becomeFlw:
		case <-rf.timer.C:
			rf.changeState(CANDIDATE, func() {
				rf.currentTerm++
				rf.canBeALeader = make(chan struct{}, 1)
				rf.broadcastRequestVote()
			})
		}
	}
}

func (rf *Raft) createCandidateHandler() func() {
	return func() {
		rf.resetTimer(electionTime())
		select {
		case <-rf.becomeFlw:
			rf.changeState(FOLLOWER, func() {})
		case <-rf.canBeALeader:
			rf.changeState(LEADER, func() {
				rf.broadcastAppendEntities()
			})
		case <-rf.timer.C:
			rf.changeState(CANDIDATE, func() {
				rf.currentTerm++
				rf.broadcastRequestVote()
			})
		}

	}
}

func (rf *Raft) createLeaderHandler() func() {
	return func() {
		rf.resetTimer(time.Second / 11)
		select {
		case <-rf.becomeFlw:
			rf.changeState(FOLLOWER, func() {

			})
		case <-rf.timer.C:
			rf.changeState(LEADER, rf.broadcastAppendEntities)
		}
	}
}

func electionTime() time.Duration {
	f := time.Duration(rand.Int31n(300) + 300)
	return time.Duration(f * time.Millisecond)
}

func (rf *Raft) changeState(state int, f func()) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.state != state {
		rf.state = state
		rf.stateHandler = rf.createStateHandler()
		printServerState(state, rf)
	} else {
		printServerState(state, rf)
	}

	f()
}
func printServerState(state int, rf *Raft) {
	switch state {
	case FOLLOWER:
		fmt.Printf("server{%v} state is follower \n", rf.me)
	case CANDIDATE:
		fmt.Printf("server{%v} state is candidate \n", rf.me)
	case LEADER:
		fmt.Printf("server{%v} state is leader \n", rf.me)
	}
}

func (rf *Raft) resetTimer(duration time.Duration) {
	rf.stopTimer()
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.timer.Reset(duration) {
		panic("timer is running or not stop")
	}
}

func (rf *Raft) stopTimer() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.timer.Stop() {
		select {
		case <-rf.timer.C:
		default:
			return
		}
	}
}

func (rf *Raft) becomeFollower() {
	rf.stopTimer()
	rf.becomeFlw <- struct{}{}
}

func (rf *Raft) becomeLeader() {
	rf.stopTimer()
	rf.canBeALeader <- struct{}{}
}
