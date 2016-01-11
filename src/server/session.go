package server

import (
	"bufio"
	. "common"
	"net"
	"time"
	//"runtime"
	"utils/logger"
)

type Session struct {
	sessionId int64
	w         *Worker
	c         *Client
}

func (session *Session) getWorker(sessionId int64, inbox chan []byte, conn net.Conn) *Worker {
	if session.w != nil {
		return session.w
	}

	session.w = &Worker{
		Conn: conn, status: wsSleep, Connector: Connector{SessionId: sessionId,
			in: inbox, ConnectAt: time.Now()}, canDo: make(map[string]bool)}

	return session.w
}

func (session *Session) handleConnection(server *Server, conn net.Conn) {

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetLinger(-1)
	conn.(*net.TCPConn).SetReadBuffer(20 * 1024)
	conn.(*net.TCPConn).SetWriteBuffer(20 * 1024)
	conn.(*net.TCPConn).SetKeepAlive(true)
	conn.(*net.TCPConn).SetKeepAlivePeriod(2 * time.Minute)

	sessionId := server.allocSessionId()
	inbox := make(chan []byte, 2048)

	defer func() {
		if session.w != nil || session.c != nil {
			e := &Event{tp: ctrlCloseSession, fromSessionId: sessionId,
				result: createResCh()}
			server.protoEvtCh <- e
			<-e.result
			close(inbox) //notify writer to quit
		}
	}()

	go writer(conn, inbox)
	r := bufio.NewReaderSize(conn, 20*1024)

	for {
		tp, buf, err := ReadMessage(r)
		if err != nil {
			logger.Logger().W("sessionId: %v %v", sessionId, err)
			return
		}
		args, ok := decodeArgs(tp, buf)
		if !ok {
			logger.Logger().W("tp:%v argc not match details:%v", CmdDescription(tp), string(buf))
			return
		}

		logger.Logger().T("sessionId:%v tp:%v", sessionId, CmdDescription(tp))

		switch tp {
		case CAN_DO:
			session.w = session.getWorker(sessionId, inbox, conn)
			server.protoEvtCh <- &Event{tp: tp, args: &Tuple{
				t0: session.w, t1: string(args[0])}}
			break
		case CAN_DO_TIMEOUT:
			session.w = session.getWorker(sessionId, inbox, conn)
			server.protoEvtCh <- &Event{tp: tp, args: &Tuple{
				t0: session.w, t1: string(args[0]), t2: string(args[1])}}
			break
		case CANT_DO:
			server.protoEvtCh <- &Event{tp: tp, fromSessionId: sessionId,
				args: &Tuple{t0: string(args[0])}}
			break
		case ECHO_REQ:
			sendReply(inbox, ECHO_RES, [][]byte{buf})
			break
		case PRE_SLEEP:
			session.w = session.getWorker(sessionId, inbox, conn)
			server.protoEvtCh <- &Event{tp: tp, args: &Tuple{t0: session.w}, fromSessionId: sessionId}
			break
		case SET_CLIENT_ID:
			session.w = session.getWorker(sessionId, inbox, conn)
			server.protoEvtCh <- &Event{tp: tp, args: &Tuple{t0: session.w, t1: string(args[0])}}
			break
		case GRAB_JOB, GRAB_JOB_UNIQ:
			if session.w == nil {
				logger.Logger().W("can't perform %s, need send CAN_DO first", CmdDescription(tp))
				return
			}
			e := &Event{tp: tp, fromSessionId: sessionId,
				result: createResCh()}
			server.protoEvtCh <- e
			job := <-e.result
			if job == nil {
				logger.Logger().T("sessionId:%v no job", sessionId)
				sendReplyResult(inbox, nojobReply)
				break
			}
			logger.Logger().T("grap %v %v", sessionId, job.(*Job))
			if tp == GRAB_JOB {
				sendReply(inbox, JOB_ASSIGN, [][]byte{
					[]byte(job.(*Job).Handle),
					[]byte(job.(*Job).FuncName),
					job.(*Job).Data})
			} else {
				sendReply(inbox, JOB_ASSIGN_UNIQ, [][]byte{
					[]byte(job.(*Job).Handle),
					[]byte(job.(*Job).FuncName),
					[]byte(job.(*Job).Id),
					job.(*Job).Data})
			}
			break
		case SUBMIT_JOB, SUBMIT_JOB_LOW_BG, SUBMIT_JOB_LOW:
			if session.c == nil {
				session.c = &Client{Conn: conn, Connector: Connector{SessionId: sessionId, in: inbox,
					ConnectAt: time.Now()}}
			}
			e := &Event{tp: tp,
				args:   &Tuple{t0: session.c, t1: args[0], t2: args[1], t3: args[2]},
				//result: createResCh(),
			}

			server.protoEvtCh <- e
			//handle := <-e.result
			//sendReply(inbox, JOB_CREATED, [][]byte{[]byte(handle.(string)), args[1]})
			break
		case WORK_DATA, WORK_WARNING, WORK_COMPLETE,
			WORK_FAIL, WORK_EXCEPTION, WORK_STATUS:
			if session.w == nil {
				logger.Logger().W("can't perform %s, need send CAN_DO first", CmdDescription(tp))
				return
			}
			server.protoEvtCh <- &Event{tp: tp, args: &Tuple{t0: args},
				fromSessionId: sessionId}
			break
		default:
			logger.Logger().W("not support type %s", CmdDescription(tp))
		}
		//runtime.Gosched()
	}
}
