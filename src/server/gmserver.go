package server

import (
	"bytes"
	. "common"
	"container/list"
	"fmt"
	"net"
	"storage"
	"storage/memory"
	"strconv"
	"sync/atomic"
	"runtime"
	"time"
	"utils/logger"
)

var ( //const replys, to avoid building it every time
	wakeupReply = constructReply(NOOP, nil)
	nojobReply  = constructReply(NO_JOB, nil)
)

type Tuple struct {
	t0, t1, t2, t3, t4, t5 interface{}
}

type Event struct {
	tp            uint32
	args          *Tuple
	result        chan interface{}
	fromSessionId int64
	jobHandle     string
}

type Server struct {
	protoEvtCh     chan *Event
	startSessionId int64
	tryTimes       int
	maxProc		int
	lockMainProcess bool
	funcWorker     map[string]*JobWorkerMap
	worker         map[int64]*Worker
	client         map[int64]*Client
	workJobs       map[string]*Job
	funcTimeout    map[string]int
	jobStores      map[string]storage.JobQueue
}

func NewServer(tryTimes int, maxProc int, lockMainProcess bool) *Server {
	return &Server{
		funcWorker:     make(map[string]*JobWorkerMap),
		protoEvtCh:     make(chan *Event, 256),
		worker:         make(map[int64]*Worker),
		client:         make(map[int64]*Client),
		workJobs:       make(map[string]*Job),
		jobStores:      make(map[string]storage.JobQueue),
		funcTimeout:    make(map[string]int),
		startSessionId: 0,
		tryTimes:       tryTimes,
		maxProc: maxProc,
		lockMainProcess: lockMainProcess,
	}
}

func (server *Server) getJobStatus(e *Event) {
	var buffer bytes.Buffer
	buffer.WriteString("waiting:[")
	for key, jq := range server.jobStores {
		buffer.WriteString(fmt.Sprintf("%v:%v,", key, jq.Length()))
	}
	buffer.WriteString("]\n")

	buffer.WriteString(fmt.Sprintf("protoEvtCh:%v, working:%v", len(server.protoEvtCh), len(server.workJobs)))

	for k, j := range server.workJobs {
		buffer.WriteString(fmt.Sprintf("\n %v:%v,", k, j))
	}

	e.result <- buffer.String()
}

func (server *Server) removeJobDirect(e *Event) {

	_, ok := server.workJobs[e.args.t0.(string)]
	if ok {
		logger.Logger().I("remove job %v", e.args.t0.(string))
		delete(server.workJobs, e.args.t0.(string))
		e.result <- fmt.Sprintf("deleted %v yet", e.args.t0.(string))
		return
	}else{
		e.result <- fmt.Sprintf("not found %v", e.args.t0.(string))
		return
	}
}

func (server *Server) getFuncWorkerStatus(e *Event) {
	var buffer bytes.Buffer
	for key, jw := range server.funcWorker {
		to, ok := server.funcTimeout[key]
		if !ok {
			to = 0
		}
		buffer.WriteString(fmt.Sprintf("func %v to %v[", key, to))
		for it := jw.Workers.Front(); it != nil; it = it.Next() {
			buffer.WriteString(fmt.Sprintf("id:%v cid:%v ip:%v stats:%v,\n", it.Value.(*Worker).Connector.SessionId,
				it.Value.(*Worker).workerId,
				it.Value.(*Worker).Conn.RemoteAddr(),
				it.Value.(*Worker).status))
		}
		buffer.WriteString("]\n")
	}

	e.result <- buffer.String()
}

func (server *Server) getWorkerStatus(e *Event) {
	var buffer bytes.Buffer
	buffer.WriteString("work[")
	for key, clt := range server.worker {
		buffer.WriteString(fmt.Sprintf("id:%v cid:%v ip:%v stats:%v,\n", key, clt.workerId,
			clt.Conn.RemoteAddr(), clt.status))
	}
	buffer.WriteString("]\n")

	e.result <- buffer.String()
}

