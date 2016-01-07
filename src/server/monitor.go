package server

import (
	"github.com/go-martini/martini"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"
	//"os"
	"utils/logger"
)

func registerWebHandler(s *Server, addr string) {

	if addr == "" {
		addr = ":1374"
	} else if addr == "-" {
		// Don't start web monitor
		return
	}

	m := martini.Classic()

	m.Get("/", pprof.Index)
	m.Get("/cmdline", pprof.Cmdline)
	m.Get("/profile", pprof.Profile)
	m.Get("/symbol", pprof.Symbol)
	m.Get("/block", pprof.Handler("block").ServeHTTP)
	m.Get("/heap", pprof.Handler("heap").ServeHTTP)
	m.Get("/goroutine", pprof.Handler("goroutine").ServeHTTP)
	m.Get("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	m.Get("/status/func", func(params martini.Params) string {
		e := &Event{tp: getJobStatus, result: createResCh()}
		s.protoEvtCh <- e
		return (<-e.result).(string)
	})
	m.Get("/status/worker", func(params martini.Params) string {
		e := &Event{tp: getFuncWorkerStatus, result: createResCh()}
		s.protoEvtCh <- e
		return (<-e.result).(string)
	})
	m.Get("/status/client", func(params martini.Params) string {
		e := &Event{tp: getWorkerStatus, result: createResCh()}
		s.protoEvtCh <- e
		return (<-e.result).(string)
	})
	m.Get("/status/job", func(params martini.Params) string {
		e := &Event{tp: getClientStatus, result: createResCh()}
		s.protoEvtCh <- e
		return (<-e.result).(string)
	})
	logger.Logger().E("%v", http.ListenAndServe(addr, m))
}
