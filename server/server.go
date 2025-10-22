package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

const DEFAULT_MUX_TIMEOUTS_CONFIG = 10

type MuxServerConfig struct {
	wTimeout    time.Duration
	rTimeout    time.Duration
	idleTimeout time.Duration
}

type MuxServer struct {
	// Port is the port the server will listen on.
	Port       string
	Host       string
	handler    *mux.Router
	httpServer *http.Server
}

func NewHandler() *mux.Router {
	muxRouter := mux.NewRouter()
	return muxRouter
}

func NewMuxServerConfig(wTimeout time.Duration, rTimeout time.Duration, idleTimeout time.Duration) *MuxServerConfig {
	return &MuxServerConfig{
		wTimeout:    wTimeout,
		rTimeout:    rTimeout,
		idleTimeout: idleTimeout,
	}
}

func NewMuxServer(port string, host string, muxConfig *MuxServerConfig) *MuxServer {
	var config *MuxServerConfig = muxConfig

	if config == nil {
		defaultConfig := DEFAULT_MUX_TIMEOUTS_CONFIG * time.Second
		config = NewMuxServerConfig(defaultConfig, defaultConfig, defaultConfig)
	}

	handler := NewHandler()
	srv := &http.Server{
		Handler:      handler,
		Addr:         host + ":" + port,
		WriteTimeout: config.wTimeout,
		ReadTimeout:  config.rTimeout,
		IdleTimeout:  config.idleTimeout,
	}

	log.Printf("Running on %s:%s\n", host, port)

	return &MuxServer{
		Port:       port,
		Host:       host,
		handler:    handler,
		httpServer: srv,
	}
}

func (s *MuxServer) Run(onStop, afterStop func(), ctx context.Context) {

	go func(server *http.Server) {
		if err := server.ListenAndServe(); err != nil {
			log.Print("Server is shutting down ", err)
			onStop()
			return
		}
	}(s.httpServer)

	shutDownChan := make(chan os.Signal, 1)
	signal.Notify(shutDownChan, os.Interrupt)

	<-shutDownChan
	s.httpServer.Shutdown(ctx)
	afterStop()
}

func (s *MuxServer) HandlerFunc(path string, handler http.HandlerFunc) {
	s.handler.HandleFunc(path, handler)
}
