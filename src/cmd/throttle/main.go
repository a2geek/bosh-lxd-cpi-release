package main

import (
	"bosh-lxd-cpi/config"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
	logLevelOpt   = flag.String("logLevel", "WARN", "Set log level (NONE, ERROR, WARN, INFO, DEBUG)")
)

func main() {
	flag.Parse()

	loglevel, err := boshlog.Levelify(*logLevelOpt)
	if err != nil {
		loglevel = boshlog.LevelError
	}

	logger := boshlog.NewLogger(loglevel)
	fs := boshsys.NewOsFileSystem(logger)

	config, err := config.NewConfigFromPath(*configPathOpt, fs)
	if err != nil {
		logger.Error("main", "Loading config %s", err.Error())
		os.Exit(1)
	}

	holdDuration, err := time.ParseDuration(config.Throttle.Hold)
	if err != nil {
		logger.Error("main", "Unable to parse hold duration: %s", err.Error())
		os.Exit(2)
	}

	if _, err := os.Stat(config.Throttle.Path); err == nil {
		os.Remove(config.Throttle.Path)
	}

	socket, err := net.Listen("unix", config.Throttle.Path)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()

	transactions := map[string]time.Time{}

	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				for k, v := range transactions {
					if time.Now().After(v.Add(holdDuration)) {
						logger.Warn("main", "Transaction %s expired", k)
						delete(transactions, k)
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	defer close(quit)

	uuidGen := boshuuid.NewGenerator()
	mux := http.NewServeMux()
	mux.HandleFunc("/transactions/{transactionId}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			transactionId := r.PathValue("transactionId")
			if transactionId == "" {
				logger.Debug("main", "%s %s - Bad request; transactionId=%s", r.Method, r.URL.Path, transactionId)
				w.WriteHeader(http.StatusBadRequest)
			} else if startTime, ok := transactions[transactionId]; !ok {
				logger.Debug("main", "%s %s - Not found; transactionId=%s", r.Method, r.URL.Path, transactionId)
				w.WriteHeader(http.StatusNotFound)
			} else {
				duration := time.Since(startTime)
				logger.Info("main", "%s %s - Transaction %s completed; duration=%s", r.Method, r.URL.Path, transactionId, duration.String())
				delete(transactions, transactionId)
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			logger.Debug("main", "%s %s - Method not allowed", r.Method, r.URL.Path)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/transactions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			data, err := json.Marshal(transactions)
			if err != nil {
				logger.Error("main", "%s %s - %v", r.Method, r.URL.Path, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				logger.Debug("main", "%s %s - List of active transactions", r.Method, r.URL.Path)
				w.Write(data)
			}
		case http.MethodPost:
			if len(transactions) > config.Throttle.Limit {
				logger.Debug("main", "%s %s - Too many requests", r.Method, r.URL.Path)
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
			} else {
				transactionId, err := uuidGen.Generate()
				if err != nil {
					logger.Error("main", "%s %s - %v", r.Method, r.URL.Path, err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				} else {
					logger.Info("main", "%s %s - Transaction %s created", r.Method, r.URL.Path, transactionId)
					transactions[transactionId] = time.Now()
					w.WriteHeader(http.StatusCreated)
					w.Write([]byte(transactionId))
				}
			}
		}
	})

	logger.Info("main", "Now serving traffic on socket %s", config.Throttle.Path)
	log.Fatal(http.Serve(socket, mux))
}
