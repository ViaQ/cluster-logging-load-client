package web

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var errNeedCertificateAndKey = errors.New("need both certificate and key to start TLS server")

type Server struct {
	cfg           ServerConfig
	log           logrus.FieldLogger
	listenAddress string
	server        *http.Server
}

func NewServer(cfg ServerConfig, log logrus.FieldLogger, registry prometheus.Gatherer) *Server {
	m := mux.NewRouter()
	m.Path("/metrics").Methods(http.MethodGet).Handler(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	s := &Server{
		cfg:           cfg,
		log:           log.WithField("component", "server"),
		listenAddress: cfg.ListenAddress,
		server: &http.Server{
			Addr:    cfg.ListenAddress,
			Handler: m,
		},
	}

	return s
}

func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error) {
	wg.Add(1)
	go func() {
		<-ctx.Done()
		s.log.Debug("Shutting down server...")
		if err := s.server.Shutdown(context.Background()); err != nil {
			errCh <- err
		}
	}()

	go func() {
		defer wg.Done()

		if s.cfg.TLS != nil {
			tls := s.cfg.TLS
			if tls.CertificateFile == "" || tls.KeyFile == "" {
				errCh <- errNeedCertificateAndKey
				return
			}

			s.log.Infof("Starting TLS server on %s", s.listenAddress)
			if err := s.server.ListenAndServeTLS(tls.CertificateFile, tls.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}

		s.log.Infof("Starting server on %s", s.listenAddress)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
}
