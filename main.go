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
	log.Println("Initializing application...")
	
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Fatalf("Config file %s does not exist", configFile)
	}
	log.Println("Config file found.")

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}
	log.Println("Log directory ensured.")

	// Check if log file exists, if not create it
	logPath := filepath.Join(logDir, logFile)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		file, err := os.Create(logPath)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}
		file.Close()
		log.Println("Log file created.")
	} else {
		log.Println("Log file already exists.")
	}

	// Validate config file
	if _, err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Config loaded successfully.")

	log.Println("Initialization completed successfully")
}

func main() {
	log.Println("Starting main function...")
	
	var err error
	config, err = loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	log.Println("Config loaded in main function.")

	// Set up logging
	if err := setupLogging(); err != nil {
		log.Fatalf("Error setting up logging: %v", err)
	}
	log.Println("Logging set up successfully.")

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request to /webhook endpoint.")
		if r.Method == http.MethodPost {
			handleWebhook(w, r)
		} else {
			log.Printf("Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request to /metrics endpoint.")
		if r.Method == http.MethodGet {
			handleMetrics(w, r)
		} else {
			log.Printf("Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling webhook...")
	
	projectName := r.URL.Query().Get("project")
	if projectName == "" {
		log.Println("Error: Project name is required")
		sendErrorResponse(w, "Project name is required", http.StatusBadRequest)
		return
	}
	log.Printf("Project name: %s", projectName)

	logEntry := LogEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		ProjectName: projectName,
		Headers:     make(map[string]string),
		Status:      "success",
	}

	// Log headers
	log.Println("Logging headers...")
	for name, values := range r.Header {
		if len(values) > 0 {
			logEntry.Headers[name] = values[0]
			log.Printf("Header: %s: %s", name, values[0])
		}
	}

	// Read the raw payload
	log.Println("Reading payload...")
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		logEntry.Status = fmt.Sprintf("error reading request body: %v", err)
		logRequest(logEntry)
		sendErrorResponse(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	log.Printf("Payload read, length: %d bytes", len(payload))

	// Check if the content is form-encoded
	contentType := r.Header.Get("Content-Type")
	log.Printf("Content-Type: %s", contentType)
	var jsonPayload []byte
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		log.Println("Parsing form-encoded data...")
		// Parse form data
		err = r.ParseForm()
		if err != nil {
			log.Printf("Error parsing form data: %v", err)
			logEntry.Status = fmt.Sprintf("error parsing form data: %v", err)
			logRequest(logEntry)
			sendErrorResponse(w, "Error parsing form data: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Extract the payload from the "payload" form field
		payloadStr, err := url.QueryUnescape(r.FormValue("payload"))
		if err != nil {
			log.Printf("Error unescaping payload: %v", err)
			logEntry.Status = fmt.Sprintf("error unescaping payload: %v", err)
			logRequest(logEntry)
			sendErrorResponse(w, "Error unescaping payload: "+err.Error(), http.StatusBadRequest)
			return
		}
		jsonPayload = []byte(payloadStr)
		log.Println("Form data parsed and unescaped successfully.")
	} else {
		// Assume it's JSON if not form-encoded
		log.Println("Assuming JSON payload...")
		jsonPayload = payload
	}

	// Parse the JSON payload
	log.Println("Parsing JSON payload...")
	var githubPayload map[string]interface{}
	if err := json.Unmarshal(jsonPayload, &githubPayload); err != nil {
		log.Printf("Error parsing payload: %v", err)
		logEntry.Status = fmt.Sprintf("error parsing payload: %v", err)
		logRequest(logEntry)
		sendErrorResponse(w, "Error parsing payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("JSON payload parsed successfully.")

	// Extract useful information
	log.Println("Extracting repository information...")
	repository, ok := githubPayload["repository"].(map[string]interface{})
	if ok {
		logEntry.Payload = map[string]interface{}{
			"repository_name": repository["name"],
			"full_name":       repository["full_name"],
			"default_branch":  repository["default_branch"],
			"pushed_at":       repository["pushed_at"],
		}
		log.Printf("Repository info: %v", logEntry.Payload)
	} else {
		log.Println("Warning: Could not extract repository information")
	}

	log.Printf("Checking if project %s exists in config...", projectName)
	project, ok := config[projectName]
	if !ok {
		log.Printf("Error: Invalid project %s", projectName)
		logEntry.Status = "invalid project"
		logRequest(logEntry)
		sendErrorResponse(w, "Invalid project", http.StatusBadRequest)
		return
	}
	log.Println("Project found in config.")

	log.Println("Verifying signature...")
	if !verifySignature(r.Header.Get("X-Hub-Signature-256"), jsonPayload, project.Secret) {
		log.Println("Error: Invalid signature")
		logEntry.Status = "invalid signature"
		logRequest(logEntry)
		sendErrorResponse(w, "Invalid signature", http.StatusUnauthorized)
		return
	}
	log.Println("Signature verified successfully.")

	event := r.Header.Get("X-GitHub-Event")
	log.Printf("GitHub event: %s", event)
	if event == "push" {
		log.Printf("Executing script for project: %s", projectName)
		if err := executeScript(project.Path); err != nil {
			log.Printf("Error executing script: %v", err)
			logEntry.Status = fmt.Sprintf("error executing script: %v", err)
			logRequest(logEntry)
			sendErrorResponse(w, "Error executing script: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully executed script for project: %s", projectName)
	} else {
		log.Printf("Ignoring non-push event: %s", event)
	}

	logRequest(logEntry)
	log.Println("Webhook handled successfully.")
	sendSuccessResponse(w, fmt.Sprintf("Webhook processed successfully for project: %s", projectName))

	// Check if restart is required
	if _, err := os.Stat("/tmp/restart_required"); err == nil {
		log.Println("Restart flag detected. Shutting down server...")
		os.Remove("/tmp/restart_required")
		go func() {
			time.Sleep(2 * time.Second)  // Give time for the response to be sent
			os.Exit(0)  // This will cause systemd to restart the service
		}()
	}
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	log.Printf("Sending error response: %s (Status: %d)", message, statusCode)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func sendSuccessResponse(w http.ResponseWriter, message string) {
	log.Printf("Sending success response: %s", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling metrics request...")
	logPath := filepath.Join(logDir, logFile)
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		log.Printf("Error reading log file: %v", err)
		http.Error(w, fmt.Sprintf("Error reading log file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(logContent)
	log.Println("Metrics request handled successfully.")
}

// Utility functions (loadConfig, setupLogging, verifySignature, executeScript, logRequest)
// should be implemented in utils.go