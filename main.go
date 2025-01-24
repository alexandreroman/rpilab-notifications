package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

const OK_STATUS_COLOR = "00FF00"
const ERROR_STATUS_COLOR = "FF0000"
const IN_PROGRESS_STATUS_COLOR = "FFA500"

var blink1ServerUrl string

func readinessProbe(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "READY")
}

func livenessProbe(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "UP")
}

type AlertEvent struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func handleAlert(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}
	var alertEvent AlertEvent
	if err := json.NewDecoder(req.Body).Decode(&alertEvent); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode Alert payload: %s", err.Error()), http.StatusBadRequest)
		return
	}
	log.Printf("Received Alert payload: %s", alertEvent)

	switch alertEvent.Reason {
	case "ReconciliationSucceeded":
		{
			triggerStatusReady()
		}
	case "ReconciliationFailed", "BuildFailed":
		{
			triggerStatusError()
		}
	case "Progressing":
		{
			triggerStatusInProgress()
		}
	}

	fmt.Fprintf(w, "OK")
}

func triggerStatusReady() {
	log.Println("Trigger status READY")
	callBlink1Server(OK_STATUS_COLOR, false)
}

func triggerStatusError() {
	log.Println("Trigger status ERROR")
	callBlink1Server(ERROR_STATUS_COLOR, true)
}

func triggerStatusInProgress() {
	log.Println("Trigger status IN_PROGRESS")
	callBlink1Server(IN_PROGRESS_STATUS_COLOR, true)
}

func callBlink1Server(color string, blink bool) error {
	base_url := blink1ServerUrl + "/set"
	params := url.Values{}
	params.Add("color", color)
	if blink {
		base_url = blink1ServerUrl + "/blink"
		params.Add("repeat", "10")
	} else {
		params.Add("delay", "2")
	}
	url := fmt.Sprintf("%s?%s", base_url, params.Encode())

	var resp *http.Response
	var err error
	if resp, err = http.Get(url); err != nil {
		log.Printf("Failed to call blink1-server: %s", err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

func main() {
	log.Println("Starting rpilab-notifications")

	portString := os.Getenv("PORT")
	var port int
	if portString == "" {
		port = 8080
	} else {
		var err error
		port, err = strconv.Atoi(portString)
		if err != nil {
			log.Fatalf("Failed to parse env variable PORT: %s", portString)
			return
		}
	}

	blink1ServerUrl = os.Getenv("BLINK1_SERVER_URL")
	if blink1ServerUrl == "" {
		blink1ServerUrl = "http://blink1-server.blink1-server.svc.cluster.local"
	}
	log.Printf("Using blink1 server URL: %s", blink1ServerUrl)

	http.HandleFunc("/alert", handleAlert)
	http.HandleFunc("/readyz", readinessProbe)
	http.HandleFunc("/livez", livenessProbe)

	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
