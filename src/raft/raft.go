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

import (
	//	"bytes"
	"sync"
	"sync/atomic"
	"time"

	// "6.824/labgob"
	"6.824/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D: 日志压缩
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const (
	FAILED                 = -1
	LEADER                 = 1
	CANDIDATE              = 2
	FOLLOWER               = 3
	BROADCAST_TIME         = 100  // 限制为每秒十次心跳
	ELECTION_TIMEOUT_BASE  = 1000 // broadcastTime ≪ electionTimeout ≪ MTBF
	ELECTION_TIMEOUT_RANGE = 1000
)

type LogEntry struct {
	Command interface{}
	Term    int
	Index   int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	applych chan ApplyMsg // channel,用于发送提交的日志
	cond    *sync.Cond    // 唤醒线程
	// quickCheck rune
	// name string
	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// initialize in Make()
	BroadcastTime   int // 心跳间隔
	ElectionTimeout int // 随机选举超时
	Log             []LogEntry
	Role            int // 角色
	CurrentTerm     int
	VotedFor        int
	NextIndex       []int // 即将发送到server的日志
	MatchIndex      []int // 已经复制到server的日志

	// lazy init
	CommitIndex       int // 已提交的日志
	LastApplied       int // 已应用的日志
	LastIncludedIndex int // 快照压缩替换掉的之前的索引
	LastIncludedTerm  int // 快照压缩替换掉的索引的term
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	if rf.killed() {
		return FAILED, false
	}
	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	term = rf.CurrentTerm
	isleader = rf.Role == LEADER
	rf.mu.Unlock()
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
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

// restore previously persisted state.
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

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	// candidate request vote
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	// replier's msg
	Term        int
	VoteGranted bool // candidate是否获得投票
}

// appendEntries RPC
type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int // leader's commitIndex
}

type AppendEntriesReply struct {
	Term    int
	Success bool
	// conflict entry
	XTerm  int // term in the conflicting entry
	XIndex int // index of first entry with XTerm
	XLen   int // length of the conflicting entry
}

// snapshot RPC

type InstallSnapshotArgs struct{}

type InstallSnapshotReply struct{}

// example RequestVote RPC handler.
/*
1. Reply false if term < currentTerm
2. If votedFor is null or candidateId, and candidate’s log is at
	least as up-to-date as receiver’s log, grant vote
*/
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	// rf.DPrintf("[%d] Get RequestVote from %d", rf.me, args.CandidateId)
	defer rf.mu.Unlock()
	reply.Term = rf.CurrentTerm
	LastIndex := rf.LastLogEntry().Index
	LastTerm := rf.LastLogEntry().Term
	// term>candidate's term, 拒绝投票
	if rf.CurrentTerm > args.Term {
		reply.VoteGranted = false
		return
	} else if rf.CurrentTerm < args.Term {
		// convert to follower
		rf.Role = FOLLOWER
		rf.CurrentTerm = args.Term // candidate's term
		rf.VotedFor = -1           // 重置选票，中间状态
		rf.persist()
	}
	// candidate's log is at least as up-to-date as receiver's log
	var ValidateLeader func() bool
	ValidateLeader = func() bool {
		if LastTerm < args.LastLogTerm || (LastTerm == args.LastLogTerm && LastIndex <= args.LastLogIndex) {
			return true
		}
		return false
	}
	// 检查选票是否存在
	if rf.VotedFor == -1 && ValidateLeader() {
		reply.VoteGranted = true
		rf.VotedFor = args.CandidateId
		rf.persist() // 持久化保存
		// 投票成功，重置选举超时，防止不符合的candidate阻塞潜在leader
		rf.ElectionTimeout = GetElectionTimeout()
		rf.DPrintf("[%d %d] vote for %d, term = %d", rf.me, rf.Role, args.CandidateId, rf.CurrentTerm)
	}
	// if rf.currentTerm == args.Term，直接结束
}

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
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

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
// 客户端向Raft服务器发送命令，创建日志条目并插入本地日志
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	// 请求失败
	_, isLeader := rf.GetState()
	if !isLeader || rf.killed() {
		rf.DPrintf("[%d] Start() Fail isleader = %t, isKilled = %t", rf.me, isLeader, rf.killed())
		return -1, -1, false
	}

	// Your code here (2B).

	return index, term, true
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

