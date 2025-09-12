package tunnelgrpc

import (
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

type TunnelServer struct {
	api.UnimplementedTunnelServiceServer
	sm *SessionManager
}

func NewTunnelServer(sessionManager *SessionManager) *TunnelServer {
	return &TunnelServer{
		sm: sessionManager,
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

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Client for tunnel %s disconnected.", tunnelID)
			return ctx.Err()
		default:
			// В этой версии мы не ждем сообщений от клиента после регистрации,
			// а только слушаем отключение. Позже сюда добавится обработка Data сообщений.
			// Чтобы не грузить CPU, можно добавить небольшую паузу или использовать
			// более сложный механизм с каналами.
		}
	}
}
