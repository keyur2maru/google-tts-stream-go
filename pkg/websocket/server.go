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

	// Register connection
	s.mu.Lock()
	s.conns[conn.id] = conn
	s.mu.Unlock()

	// Ensure cleanup on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn.id)
		s.mu.Unlock()
		wsConn.Close()
		s.ttsService.CloseSession(conn.id)
	}()

	// Handle the first message to initialize the stream
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		logger.Errorf("Failed to read initial message: %v", err)
		return
	}

	var req TTSRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		s.sendError(conn, "Invalid initial request format")
		return
	}

	// Create TTS session
	ctx := context.Background()
	if err := s.ttsService.CreateSession(ctx, conn.id, req.Voice, req.Lang); err != nil {
		s.sendError(conn, fmt.Sprintf("Failed to create TTS session: %v", err))
		return
	}

	// Get audio channel
	audioChan, errChan, err := s.ttsService.GetAudioChannel(conn.id)
	if err != nil {
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
		logger.Errorf("Failed to send start message: %v", err)
		return
	}

	// Handle incoming messages
	for {
		_, msg, err := wsConn.ReadMessage()
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("Error reading message: %v", err)
			}
			break
		}

		var req TTSRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			s.sendError(conn, "Invalid request format")
			continue
		}

		// Send text to existing stream
		if err := s.ttsService.SynthesizeText(conn.id, req.Text); err != nil {
			s.sendError(conn, fmt.Sprintf("Failed to synthesize text: %v", err))
			continue
		}
	}
}

func (s *Server) streamAudio(conn *Connection, audioChan <-chan []byte, errChan <-chan error) {
	for {
		select {
		case audioData, ok := <-audioChan:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
				logger.Errorf("Failed to send audio data: %v", err)
				return
			}

		case err, ok := <-errChan:
			if !ok {
				return
			}
			s.sendError(conn, fmt.Sprintf("TTS error: %v", err))
			return
		}
	}
}

func (s *Server) sendError(conn *Connection, message string) {
	resp := TTSResponse{
		Type:  "error",
		Error: message,
	}
	if err := conn.WriteJSON(resp); err != nil {
		logger.Errorf("Failed to send error message: %v", err)
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
