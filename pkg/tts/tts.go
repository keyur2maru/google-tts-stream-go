package tts

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	pb "gcloud-tts-rpc-websockets-go/google/cloud/texttospeech/v1beta1"

	"gcloud-tts-rpc-websockets-go/pkg/logger"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

// TTSSession represents an active TTS streaming session
type TTSSession struct {
	stream        pb.TextToSpeech_StreamingSynthesizeClient
	streamLock    sync.Mutex
	config        *pb.StreamingSynthesizeConfig
	audioChan     chan []byte
	errorChan     chan error
	ctx           context.Context
	cancel        context.CancelFunc
	lastInputTime time.Time
	client        pb.TextToSpeechClient
	resetChan     chan struct{}
	needsReset    bool
}

// TTSService handles text-to-speech conversion using Google Cloud TTS
type TTSService struct {
	client       pb.TextToSpeechClient
	sampleRate   int32
	sessions     map[string]*TTSSession // Key is WebSocket connection ID
	sessionsLock sync.RWMutex
}

func (s *TTSService) resetStream(session *TTSSession, connID string) error {
	session.streamLock.Lock()
	defer session.streamLock.Unlock()

	// Just mark that we need a reset, don't actually do it yet
	session.needsReset = true
	logger.Debugf("[TTS][%s] Stream marked for reset on next input", connID)
	return nil
}

// NewTTSService creates a new TTS service instance
func NewTTSService(ctx context.Context, ts oauth2.TokenSource) (*TTSService, error) {
	log.Println("Initializing TTS Service...")

	// Create gRPC dial options with TLS and OAuth2
	creds := credentials.NewClientTLSFromCert(nil, "")
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: ts}),
	}

	// Connect to the service
	conn, err := grpc.DialContext(ctx, "texttospeech.googleapis.com:443", opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %v", err)
	}

	return &TTSService{
		client:     pb.NewTextToSpeechClient(conn),
		sampleRate: 24000,
		sessions:   make(map[string]*TTSSession),
	}, nil
}

// CreateSession initializes a new streaming session for a client
func (s *TTSService) CreateSession(ctx context.Context, connID string, voice string, lang string) error {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	logger.Debugf("[TTS][%s] Creating new session with voice=%s, lang=%s", connID, voice, lang)

	// Check if session already exists
	if _, exists := s.sessions[connID]; exists {
		logger.Debugf("[TTS][%s] Session already exists", connID)
		return fmt.Errorf("session already exists for connection: %s", connID)
	}

	// Initialize streaming client
	stream, err := s.client.StreamingSynthesize(ctx)
	if err != nil {
		logger.Errorf("[TTS][%s] Failed to create streaming client: %v", connID, err)
		return fmt.Errorf("failed to create streaming client: %v", err)
	}

	// Create session configuration
	config := &pb.StreamingSynthesizeConfig{
		Voice: &pb.VoiceSelectionParams{
			LanguageCode: lang,
			Name:         voice,
		},
	}

	// Create new session
	session := &TTSSession{
		stream:        stream,
		config:        config,
		audioChan:     make(chan []byte, 100),
		errorChan:     make(chan error, 1),
		ctx:           ctx,
		lastInputTime: time.Now(),
		client:        s.client,
		resetChan:     make(chan struct{}),
	}

	// Send initial configuration
	err = stream.Send(&pb.StreamingSynthesizeRequest{
		StreamingRequest: &pb.StreamingSynthesizeRequest_StreamingConfig{
			StreamingConfig: config,
		},
	})
	if err != nil {
		logger.Errorf("[TTS][%s] Failed to send initial config: %v", connID, err)
		return fmt.Errorf("failed to send config: %v", err)
	}

	// Store session and start goroutines
	s.sessions[connID] = session
	go s.receiveAudio(session, connID)
	go s.monitorSession(session, connID)

	logger.Debugf("[TTS][%s] Session created and initialized", connID)
	return nil
}

func (s *TTSService) monitorSession(session *TTSSession, connID string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-session.ctx.Done():
			return
		case <-ticker.C:
			if time.Since(session.lastInputTime) > 4*time.Second && !session.needsReset {
				logger.Debugf("[TTS][%s] Input timeout approaching, marking stream for reset", connID)
				if err := s.resetStream(session, connID); err != nil {
					logger.Errorf("[TTS][%s] Failed to mark stream for reset: %v", connID, err)
				}
			}
		}
	}
}