// 判断goroutines是否kill,需要lock
func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	// 无限执行raft集群任务
	for rf.killed() == false {
		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		rf.mu.Lock()
		rf.UpdateLastApplied()
		// 在任务函数中处理完元数据 / 在耗时操作前 记得解锁，防止死锁
		switch rf.Role {
		case LEADER:
			rf.DoLeaderTask()
		case CANDIDATE:
			rf.DoCandidateTask()
		case FOLLOWER:
			if !rf.DoFollowerTask() {
				continue // 转换为candidate
			}
		default:
			rf.mu.Unlock()
		}

	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}

	rf.mu.Lock()

	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applych = applyCh
	rf.cond = sync.NewCond(&rf.mu)
	rf.DPrintf("[%d] is Making, len(peers) = %d\n", me, len(peers))

	// Your initialization code here (2A, 2B, 2C).
	rf.BroadcastTime = BROADCAST_TIME
	rf.ElectionTimeout = GetElectionTimeout() // 初始化，随机选举超时
	rf.Role = FOLLOWER                        // 初始化为follower
	rf.CurrentTerm = 0
	rf.VotedFor = -1 // 未投票
	rf.MatchIndex = make([]int, len(peers))
	rf.NextIndex = make([]int, len(peers))

	rf.mu.Unlock()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

// 自定义函数

func (rf *Raft) DoLeaderTask() {
	rf.mu.Unlock()
	rf.TrySendEntries(false) // false代表是否为leader第一次调用该函数
	rf.UpdateCommitIndex()
	time.Sleep(time.Duration(rf.BroadcastTime) * time.Millisecond) // 睡眠，实现心跳间隔
}

func (rf *Raft) DoFollowerTask() bool {
	rf.DPrintf("[%d] 's ElectionTimeout = %d\n", rf.me, rf.ElectionTimeout)
	// 检查是否即将超时，electionTimeout<100ms
	if rf.ElectionTimeout < rf.BroadcastTime {
		rf.Role = CANDIDATE
		rf.DPrintf("[%d] is ElectionTimeout, convert to CANDIDATE\n", rf.me)
		rf.mu.Unlock()
		return false
	}
	// 未超时，继续等待
	rf.ElectionTimeout -= rf.BroadcastTime
	rf.mu.Unlock()
	time.Sleep(time.Duration(rf.BroadcastTime) * time.Millisecond) // 心跳间隔睡眠
	return true
}

