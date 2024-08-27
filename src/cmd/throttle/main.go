package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

const SOCKET_FILE = "/tmp/throttle.sock"

func main() {
	socket, err := net.Listen("unix", SOCKET_FILE)
	if err != nil {
		log.Fatal(err)
	}
	// TODO: fix delete
	defer os.Remove(SOCKET_FILE)

	// TODO: Time
	transactions := map[string]time.Time{}
	uuidGen := boshuuid.NewGenerator()

	mux := http.NewServeMux()

	mux.HandleFunc("/transactions/{transactionId}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			transactionId := r.PathValue("transactionId")
			if transactionId == "" {
				w.WriteHeader(http.StatusBadRequest)
			} else if _, ok := transactions[transactionId]; !ok {
				w.WriteHeader(http.StatusNotFound)
			} else {
				delete(transactions, transactionId)
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/transactions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			data, err := json.Marshal(transactions)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				w.Write(data)
			}
		case http.MethodPost:
			if len(transactions) > 4 {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
			} else {
				uuid, err := uuidGen.Generate()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				} else {
					transactions[uuid] = time.Now() // TODO handle the duration
					w.Write([]byte(uuid))
					w.WriteHeader(http.StatusCreated)
				}
			}
		}
	})

	err = http.Serve(socket, mux)
	if err != nil {
		log.Fatal(err)
	}
}
