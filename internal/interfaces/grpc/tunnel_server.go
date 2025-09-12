package tunnelgrpc

import (
	"io"
	"log"
	"sync"

	"github.com/waste3d/ghost-tunnel/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]api.TunnelService_EstablishTunnelServer
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]api.TunnelService_EstablishTunnelServer),
	}
}

func (sm *SessionManager) Add(tunnelID string, stream api.TunnelService_EstablishTunnelServer) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[tunnelID] = stream
}

func (sm *SessionManager) Get(tunnelID string) (api.TunnelService_EstablishTunnelServer, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	stream, ok := sm.sessions[tunnelID]
	return stream, ok
}

func (sm *SessionManager) Remove(tunnelID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, tunnelID)
}

type ConnectionManager struct {
	channels map[string]chan []byte
	mu       sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		channels: make(map[string]chan []byte),
	}
}

func (cm *ConnectionManager) Add(connID string, ch chan []byte) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.channels[connID] = ch
}

func (cm *ConnectionManager) Remove(connID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	ch, ok := cm.channels[connID]
	if ok {
		close(ch)
		delete(cm.channels, connID)
	}
}

type TunnelServer struct {
	api.UnimplementedTunnelServiceServer
	sm      *SessionManager
	connMgr *ConnectionManager
}

func NewTunnelServer(sessionManager *SessionManager, connMgr *ConnectionManager) *TunnelServer {
	return &TunnelServer{
		sm:      sessionManager,
		connMgr: connMgr,
	}
}

func (s *TunnelServer) EstablishTunnel(stream api.TunnelService_EstablishTunnelServer) error {
	log.Println("Client connected")

	msg, err := stream.Recv()
	if err != nil {
		log.Println("Error receiving message:", err)
		return err
	}

	reg := msg.GetRegister()
	if reg == nil {
		log.Println("First message was not Register. Disconnecting client.")
		return status.Errorf(codes.InvalidArgument, "first message must be a Register message")
	}

	tunnelID := reg.GetTunnelId()
	log.Printf("Client registered for tunnel ID: %s", tunnelID)

	s.sm.Add(tunnelID, stream)
	defer s.sm.Remove(tunnelID)

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				log.Printf("Client for tunnel %s disconnected.", tunnelID)
				return nil
			}
			log.Printf("Error receiving from client tunnel %s: %v", tunnelID, err)
			return err
		}

		if data := msg.GetData(); data != nil {
			s.connMgr.mu.RLock()
			ch, ok := s.connMgr.channels[data.GetConnectionId()]
			s.connMgr.mu.RUnlock()

			if ok {
				ch <- data.GetChunk()
			}
		}
	}
}
