package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"gopkg.in/ini.v1"

	"gcloud-tts-rpc-websockets-go/pkg/logger"
	"gcloud-tts-rpc-websockets-go/pkg/tts"
	"gcloud-tts-rpc-websockets-go/pkg/websocket"
)

func main() {
	// Load configuration
	cfg, err := ini.Load("configs/config.ini")
	if err != nil {
		logger.Errorf("Failed to read config: %v", err)
		os.Exit(1)
	}

	// Parse configuration
	config := websocket.Config{
		BindAddress: cfg.Section("general").Key("bind").String(),
		Port:        cfg.Section("general").Key("port").MustInt(8086),
		CertFile:    cfg.Section("general").Key("cert").String(),
		KeyFile:     cfg.Section("general").Key("key").String(),
	}

	// Initialize Google Cloud credentials
	ctx := context.Background()
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		logger.Errorf("Failed to create token source: %v", err)
		os.Exit(1)
	}

	// Initialize TTS service
	ttsService, err := tts.NewTTSService(ctx, ts)
	if err != nil {
		logger.Errorf("Failed to create TTS service: %v", err)
		os.Exit(1)
	}

	// Create and start WebSocket server
	server := websocket.NewServer(config, ttsService)

	// Start server
	addr := fmt.Sprintf("%s:%d", config.BindAddress, config.Port)
	logger.Infof("Starting TTS WebSocket server on %s", addr)

	if config.CertFile != "" && config.KeyFile != "" {
		err = server.ListenAndServeTLS()
	} else {
		err = server.Listen()
	}

	if err != nil {
		logger.Errorf("Server error: %v", err)
		os.Exit(1)
	}
}