// leader election
/*
To begin an election, a follower increments its current
term and transitions to candidate state. It then votes for
itself and issues RequestVote RPCs in parallel to each of
the other servers in the cluster. A candidate continues in
this state until one of three things happens: (a) it wins the
election, (b) another server establishes itself as leader, or
(c) a period of time goes by with no winner. These out-
comes are discussed separately in the paragraphs below.
*/
func (rf *Raft) DoCandidateTask() {
	// prepare for election
	rf.CurrentTerm++
	votesGet := 1       // 得票数
	rf.VotedFor = rf.me // 投票给自己
	rf.persist()
	rf.ElectionTimeout = GetElectionTimeout() // 重置选举超时
	term := rf.CurrentTerm
	electionTimeout := rf.ElectionTimeout
	lastLogIndex := rf.LastLogEntry().Index
	lastLogTerm := rf.LastLogEntry().Term
	rf.DPrintf("[%d] start election, term = %d\n", rf.me, term)
	rf.mu.Unlock()
	// 阶段1：并发goroutine：RequestVote RPC
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			// 并发执行goroutine，发送requestVote RPC请求，且不得持有锁
			go func(server int) {
				args := RequestVoteArgs{
					Term:         term,
					CandidateId:  rf.me,
					LastLogIndex: lastLogIndex,
					LastLogTerm:  lastLogTerm,
				}
				reply := RequestVoteReply{}
				rf.DPrintf("[%d] send RequestVote to server [%d]\n", rf.me, server)
				// 发送失败，终止对应线程
				if !rf.sendRequestVote(server, &args, &reply) {
					rf.cond.Broadcast() // 唤醒rf.cond的goroutine
					return
				}
				rf.mu.Lock()
				defer rf.mu.Unlock()
				// check 是否获得选票
				if reply.VoteGranted {
					votesGet++
				}
				// 立刻转换为follower并重置超时
				if reply.Term > rf.CurrentTerm {
					rf.CurrentTerm = reply.Term
					rf.Role = FOLLOWER
					rf.ElectionTimeout = GetElectionTimeout()
					rf.persist()
				}
				rf.cond.Broadcast()
			}(i)
		}
	}
	// 阶段2：goroutine超时唤醒
	// 超时唤醒goroutine，并唤醒主线程提醒超时
	var timeout rune
	go func(electionTimeout int, timeout *rune) {
		time.Sleep(time.Duration(electionTimeout) * time.Millisecond)
		// 原子操作将timeout置为1，保证在多线程环境下安全操作共享变量
		atomic.StoreInt32(timeout, 1)
		rf.cond.Broadcast()
	}(electionTimeout, &timeout)

	// 阶段3：主线程循环判断选举是否结束
	// 判断currentTerm, State, ifTimeout
	var validateElectionState func() bool
	validateElectionState = func() bool {
		if rf.CurrentTerm == term &&
			rf.Role == CANDIDATE &&
			atomic.LoadInt32(&timeout) == 0 {
			return true
		}
		return false
	}
	for {
		rf.mu.Lock()
		// 选举尚未结束
		if votesGet <= len(rf.peers)/2 && validateElectionState() {
			rf.cond.Wait() // 主线程睡眠
		}
		if !validateElectionState() {
			rf.mu.Unlock()
			break
		}
		// 成为leader
		if votesGet > len(rf.peers)/2 {
			rf.DPrintf("[%d] is voted as Leader, term is [%d]\n", rf.me, rf.CurrentTerm)
			rf.Role = LEADER
			// initialize leader's log state
			rf.CommitIndex = 0
			for i := 0; i < len(rf.peers); i++ {
				rf.MatchIndex[i] = 0
				rf.NextIndex[i] = rf.LastLogEntry().Index + 1
			}
			rf.mu.Unlock()
			rf.TrySendEntries(true) // heartbeats
			break
		}
		rf.mu.Unlock()
	}
	rf.DPrintf("Candidate [%d] finishes election", rf.me)
}

/*
If commitIndex > lastApplied: increment lastApplied, apply
log[lastApplied] to state machine
*/
func (rf *Raft) UpdateLastApplied() {

}

/*
If there exists an N such that N > commitIndex, a majority
of matchIndex[i] ≥ N, and log[N].term == currentTerm:
set commitIndex = N
*/
func (rf *Raft) UpdateCommitIndex() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	newCommitIndex := rf.CommitIndex
	for N := rf.CommitIndex + 1; N < rf.LastLogEntry().Index; N++ {
		if N > rf.LastIncludedIndex && rf.Log[N].Term == rf.CurrentTerm {
			count := 1
			for i := 0; i < len(rf.peers); i++ {
				// 已经复制过的日志
				if rf.MatchIndex[i] >= N {
					count++
					// 超过半数即可
					if count > len(rf.peers)/2 {
						newCommitIndex = N
						break
					}
				}
			}
		}
	}
	// 新的需要提交的日志
	rf.CommitIndex = newCommitIndex
	rf.DPrintf("[%d] update CommitIndex, term = %d, NextIndex is %v, MatchIndex is %v, CommitIndex is %d", rf.me, rf.CurrentTerm, rf.NextIndex, rf.MatchIndex, rf.CommitIndex)
}