func (server *Server) getClientStatus(e *Event) {
	var buffer bytes.Buffer
	buffer.WriteString("client[")
	for key, wk := range server.client {
		buffer.WriteString(fmt.Sprintf("id:%v ip:%v,\n", key,
			wk.Conn.RemoteAddr()))
	}
	buffer.WriteString("]\n")

	e.result <- buffer.String()
}

func (server *Server) allocSessionId() int64 {
	return atomic.AddInt64(&server.startSessionId, 1)
}

func (server *Server) clearTimeoutJob() {

	now := time.Now().Unix()
	for k, j := range server.workJobs {
		if j.TimeoutSec > 0 {
			if (j.CreateAt.Unix() + int64(j.TimeoutSec)) <= now {
				c, ok := server.client[j.CreateBy]
				if ok {
					c.Send(constructReply(WORK_FAIL, [][]byte{[]byte(j.Handle)}))
				} else {
					logger.Logger().I("client not exist cant send %v", j)
				}
				delete(server.workJobs, k)
				logger.Logger().I("remove time out job %v", j)
			}
		}
	}
}

func (server *Server) Start(addr string, monAddr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Logger().E("listen %v", err)
	}

	logger.Logger().I("listening on %v", addr)
	go server.EvtLoop()

	go registerWebHandler(server, monAddr)

	for {
		conn, err := ln.Accept()
		if err != nil { // handle error
			logger.Logger().E("accept %v", err)
			continue
		}

		session := &Session{}
		go session.handleConnection(server, conn)
	}
}

func (server *Server) EvtLoop() {
	if server.maxProc > 1 && server.lockMainProcess {
		logger.Logger().I("EvtLoop LockOSThread")
		runtime.LockOSThread()
	}
	tick := time.NewTicker(2 * time.Second)
	for {
		select {
		case e := <-server.protoEvtCh:
			server.handleProtoEvt(e)
		case <-tick.C:
			server.clearTimeoutJob()
		}
	}
}

func (server *Server) addWorker(l *list.List, w *Worker) {
	for it := l.Front(); it != nil; it = it.Next() {
		if it.Value.(*Worker).SessionId == w.SessionId {
			logger.Logger().W("already add")
			return
		}
	}

	l.PushBack(w)
}

func (server *Server) getJobWorkPair(funcName string) *JobWorkerMap {
	jw, ok := server.funcWorker[funcName]
	if !ok { //create list
		jw = &JobWorkerMap{Workers: list.New()}
		server.funcWorker[funcName] = jw
	}

	return jw
}

func (server *Server) handleCanDo(funcName string, w *Worker, timeout int) {

	jw := server.getJobWorkPair(funcName)
	server.addWorker(jw.Workers, w)
	server.worker[w.SessionId] = w
	server.funcTimeout[funcName] = timeout
	w.canDo[funcName] = true

	logger.Logger().T("can do func:%v sessionId:%v", funcName, w.SessionId)
}

func (server *Server) addFuncJobStore(funcName string) storage.JobQueue {

	k, ok := server.jobStores[funcName]

	if ok {
		return k
	}

	queue := &memory.MemJobQueue{}
	queue.Initial(funcName)
	server.jobStores[funcName] = queue

	logger.Logger().T("addFuncJobStore:%v", funcName)
	return queue
}

func (server *Server) removeCanDo(funcName string, sessionId int64) {

	if jw, ok := server.funcWorker[funcName]; ok {
		server.removeWorker(jw.Workers, sessionId)
	}

	logger.Logger().T("removeCanDo:%v sessionId:%v", funcName, sessionId)
	delete(server.worker[sessionId].canDo, funcName)
}

func (server *Server) removeWorkerBySessionId(sessionId int64) {
	for _, jw := range server.funcWorker {
		server.removeWorker(jw.Workers, sessionId)
	}
	delete(server.worker, sessionId)
}

func (server *Server) removeWorker(l *list.List, sessionId int64) {
	for it := l.Front(); it != nil; it = it.Next() {
		if it.Value.(*Worker).SessionId == sessionId {
			logger.Logger().T("removeWorker sessionId %v %v", sessionId, it.Value.(*Worker).workerId)
			l.Remove(it)
			return
		}
	}
}

