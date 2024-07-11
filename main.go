package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	port         = "3002"
	configFile   = "config.json"
	cdScriptName = "cd.sh"
	logDir       = "logs"
	logFile      = "log.log"
)

type Project struct {
	Secret string `json:"secret"`
	Path   string `json:"path"`
}

type Config map[string]Project

type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	ProjectName string                 `json:"project_name"`
	Headers     map[string]string      `json:"headers"`
	Payload     map[string]interface{} `json:"payload"`
	Status      string                 `json:"status"`
}

var (
	config Config
	logger *log.Logger
)

func init() {
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Fatalf("Config file %s does not exist", configFile)
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// Check if log file exists, if not create it
	logPath := filepath.Join(logDir, logFile)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		file, err := os.Create(logPath)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}
		file.Close()
	}

	// Validate config file
	if _, err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Initialization completed successfully")
}

func main() {
	var err error
	config, err = loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Set up logging
	if err := setupLogging(); err != nil {
		log.Fatalf("Error setting up logging: %v", err)
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleWebhook(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleMetrics(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Query().Get("project")
	if projectName == "" {
		http.Error(w, "Project name is required", http.StatusBadRequest)
		return
	}

	logEntry := LogEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		ProjectName: projectName,
		Headers:     make(map[string]string),
		Status:      "success",
	}

	// Log headers
	for name, values := range r.Header {
		if len(values) > 0 {
			logEntry.Headers[name] = values[0]
		}
	}

	// Read the raw payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		logEntry.Status = fmt.Sprintf("error reading request body: %v", err)
		logRequest(logEntry)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Check if the content is form-encoded
	contentType := r.Header.Get("Content-Type")
	var jsonPayload []byte
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Parse form data
		err = r.ParseForm()
		if err != nil {
			logEntry.Status = fmt.Sprintf("error parsing form data: %v", err)
			logRequest(logEntry)
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}
		// Extract the payload from the "payload" form field
		payloadStr, err := url.QueryUnescape(r.FormValue("payload"))
		if err != nil {
			logEntry.Status = fmt.Sprintf("error unescaping payload: %v", err)
			logRequest(logEntry)
			http.Error(w, "Error unescaping payload", http.StatusBadRequest)
			return
		}
		jsonPayload = []byte(payloadStr)
	} else {
		// Assume it's JSON if not form-encoded
		jsonPayload = payload
	}

	// Parse the JSON payload
	var githubPayload map[string]interface{}
	if err := json.Unmarshal(jsonPayload, &githubPayload); err != nil {
		logEntry.Status = fmt.Sprintf("error parsing payload: %v", err)
		logRequest(logEntry)
		http.Error(w, "Error parsing payload", http.StatusBadRequest)
		return
	}

	// Extract useful information
	repository, ok := githubPayload["repository"].(map[string]interface{})
	if ok {
		logEntry.Payload = map[string]interface{}{
			"repository_name": repository["name"],
			"full_name":       repository["full_name"],
			"default_branch":  repository["default_branch"],
			"pushed_at":       repository["pushed_at"],
		}
	}

	project, ok := config[projectName]
	if !ok {
		logEntry.Status = "invalid project"
		logRequest(logEntry)
		http.Error(w, "Invalid project", http.StatusBadRequest)
		return
	}

	if !verifySignature(r.Header.Get("X-Hub-Signature-256"), jsonPayload, project.Secret) {
		logEntry.Status = "invalid signature"
		logRequest(logEntry)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "push" {
		if err := executeScript(project.Path); err != nil {
			logEntry.Status = fmt.Sprintf("error executing script: %v", err)
			logRequest(logEntry)
			http.Error(w, "Error executing script", http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully executed script for project: %s", projectName)
	}

	logRequest(logEntry)
	w.WriteHeader(http.StatusOK)
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	logPath := filepath.Join(logDir, logFile)
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading log file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(logContent)
}