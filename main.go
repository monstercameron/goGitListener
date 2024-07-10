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
)

const (
	port         = "3002"
	configFile   = "config.json"
	cdScriptName = "cd.sh"
)

type Project struct {
	Secret string `json:"secret"`
	Path   string `json:"path"`
}

type Config map[string]Project

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(w, r, config)
	})

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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

func handleWebhook(w http.ResponseWriter, r *http.Request, config Config) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	projectName := r.URL.Query().Get("project")
	project, ok := config[projectName]
	if !ok {
		http.Error(w, "Invalid project", http.StatusBadRequest)
		return
	}

	if !verifySignature(r.Header.Get("X-Hub-Signature-256"), payload, project.Secret) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "push" {
		if err := executeScript(project.Path); err != nil {
			log.Printf("Error executing script: %v", err)
			http.Error(w, "Error executing script", http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully executed script for project: %s", projectName)
	}

	w.WriteHeader(http.StatusOK)
}

func verifySignature(signature string, payload []byte, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func executeScript(projectPath string) error {
	scriptPath := filepath.Join(projectPath, cdScriptName)
	cmd := exec.Command("/bin/sh", scriptPath)
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %v\nOutput: %s", err, output)
	}
	return nil
}