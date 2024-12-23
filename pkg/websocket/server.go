package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"gcloud-tts-rpc-websockets-go/pkg/logger"
	"gcloud-tts-rpc-websockets-go/pkg/tts"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Config holds server configuration
type Config struct {
	BindAddress string
	Port        int
	CertFile    string
	KeyFile     string
}

// TTSRequest represents the incoming TTS request
type TTSRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
	Lang  string `json:"lang"`
}

// TTSResponse represents a control message
type TTSResponse struct {
	Type   string `json:"type"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for testing
	},
}

// Connection represents a WebSocket connection with synchronized write operations
type Connection struct {
	id      string
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// WriteJSON safely writes a JSON message to the WebSocket
func (c *Connection) WriteJSON(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(v)
}

// WriteMessage safely writes a message to the WebSocket
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(messageType, data)
}

// Server represents our WebSocket TTS server
type Server struct {
	config     Config
	ttsService *tts.TTSService
	mu         sync.Mutex
	conns      map[string]*Connection
}

// NewServer creates a new WebSocket server instance
func NewServer(config Config, ttsService *tts.TTSService) *Server {
	return &Server{
		config:     config,
		ttsService: ttsService,
		conns:      make(map[string]*Connection),
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("Failed to upgrade connection: %v", err)
		return
	}

	// Create connection wrapper with unique ID
	conn := &Connection{
		id:   uuid.New().String(),
		conn: wsConn,
	}

	logger.Debugf("[WS][%s] New WebSocket connection established", conn.id)

	// Register connection
	s.mu.Lock()
	s.conns[conn.id] = conn
	s.mu.Unlock()

	logger.Debugf("[WS][%s] Connection registered", conn.id)

	// Ensure cleanup on disconnect
	defer func() {
		logger.Debugf("[WS][%s] Connection closing, cleaning up", conn.id)
		s.mu.Lock()
		delete(s.conns, conn.id)
		s.mu.Unlock()
		wsConn.Close()
		s.ttsService.CloseSession(conn.id)
		logger.Debugf("[WS][%s] Cleanup completed", conn.id)
	}()

	// Handle the first message to initialize the stream
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		logger.Errorf("[WS][%s] Failed to read initial message: %v", conn.id, err)
		return
	}

	logger.Debugf("[WS][%s] Received initial message: %s", conn.id, string(msg))

	var req TTSRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		logger.Errorf("[WS][%s] Failed to parse initial message: %v", conn.id, err)
		s.sendError(conn, "Invalid initial request format")
		return
	}

	// Create TTS session
	ctx := context.Background()
	logger.Debugf("[WS][%s] Creating TTS session with voice=%s, lang=%s",
		conn.id, req.Voice, req.Lang)

	if err := s.ttsService.CreateSession(ctx, conn.id, req.Voice, req.Lang); err != nil {
		logger.Errorf("[WS][%s] Failed to create TTS session: %v", conn.id, err)
		s.sendError(conn, fmt.Sprintf("Failed to create TTS session: %v", err))
		return
	}

	if err := s.ttsService.SynthesizeText(conn.id, req.Text); err != nil {
		logger.Errorf("[WS][%s] Failed to synthesize initial text: %v", conn.id, err)
		s.sendError(conn, fmt.Sprintf("Failed to synthesize text: %v", err))
		return
	}

	// Get audio channel
	audioChan, errChan, err := s.ttsService.GetAudioChannel(conn.id)
	if err != nil {
		logger.Errorf("[WS][%s] Failed to get audio channel: %v", conn.id, err)
		s.sendError(conn, fmt.Sprintf("Failed to get audio channel: %v", err))
		return
	}

	// Start audio streaming goroutine
	go s.streamAudio(conn, audioChan, errChan)

	// Send success response
	if err := conn.WriteJSON(TTSResponse{
		Type:   "start",
		Status: "ready",
	}); err != nil {
		logger.Errorf("[WS][%s] Failed to send start message: %v", conn.id, err)
		return
	}

	logger.Debugf("[WS][%s] Initial setup complete, entering message loop", conn.id)

	// Handle incoming messages
	for {
		_, msg, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Debugf("[WS][%s] WebSocket closed normally", conn.id)
			} else {
				logger.Errorf("[WS][%s] Error reading message: %v", conn.id, err)
			}
			break
		}

		logger.Debugf("[WS][%s] Received message: %s", conn.id, string(msg))

		var req TTSRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			logger.Errorf("[WS][%s] Failed to parse message: %v", conn.id, err)
			s.sendError(conn, "Invalid request format")
			continue
		}

		// Send text to existing stream
		logger.Debugf("[WS][%s] Synthesizing text: %q", conn.id, req.Text)
		if err := s.ttsService.SynthesizeText(conn.id, req.Text); err != nil {
			logger.Errorf("[WS][%s] Failed to synthesize text: %v", conn.id, err)
			s.sendError(conn, fmt.Sprintf("Failed to synthesize text: %v", err))
			continue
		}
	}
}

func (s *Server) streamAudio(conn *Connection, audioChan <-chan []byte, errChan <-chan error) {
	logger.Debugf("[WS][%s] Starting audio streaming", conn.id)
	audioChunks := 0
	totalBytes := 0

	for {
		select {
		case audioData, ok := <-audioChan:
			if !ok {
				logger.Debugf("[WS][%s] Audio channel closed. Total chunks: %d, Total bytes: %d",
					conn.id, audioChunks, totalBytes)
				return
			}
			audioChunks++
			totalBytes += len(audioData)
			logger.Debugf("[WS][%s] Sending audio chunk %d: %d bytes",
				conn.id, audioChunks, len(audioData))

			if err := conn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
				logger.Errorf("[WS][%s] Failed to send audio data: %v", conn.id, err)
				return
			}

		case err, ok := <-errChan:
			if !ok {
				logger.Debugf("[WS][%s] Error channel closed", conn.id)
				return
			}
			logger.Errorf("[WS][%s] TTS error: %v", conn.id, err)
			s.sendError(conn, fmt.Sprintf("TTS error: %v", err))
			return
		}
	}
}

func (s *Server) sendError(conn *Connection, message string) {
	logger.Errorf("[WS][%s] Sending error: %s", conn.id, message)
	resp := TTSResponse{
		Type:  "error",
		Error: message,
	}
	if err := conn.WriteJSON(resp); err != nil {
		logger.Errorf("[WS][%s] Failed to send error message: %v", conn.id, err)
	}
}

// Listen starts the server without TLS
func (s *Server) Listen() error {
	http.HandleFunc("/ws", s.handleWebSocket)
	return http.ListenAndServe(
		fmt.Sprintf("%s:%d", s.config.BindAddress, s.config.Port),
		nil,
	)
}

// ListenAndServeTLS starts the server with TLS
func (s *Server) ListenAndServeTLS() error {
	http.HandleFunc("/ws", s.handleWebSocket)
	return http.ListenAndServeTLS(
		fmt.Sprintf("%s:%d", s.config.BindAddress, s.config.Port),
		s.config.CertFile,
		s.config.KeyFile,
		nil,
	)
}
