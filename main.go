package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/cihub/seelog"

	"github.com/anchnet/graph/api"
	"github.com/anchnet/graph/g"
	"github.com/anchnet/graph/http"
	"github.com/anchnet/graph/index"
	"github.com/anchnet/graph/rrdtool"
)

func start_signal(pid int, cfg *g.GlobalConfig) {
	sigs := make(chan os.Signal, 1)
	log.Info(pid, "register signal notify")
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		s := <-sigs
		log.Info("recv", s)

		switch s {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			log.Info("gracefull shut down")
			if cfg.Http.Enabled {
				http.Close_chan <- 1
				<-http.Close_done_chan
			}
			log.Info("http stop ok")

			if cfg.Rpc.Enabled {
				api.Close_chan <- 1
				<-api.Close_done_chan
			}
			log.Info("rpc stop ok")

			rrdtool.Out_done_chan <- 1
			rrdtool.FlushAll(true)
			log.Info("rrdtool stop ok")

			log.Info(pid, "exit")
			os.Exit(0)
		}
	}
}

func main() {
	cfg := flag.String("c", "cfg.json", "specify config file")
	version := flag.Bool("v", false, "show version")
	versionGit := flag.Bool("vg", false, "show version and git commit log")
	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}
	if *versionGit {
		fmt.Println(g.VERSION, g.COMMIT)
		os.Exit(0)
	}

	// global config
	g.ParseConfig(*cfg)
	//init seelog
	g.InitSeeLog()
	// init db
	g.InitDB()
	// rrdtool before api for disable loopback connection
	rrdtool.Start()
	// start api
	go api.Start()
	// start indexing
	index.Start()
	// start http server
	go http.Start()

	start_signal(os.Getpid(), g.Config())
}
