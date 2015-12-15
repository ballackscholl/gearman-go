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

	m.Get("/pprof", pprof.Index)
	m.Get("/pprof/cmdline", pprof.Cmdline)
	m.Get("/pprof/profile", pprof.Profile)
	m.Get("/pprof/symbol", pprof.Symbol)
	m.Get("/pprof/block", pprof.Handler("block").ServeHTTP)
	m.Get("/pprof/heap", pprof.Handler("heap").ServeHTTP)
	m.Get("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	m.Get("/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	m.Get("/status/func", func(params martini.Params) string {
		return s.GetFuncWorkerStatus()
	})
	m.Get("/status/worker", func(params martini.Params) string {
		return s.GetWorkerStatus()
	})
	m.Get("/status/client", func(params martini.Params) string {
		return s.GetClientStatus()
	})
	m.Get("/status/job", func(params martini.Params) string {
		return s.GetJobStatus()
	})
	logger.Logger().E("%v", http.ListenAndServe(addr, m))
}
