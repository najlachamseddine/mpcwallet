package service

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	errServerAlreadyRunning = errors.New("server already running")
)

var (
	nilResponse = struct{}{}
)

type WalletMPCService struct {
	listenAddr string
	log        *logrus.Entry
	srv        *http.Server
}

type ServerOpts struct {
	ListenAddr string
	Log        *logrus.Entry
}

func NewWalletMPCService(opts ServerOpts) (*WalletMPCService, error) {
	return &WalletMPCService{
		listenAddr: opts.ListenAddr,
		log:        opts.Log,
	}, nil
}

func (m *WalletMPCService) getRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", m.handleRoot)

	r.HandleFunc(pathCreateWallet, m.handleCreateWallet).Methods(http.MethodPost)
	// r.HandleFunc(pathGetWallets, w.handleGetWallets).Methods(http.MethodGet)
	// r.HandleFunc(pathGetSignature, w.handleGetSignature).Methods(http.MethodGet)

	r.Use(mux.CORSMethodMiddleware(r))
	loggedRouter := LoggingMiddlewareLogrus(m.log, r)
	return loggedRouter
}

func (m *WalletMPCService) handleRoot(w http.ResponseWriter, req *http.Request) {
	m.respondOK(w, nilResponse)
}

func (m *WalletMPCService) handleCreateWallet(w http.ResponseWriter, req *http.Request) {

}

func (m *WalletMPCService) respondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (w *WalletMPCService) StartHTTPServer() error {
	if w.srv != nil {
		return errServerAlreadyRunning
	}

	w.srv = &http.Server{
		Addr:    w.listenAddr,
		Handler: w.getRouter(),
	}

	err := w.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err

}
