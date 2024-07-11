package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func setupLogging() error {
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

func verifySignature(signature string, payload []byte, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func executeScript(projectPath string) error {
	scriptPath := filepath.Join(projectPath, "scripts", cdScriptName)

	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script does not exist: %s", scriptPath)
	}

	// Get the current file mode
	info, err := os.Stat(scriptPath)
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}

	// Check if the script is executable
	if info.Mode().Perm()&0111 == 0 {
		// Make the script executable
		if err := os.Chmod(scriptPath, info.Mode()|0111); err != nil {
			return fmt.Errorf("error making script executable: %v", err)
		}
		log.Printf("Made script executable: %s", scriptPath)
	}

	// Execute the script
	cmd := exec.Command("/bin/sh", scriptPath)
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %v\nOutput: %s", err, output)
	}

	log.Printf("Script executed successfully: %s", scriptPath)
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