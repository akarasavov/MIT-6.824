package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import "sync"
import (
	"labrpc"
	"time"
	"fmt"
)

// import "bytes"
// import "labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	currentTerm int
	voteFor     int
	state       int

	timer *time.Timer

	becomeFlw    chan struct{}
	canBeALeader chan struct{}
	stateHandler func()
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	var term = rf.currentTerm
	var isleader = rf.state == LEADER
	// Your code here (2A).
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term        int
	CandidateId int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

type AppendEntitiesArgs struct {
	Term     int
	LeaderId int
}

type AppendEntitiesReply struct {
	Term    int
	Success bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	candidate := args.CandidateId
	fmt.Printf("server{%v} receive RequestVoteArgs term{%v} candidateId{%v} \n", rf.me, args.Term, candidate)

	reply.VoteGranted = false
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		fmt.Printf("server{%v} RequestVoteReply term{%v} voteGranted{%v} for candidate{%v} \n", rf.me, reply.Term, reply.VoteGranted, candidate)
		return
	}
	becomeFollower := false
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.voteFor = -1
		becomeFollower = true
		reply.Term = rf.currentTerm
	}

	canVoteForCandidate := rf.voteFor == -1 || rf.voteFor == candidate
	if canVoteForCandidate {
		reply.VoteGranted = true
		rf.voteFor = candidate
		becomeFollower = true
	}
	if becomeFollower {
		go rf.becomeFollower()
	}
	fmt.Printf("server{%v} RequestVoteReply term{%v} voteGranted{%v} for candidate{%v} \n", rf.me, reply.Term, reply.VoteGranted, candidate)

}

func (rf *Raft) broadcastRequestVote() {
	rf.voteFor = rf.me

	count := 1
	handelReply := func(reply *RequestVoteReply) {
		rf.mu.Lock()
		defer rf.mu.Unlock()

		if rf.state == CANDIDATE {
			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				go rf.becomeFollower()
			} else if reply.VoteGranted {
				if count >= len(rf.peers)/2 {
					go rf.becomeLeader()
				}
				count++
			}
		}
	}

	for i := range rf.peers {
		if i != rf.me {
			args := RequestVoteArgs{rf.currentTerm, rf.me}
			go func(server int, args RequestVoteArgs) {
				reply := RequestVoteReply{-1, false}
				if rf.sendRequestVote(server, &args, &reply) {
					handelReply(&reply)
				}
			}(i, args)
		}
	}

}

func (rf *Raft) AppendEntities(args *AppendEntitiesArgs, reply *AppendEntitiesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	fmt.Printf("server{%v} receive AppendEntities term{%v} LeaderId{%v} \n", rf.me, args.Term, args.LeaderId)

	if rf.currentTerm > args.Term {
		reply.Success = false
		reply.Term = rf.currentTerm
		fmt.Printf("AppendEntitiesReply term{%v} sucess{%v} \n", reply.Term, reply.Success)
		return
	}

	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		reply.Success = true
		reply.Term = args.Term

		go rf.becomeFollower()

		fmt.Printf("AppendEntitiesReply term{%v} sucess={%v} \n", reply.Term, reply.Success)
	}

}

func (rf *Raft) broadcastAppendEntities() {
	replyHandler := func(args *AppendEntitiesArgs, reply *AppendEntitiesReply) {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if rf.state == LEADER && rf.currentTerm == args.Term {
			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.voteFor = -1
				go rf.becomeFollower()
				return
			}
		}
	}

	for i := range rf.peers {
		if i != rf.me {
			args := AppendEntitiesArgs{rf.currentTerm, rf.me}
			go func(server int, args AppendEntitiesArgs) {
				reply := AppendEntitiesReply{-1, false}
				if rf.sendAppendEntities(server, &args, &reply) {
					replyHandler(&args, &reply)
				}
			}(i, args)
		}
	}
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntities(server int, args *AppendEntitiesArgs, reply *AppendEntitiesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntities", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.voteFor = -1
	rf.currentTerm = 0
	rf.canBeALeader = make(chan struct{})
	rf.becomeFlw = make(chan struct{}, 1)
	rf.stateHandler = rf.createStateHandler()
	rf.timer = time.NewTimer(-1)

	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	go func() {
		rf.stopTimer()
		for {
			rf.stateHandler()
		}
	}()

	return rf
}
