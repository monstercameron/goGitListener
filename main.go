package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/metrics", handleMetrics)

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func setupLogging() error {
	// Ensure the logs directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("error creating logs directory: %v", err)
	}

	// Open the log file
	logPath := filepath.Join(logDir, logFile)
	logFileHandle, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening log file: %v", err)
	}

	// Set up the logger
	logger = log.New(logFileHandle, "", log.LstdFlags)
	log.Printf("Logging to %s", logPath)

	return nil
}

func loadConfig() (Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return config, nil
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectName := r.URL.Query().Get("project")
	logEntry := LogEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		ProjectName: projectName,
		Headers:     make(map[string]string),
		Status:      "success",
	}

	// Log headers
	for name, values := range r.Header {
		logEntry.Headers[name] = values[0]
	}

	// Read and log payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		logEntry.Status = fmt.Sprintf("error reading request body: %v", err)
		logRequest(logEntry)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Parse the payload
	var githubPayload map[string]interface{}
	if err := json.Unmarshal(payload, &githubPayload); err != nil {
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

	if !verifySignature(r.Header.Get("X-Hub-Signature-256"), payload, project.Secret) {
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

func verifySignature(signature string, payload []byte, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func executeScript(projectPath string) error {
	scriptPath := filepath.Join(projectPath, "scripts", cdScriptName)
	cmd := exec.Command("/bin/sh", scriptPath)
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %v\nOutput: %s", err, output)
	}
	return nil
}

func logRequest(entry LogEntry) {
	jsonEntry, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}
	logger.Println(string(jsonEntry))
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