package service

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	errServerAlreadyRunning = errors.New("server already running")
)

var (
	nilResponse = struct{}{}
	wallets     = make(map[string]*Wallet)
	walletsLock sync.Mutex
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

type Wallet struct {
	Address  string
	PubKey   *ecdsa.PublicKey
	KeysData []*keygen.LocalPartySaveData
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

	r.HandleFunc(pathCreateWallet, m.handleCreateWallet) //.Methods(http.MethodPost)
	r.HandleFunc(pathGetWallets, m.handleGetWallets).Methods(http.MethodGet)
	r.HandleFunc(pathGetSignature, m.handleGetSignature).Methods(http.MethodGet)

	r.Use(mux.CORSMethodMiddleware(r))
	loggedRouter := LoggingMiddlewareLogrus(m.log, r)
	return loggedRouter
}

func (m *WalletMPCService) handleRoot(w http.ResponseWriter, req *http.Request) {
	m.log.Logger.Info("handle root")
	m.respondOK(w, nilResponse)
}

func (m *WalletMPCService) handleCreateWallet(w http.ResponseWriter, req *http.Request) {
	m.log.Logger.Info("handle create wallet")
	m.log.Logger.Info("wallet creation in progress...")
	keysData, err := generateTSSKey()
	if err != nil {
		m.log.WithField("response", len(keysData)).WithError(err).Error("Couldn't create an tss mpc wallet")
		http.Error(w, "", http.StatusInternalServerError)
	}

	pubKey := *keysData[0].ECDSAPub.ToECDSAPubKey()
	addr := toAddress(pubKey)
	if addr != "" {
		newWallet := &Wallet{
			Address:  addr,
			PubKey:   &pubKey,
			KeysData: keysData,
		}
		go func() {
			walletsLock.Lock()
			wallets[addr] = newWallet
			walletsLock.Unlock()
		}()
		m.respondOK(w, json.NewEncoder(w).Encode(newWallet))
	} else {
		m.log.WithField("response address", addr).Error("Couldn't create an tss mpc wallet")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (m *WalletMPCService) handleGetWallets(w http.ResponseWriter, req *http.Request) {
	m.log.Logger.Info("handle get wallets")
	walletsLock.Lock()
	defer walletsLock.Unlock()

	var addresses []string
	fmt.Println(wallets)
	for addr := range wallets {
		addresses = append(addresses, addr)
	}
	m.respondOK(w, json.NewEncoder(w).Encode(addresses))
}

func (m *WalletMPCService) handleGetSignature(w http.ResponseWriter, req *http.Request) {
	m.log.Logger.Info("handle get signature")
	q := req.URL.Query()
	dataHex := q.Get("data")
	walletAddr := q.Get("wallet")
	m.log.Logger.Infof("data %s", dataHex)
	m.log.Logger.Infof("wallet %s", walletAddr)
	if dataHex == "" {
		http.Error(w, "missing data param", http.StatusBadRequest)
		return
	}
	if walletAddr == "" {
		http.Error(w, "missing wallet param", http.StatusBadRequest)
		return
	}
	dataBytes, err := hex.DecodeString(dataHex)
	if err != nil {
		http.Error(w, "invalid data hex", http.StatusBadRequest)
		return
	}

	wallet, ok := wallets[walletAddr]

	if !ok {
		http.Error(w, "wallet not found", http.StatusNotFound)
		return
	}

	sig, err := tssSign(wallet.KeysData, dataBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Signing failed: %v", err), http.StatusInternalServerError)
		return
	}

	r := sig.GetR()
	s := sig.GetS()

	signature := sig.GetSignature()
	data := ethereumMessageHash(dataBytes)
	ok = verifySignature(*wallet.PubKey, data, bytesToBigInt(r), bytesToBigInt(s))
	if !ok {
		http.Error(w, fmt.Sprintf("Signing failed on signature verification: %v", err), http.StatusInternalServerError)
		return
	}
	resp := map[string]string{"R": hex.EncodeToString(r), "S": hex.EncodeToString(s), "Signature": hex.EncodeToString(signature)}
	json.NewEncoder(w).Encode(resp)
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
