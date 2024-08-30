package throttle

import (
	"bosh-lxd-cpi/config"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

func NewThrottleServer(config config.ThrottleConfig, logger boshlog.Logger) (ThrottleServer, error) {
	holdDuration, err := time.ParseDuration(config.Hold)
	if err != nil {
		logger.Error("main", "Unable to parse hold duration: %s", err.Error())
		return ThrottleServer{}, err
	}
	if _, err := os.Stat(config.Path); err == nil {
		os.Remove(config.Path)
	}

	return ThrottleServer{
		logger:       logger,
		path:         config.Path,
		holdDuration: holdDuration,
		limit:        config.Limit,
		transactions: map[string]time.Time{},
		uuidGen:      boshuuid.NewGenerator(),
	}, nil
}

type ThrottleServer struct {
	logger       boshlog.Logger
	path         string
	holdDuration time.Duration
	limit        int
	transactions map[string]time.Time
	uuidGen      boshuuid.Generator
}

func (ts *ThrottleServer) Serve() error {
	socket, err := net.Listen("unix", ts.path)
	if err != nil {
		return err
	}
	defer socket.Close()

	go ts.expireTransactions()

	mux := http.NewServeMux()
	mux.HandleFunc("/transactions/{transactionId}", ts.handleTransactionWithId)
	mux.HandleFunc("/transactions", ts.handleTransactions)

	ts.logger.Info("main", "Now serving traffic on socket %s", ts.path)
	return http.Serve(socket, mux)
}

func (ts *ThrottleServer) expireTransactions() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		for k, v := range ts.transactions {
			if time.Now().After(v.Add(ts.holdDuration)) {
				ts.logger.Warn("main", "Transaction %s expired", k)
				delete(ts.transactions, k)
			}
		}
	}
}

func (ts *ThrottleServer) handleTransactionWithId(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		transactionId := r.PathValue("transactionId")
		if transactionId == "" {
			ts.logger.Debug("main", "%s %s - Bad request; transactionId=%s", r.Method, r.URL.Path, transactionId)
			w.WriteHeader(http.StatusBadRequest)
		} else if startTime, ok := ts.transactions[transactionId]; !ok {
			ts.logger.Debug("main", "%s %s - Not found; transactionId=%s", r.Method, r.URL.Path, transactionId)
			w.WriteHeader(http.StatusNotFound)
		} else {
			duration := time.Since(startTime)
			ts.logger.Info("main", "%s %s - Transaction %s completed; duration=%s", r.Method, r.URL.Path, transactionId, duration.String())
			delete(ts.transactions, transactionId)
			w.WriteHeader(http.StatusNoContent)
		}
	default:
		ts.logger.Debug("main", "%s %s - Method not allowed", r.Method, r.URL.Path)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ts *ThrottleServer) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data, err := json.Marshal(ts.transactions)
		if err != nil {
			ts.logger.Error("main", "%s %s - %v", r.Method, r.URL.Path, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			ts.logger.Debug("main", "%s %s - List of active transactions", r.Method, r.URL.Path)
			w.Write(data)
		}
	case http.MethodPost:
		if len(ts.transactions) > ts.limit {
			ts.logger.Debug("main", "%s %s - Too many requests", r.Method, r.URL.Path)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
		} else {
			transactionId, err := ts.uuidGen.Generate()
			if err != nil {
				ts.logger.Error("main", "%s %s - %v", r.Method, r.URL.Path, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				ts.logger.Info("main", "%s %s - Transaction %s created", r.Method, r.URL.Path, transactionId)
				ts.transactions[transactionId] = time.Now()
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(transactionId))
			}
		}
	}
}
