package tts

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	pb "gcloud-tts-rpc-websockets-go/google/cloud/texttospeech/v1beta1"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

// TTSSession represents an active TTS streaming session
type TTSSession struct {
	stream     pb.TextToSpeech_StreamingSynthesizeClient
	streamLock sync.Mutex
	config     *pb.StreamingSynthesizeConfig
	audioChan  chan []byte
	errorChan  chan error
	ctx        context.Context
	cancel     context.CancelFunc
}

// TTSService handles text-to-speech conversion using Google Cloud TTS
type TTSService struct {
	client       pb.TextToSpeechClient
	sampleRate   int32
	sessions     map[string]*TTSSession // Key is WebSocket connection ID
	sessionsLock sync.RWMutex
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

	// Check if session already exists
	if _, exists := s.sessions[connID]; exists {
		return fmt.Errorf("session already exists for connection: %s", connID)
	}

	// Create cancellable context for the session
	sessionCtx, cancel := context.WithCancel(ctx)

	// Initialize streaming client
	stream, err := s.client.StreamingSynthesize(sessionCtx)
	if err != nil {
		cancel()
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
		stream:    stream,
		config:    config,
		audioChan: make(chan []byte, 100),
		errorChan: make(chan error, 1),
		ctx:       sessionCtx,
		cancel:    cancel,
	}

	// Send initial configuration
	err = stream.Send(&pb.StreamingSynthesizeRequest{
		StreamingRequest: &pb.StreamingSynthesizeRequest_StreamingConfig{
			StreamingConfig: config,
		},
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to send config: %v", err)
	}

	// Start receiving audio in background
	go s.receiveAudio(session)

	// Store session
	s.sessions[connID] = session
	return nil
}

// SynthesizeText sends text to be synthesized in an existing session
func (s *TTSService) SynthesizeText(connID string, text string) error {
	s.sessionsLock.RLock()
	session, exists := s.sessions[connID]
	s.sessionsLock.RUnlock()

	if !exists {
		return fmt.Errorf("no session found for connection: %s", connID)
	}

	// Lock the stream for sending
	session.streamLock.Lock()
	defer session.streamLock.Unlock()

	// Send text input
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

	return nil
}

// receiveAudio handles receiving audio data from the gRPC stream
func (s *TTSService) receiveAudio(session *TTSSession) {
	defer close(session.audioChan)
	defer close(session.errorChan)

	for {
		resp, err := session.stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			session.errorChan <- fmt.Errorf("failed to receive: %v", err)
			return
		}

		select {
		case <-session.ctx.Done():
			return
		case session.audioChan <- resp.AudioContent:
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