func (s *TTSService) SynthesizeText(connID string, text string) error {
	s.sessionsLock.RLock()
	session, exists := s.sessions[connID]
	s.sessionsLock.RUnlock()

	if !exists {
		logger.Errorf("[TTS][%s] No session found for synthesis", connID)
		return fmt.Errorf("no session found for connection: %s", connID)
	}

	session.streamLock.Lock()
	defer session.streamLock.Unlock()

	// Check if stream needs initialization or reset
	if session.needsReset || session.stream == nil {
		// Create new stream
		logger.Debugf("[TTS][%s] Creating new stream for synthesis", connID)
		newStream, err := session.client.StreamingSynthesize(session.ctx)
		if err != nil {
			return fmt.Errorf("failed to create new stream: %v", err)
		}

		// Close old stream and signal receive loop to stop if they exist
		if session.stream != nil {
			session.stream.CloseSend()
		}
		if session.resetChan != nil {
			close(session.resetChan)
			// Short wait to ensure receive loop has stopped
			time.Sleep(10 * time.Millisecond)
		}
		session.resetChan = make(chan struct{})

		// Send initial configuration first
		logger.Debugf("[TTS][%s] Sending initial config to new stream", connID)
		err = newStream.Send(&pb.StreamingSynthesizeRequest{
			StreamingRequest: &pb.StreamingSynthesizeRequest_StreamingConfig{
				StreamingConfig: session.config,
			},
		})
		if err != nil {
			newStream.CloseSend()
			return fmt.Errorf("failed to send config: %v", err)
		}

		// Update session with new stream
		session.stream = newStream
		session.needsReset = false

		// Start new receive loop
		go s.receiveAudio(session, connID)

		// Small delay to ensure config is processed
		time.Sleep(10 * time.Millisecond)
		logger.Debugf("[TTS][%s] New stream initialized and ready", connID)
	}

	// Now send the text
	logger.Debugf("[TTS][%s] Sending text to stream: %q", connID, text)
	err := session.stream.Send(&pb.StreamingSynthesizeRequest{
		StreamingRequest: &pb.StreamingSynthesizeRequest_Input{
			Input: &pb.StreamingSynthesisInput{
				InputSource: &pb.StreamingSynthesisInput_Text{
					Text: text,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send text: %v", err)
	}

	session.lastInputTime = time.Now()
	return nil
}

// receiveAudio handles receiving audio data from the gRPC stream
func (s *TTSService) receiveAudio(session *TTSSession, connID string) {
	var (
		audioChunks   int64 = 0
		totalBytes    int64 = 0
		startTime           = time.Now()
		lastChunkTime       = startTime
	)

	logger.Debugf("[TTS][%s][RECV] Starting audio receive loop at %v", connID, startTime)

	defer func() {
		duration := time.Since(startTime)
		if audioChunks > 0 {
			avgChunkSize := float64(totalBytes) / float64(audioChunks)
			avgChunkInterval := duration.Seconds() / float64(audioChunks)

			logger.Debugf("[TTS][%s][RECV] Audio receive completed:", connID)
			logger.Debugf("  - Duration: %.2f seconds", duration.Seconds())
			logger.Debugf("  - Total chunks: %d", audioChunks)
			logger.Debugf("  - Total bytes: %d", totalBytes)
			logger.Debugf("  - Avg chunk size: %.2f bytes", avgChunkSize)
			logger.Debugf("  - Avg chunk interval: %.2f ms", avgChunkInterval*1000)
		} else {
			logger.Debugf("[TTS][%s][RECV] No audio chunks received in %.2f seconds",
				connID, duration.Seconds())
		}
	}()

	for {
		select {
		case <-session.resetChan:
			return // Exit on reset signal
		case <-session.ctx.Done():
			return
		default:
			before := time.Now()
			resp, err := session.stream.Recv()
			if err != nil {
				logger.Errorf("[TTS][%s][RECV] Error receiving audio: %v", connID, err)
				session.errorChan <- fmt.Errorf("failed to receive: %v", err)

				// Add this section: Handle stream abort
				if err.Error() == "rpc error: code = Aborted desc = Stream aborted due to long duration elapsed without input sent" {
					// Mark session for reset, but don't reset immediately
					session.needsReset = true
					// Clear any pending audio data
					select {
					case <-session.audioChan:
					default:
					}
				}
				return
			}
			recvDuration := time.Since(before)

			if err == io.EOF {
				logger.Debugf("[TTS][%s][RECV] Stream completed naturally after %d chunks",
					connID, audioChunks)
				return
			}
			if err != nil {
				logger.Errorf("[TTS][%s][RECV] Error receiving audio after %.2fms: %v",
					connID, float64(recvDuration.Microseconds())/1000, err)
				session.errorChan <- fmt.Errorf("failed to receive: %v", err)
				return
			}

			audioChunks++
			totalBytes += int64(len(resp.AudioContent))
			chunkInterval := time.Since(lastChunkTime)
			lastChunkTime = time.Now()

			logger.Debugf("[TTS][%s][RECV] Chunk-%d received in %.2fms: %d bytes (%.2fms since last)",
				connID, audioChunks, float64(recvDuration.Microseconds())/1000,
				len(resp.AudioContent), float64(chunkInterval.Microseconds())/1000)

			select {
			case <-session.ctx.Done():
				logger.Debugf("[TTS][%s][RECV] Context cancelled after %d chunks",
					connID, audioChunks)
				return
			case session.audioChan <- resp.AudioContent:
				logger.Debugf("[TTS][%s][RECV] Chunk-%d queued to audio channel",
					connID, audioChunks)
			}
		}
	}
}

// GetAudioChannel returns the audio channel for a session
func (s *TTSService) GetAudioChannel(connID string) (<-chan []byte, <-chan error, error) {
	s.sessionsLock.RLock()
	defer s.sessionsLock.RUnlock()

	session, exists := s.sessions[connID]
	if !exists {
		return nil, nil, fmt.Errorf("no session found for connection: %s", connID)
	}

	return session.audioChan, session.errorChan, nil
}

// CloseSession ends a streaming session
func (s *TTSService) CloseSession(connID string) error {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	session, exists := s.sessions[connID]
	if !exists {
		return nil
	}

	session.cancel()
	delete(s.sessions, connID)
	return nil
}