func (server *Server) popJob(sessionId int64) *Job {

	for funcName, cando := range server.worker[sessionId].canDo {
		if !cando {
			continue
		}

		if queue, ok := server.jobStores[funcName]; ok {
			if queue.Length() == 0 {
				continue
			}

			jb := queue.PopJob()
			if jb != nil {
				logger.Logger().T("pop job work:%v job:%v", sessionId, jb)
				return jb
			}
		}
	}

	return nil

}

func (server *Server) wakeupWorker(funcName string, w *Worker) bool {

	if w.status == wsRunning {
		return false
	}

	jq, ok := server.jobStores[funcName]
	if !ok || jq.Length() == 0 {
		return false
	}

	logger.Logger().T("wakeup sessionId: %v %v", w.SessionId, w.workerId)
	w.Send(wakeupReply)
	return true
}

func (server *Server) handleSubmitJob(e *Event) {
	args := e.args
	c := args.t0.(*Client)

	server.client[c.SessionId] = c

	funcName := bytes2str(args.t1)

	timeout := 0
	v, ok := server.funcTimeout[funcName]
	if ok {
		timeout = v
	}

	j := &Job{Id: bytes2str(args.t2), Data: args.t3.([]byte),
		Handle: allocJobId(), CreateAt: time.Now(), CreateBy: c.SessionId,
		FuncName: funcName, Priority: PRIORITY_LOW, TimeoutSec: timeout}

	j.IsBackGround = isBackGround(e.tp)

	logger.Logger().T("%v func:%v uniq:%v info:%+v", CmdDescription(e.tp),
		args.t1, args.t2, j)

	j.Priority = cmd2Priority(e.tp)

	//e.result <- j.Handle
	sendReply(c.in, JOB_CREATED, [][]byte{[]byte(j.Handle), []byte(j.Id)})

	server.doAddJob(j)
}

func (server *Server) doAddJob(j *Job) {

	queue := server.addFuncJobStore(j.FuncName)
	j.ProcessBy = 0
	queue.PushJob(j)
	workers, ok := server.funcWorker[j.FuncName]
	if ok {
		var i int = 0
		for it := workers.Workers.Front(); it != nil; it = it.Next() {
			if server.wakeupWorker(j.FuncName, it.Value.(*Worker)){
				i++
			}
			if server.tryTimes > 0 && i >= server.tryTimes {
				break
			}
		}
	}

}

func (sever *Server) checkAndRemoveJob(tp uint32, j *Job) {
	switch tp {
	case WORK_COMPLETE, WORK_EXCEPTION, WORK_FAIL:
		sever.removeJob(j)
	}
}

func (sever *Server) removeJob(j *Job) {
	delete(sever.workJobs, j.Handle)
}

func (server *Server) handleWorkReport(e *Event) {

	args := e.args
	slice := args.t0.([][]byte)
	jobhandle := bytes2str(slice[0])

	logger.Logger().T("%v job handle %v", CmdDescription(e.tp), jobhandle)

	j, ok := server.workJobs[jobhandle]
	if !ok {
		logger.Logger().W("job lost:%v  handle %v", CmdDescription(e.tp), jobhandle)
		return
	} 

	if j.Handle != jobhandle {
		logger.Logger().E("job handle not match")
	}

	server.checkAndRemoveJob(e.tp, j)

	if WORK_STATUS == e.tp {
		j.Percent, _ = strconv.Atoi(string(slice[1]))
		j.Denominator, _ = strconv.Atoi(string(slice[2]))
	}

	if j.IsBackGround {
		return
	}

	c, ok := server.client[j.CreateBy]
	if !ok {
		logger.Logger().W("sessionId missing %v %v", j.Handle, j.CreateBy)
		return
	}

	reply := constructReply(e.tp, slice)
	c.Send(reply)
}