// 尝试执行心跳rpc / 日志复制 / 快照复制
// initialize: 选上后的初次调用
func (rf *Raft) TrySendEntries(initialize bool) {
	for i := 0; i < len(rf.peers); i++ {
		rf.mu.Lock()
		nextIndex := rf.NextIndex[i]
		firstLogIndex := rf.FirstLogEntry().Index
		lastLogIndex := rf.LastLogEntry().Index
		rf.mu.Unlock()
		if i != rf.me {
			// 成为leader后初次调用
			// lastLogIndex >= nextIndex，说明本地有新的日志需要复制到目标节点
			if lastLogIndex >= nextIndex || initialize {
				// 执行日志复制更新目标节点的日志状态
				if firstLogIndex <= nextIndex {
					go rf.SendEntries(i)
				} else {
					// 目标节点的日志远远落后，需要发送snapshot来更新状态
					go rf.SendSnapshot()
				}
			} else {
				// 没有日志要发送，就heartbeat保活
				go rf.SendHeartbeat()
			}
		}
	}
}

// heartbeat
func (rf *Raft) SendHeartbeat() {}

// 日志复制
/*
The consistency check acts as an induction
step: the initial empty state of the logs satisfies the Log
Matching Property, and the consistency check preserves
the Log Matching Property whenever logs are extended.
As a result, whenever AppendEntries returns successfully,
the leader knows that the follower’s log is identical to its
own log up through the new entries.
*/

func (rf *Raft) SendEntries(server int) {
	finish := false
	// 判断当前是否仍未leader,以及有无新的日志需要发送
	for !finish {
		rf.mu.Lock()
		if rf.Role != LEADER {
			rf.mu.Unlock()
			return
		}
		if rf.NextIndex[server] <= rf.LastIncludedIndex {
			rf.mu.Unlock()
			return
		}
		finish = true
		currentTerm := rf.CurrentTerm
		leaderCommit := rf.CommitIndex
		prevLogIndex := rf.NextIndex[server] - 1
		prevLogTerm := rf.GetLogIndex(prevLogIndex).Term
		// 需要发送的日志
		entries := rf.Log[prevLogIndex-rf.LastIncludedIndex+1:]
		rf.DPrintf()
		rf.mu.Unlock()
		args := AppendEntriesArgs{
			Term:         currentTerm,
			LeaderId:     rf.me,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      entries,
			LeaderCommit: leaderCommit}
		reply := AppendEntriesReply{}
		//  try appendEntries rpc
		if !rf.sendAppendEntries(server, &args, &reply) {
			return
		}
		rf.mu.Lock()
		// when try to send entries and find a larger term
		// current peer convert to follower and update term equal to candidate's term
		if reply.Term > rf.CurrentTerm {
			rf.CurrentTerm = reply.Term
			rf.Role = FOLLOWER
			rf.ElectionTimeout = GetElectionTimeout()
			rf.VotedFor = -1 // reset vote
			rf.persist()
			rf.mu.Unlock()
			return
		}
		// If AppendEntries fails because of log inconsistency: decrement nextIndex and retry
		if !reply.Success {
			// case 1: follower's log is too short
			if reply.XLen < prevLogIndex {
				rf.NextIndex[server] = Max(reply.XLen, 1) // prevent nextIndex < 0
			} else {
				newNextIndex := prevLogIndex
				for newNextIndex > rf.LastIncludedIndex &&
					rf.GetLogIndex(newNextIndex).Term > reply.XTerm {
					newNextIndex--
				}
				//  case 2: leader has xTerm 
				if rf.GetLogIndex(newNextIndex).Term == reply.XTerm {
					rf.NextIndex[server] = Max(newNextIndex, rf.LastIncludedIndex+1)
				} else {
					// case 3: leader dont have xTerm
					rf.NextIndex[server] = reply.XIndex
				}
			}
			rf.DPrintf()
			finish = false
		} else {
			// appendEntries success
			rf.NextIndex[server] = Max(rf.NextIndex[server], prevLogIndex+len(entries)+1)
			rf.MatchIndex[server] = Max(rf.MatchIndex[server], prevLogIndex+len(entries))
			rf.DPrintf()
		}
		rf.mu.Unlock()
	}
}

// 快照复制
func (rf *Raft) SendSnapshot() {}