func (server *Server) handleCloseSession(e *Event) {
	sessionId := e.fromSessionId
	if w, ok := server.worker[sessionId]; ok {
		if sessionId != w.SessionId {
			logger.Logger().E("sessionId not match %d-%d, bug found", sessionId, w.SessionId)
		}
		server.removeWorkerBySessionId(w.SessionId)
	} else if c, ok := server.client[sessionId]; ok {
		logger.Logger().T("removeClient sessionId %v", sessionId)
		delete(server.client, c.SessionId)
	}
	e.result <- true
}

func (server *Server) setClientId(clientId string, w *Worker) {
	logger.Logger().T("setClientId sid:%v cid:%v", w.SessionId, clientId)
	w.workerId = clientId
}

func (server *Server) handleCtrlEvt(e *Event) {

	switch e.tp {
	case ctrlCloseSession:
		server.handleCloseSession(e)
		return
	case getJobStatus:
		server.getJobStatus(e)
		return
	case getFuncWorkerStatus:
		server.getFuncWorkerStatus(e)
		return
	case getWorkerStatus:
		server.getWorkerStatus(e)
		return
	case getClientStatus:
		server.getClientStatus(e)
		return
	case removeJob:
		server.removeJobDirect(e)
		return
	default:
		logger.Logger().W("%s, %d", CmdDescription(e.tp), e.tp)
	}

	return
}

func (server *Server) handleProtoEvt(e *Event) {
	args := e.args

	if e.tp >= ctrlCloseSession {
		server.handleCtrlEvt(e)
		return
	}

	switch e.tp {
	case CAN_DO:
		w := args.t0.(*Worker)
		funcName := args.t1.(string)
		timeout := 0
		server.handleCanDo(funcName, w, timeout)
		server.addFuncJobStore(funcName)
		break
	case CAN_DO_TIMEOUT:
		w := args.t0.(*Worker)
		funcName := args.t1.(string)
		timeout, err := strconv.Atoi(args.t2.(string))
		if err != nil {
			timeout = 0
			logger.Logger().W("timeout conv error, funcName %v", funcName)
		}
		server.handleCanDo(funcName, w, timeout)
		server.addFuncJobStore(funcName)
		break
	case CANT_DO:
		sessionId := e.fromSessionId
		funcName := args.t0.(string)
		server.removeCanDo(funcName, sessionId)
		break
	case SET_CLIENT_ID:
		server.setClientId(args.t1.(string), args.t0.(*Worker))
		break
	case GRAB_JOB, GRAB_JOB_UNIQ:

		sessionId := e.fromSessionId
		w, ok := server.worker[sessionId]
		if !ok {
			logger.Logger().W("unregister worker, sessionId %d", sessionId)
			e.result <- nil
			break
		}

		w.status = wsRunning

		j := server.popJob(sessionId)
		if j != nil {
			j.ProcessAt = time.Now()
			j.ProcessBy = sessionId
			server.workJobs[j.Handle] = j
			e.result <- j
		} else { //no job
			w.status = wsPrepareForSleep
			e.result <- nil
		}

		break
	case PRE_SLEEP:
		sessionId := e.fromSessionId
		w, ok := server.worker[sessionId]
		if !ok {
			logger.Logger().W("unregister worker, sessionId %d", sessionId)
			w = args.t0.(*Worker)
			server.worker[w.SessionId] = w
			break
		}
		w.status = wsSleep
		logger.Logger().T("worker sessionId %v %v sleep", sessionId, w.workerId)
		//check if there are any jobs for this worker
		for k, v := range w.canDo {
			if v && server.wakeupWorker(k, w) {
				break
			}
		}
		break
	case SUBMIT_JOB, SUBMIT_JOB_LOW_BG, SUBMIT_JOB_LOW:
		server.handleSubmitJob(e)
		break
	case WORK_DATA, WORK_WARNING, WORK_STATUS, WORK_COMPLETE,
		WORK_FAIL, WORK_EXCEPTION:
		server.handleWorkReport(e)
		break
	case RESET_ABILITIES:
		break
	default:
		logger.Logger().W("not support command:%s, %d", CmdDescription(e.tp), e.tp)
	}
}
