// Package service — LiveKit Room management via REST API.
// We call the LiveKit RPC endpoints directly using a pre-signed admin token,
// avoiding the heavy server-sdk-go dependency.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LiveKitRoom mirrors the JSON shape returned by LiveKit's room endpoints.
type LiveKitRoom struct {
	Sid              string `json:"sid"`
	Name             string `json:"name"`
	NumParticipants  int    `json:"num_participants"`
	MaxParticipants  uint32 `json:"max_participants"`
	CreationTime     int64  `json:"creation_time"`
	EmptyTimeout     uint32 `json:"empty_timeout"`
	ActiveRecording  bool   `json:"active_recording"`
}

// LiveKitParticipant mirrors the participant info shape.
type LiveKitParticipant struct {
	Sid      string `json:"sid"`
	Identity string `json:"identity"`
	Name     string `json:"name"`
	State    int    `json:"state"`
}

// RoomService manages LiveKit rooms via REST.
type RoomService struct {
	host      string // e.g. "http://livekit:7880"
	tokenSvc  *TokenService
	httpClient *http.Client
}

func NewRoomService(host string, tokenSvc *TokenService) *RoomService {
	// Normalise host — strip trailing slash, ensure http scheme
	host = strings.TrimRight(host, "/")
	if !strings.HasPrefix(host, "http") {
		host = strings.Replace(host, "ws://", "http://", 1)
		host = strings.Replace(host, "wss://", "https://", 1)
		if !strings.HasPrefix(host, "http") {
			host = "http://" + host
		}
	}
	return &RoomService{
		host:     host,
		tokenSvc: tokenSvc,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// adminToken issues a short-lived server-side admin token for REST calls.
func (s *RoomService) adminToken() (string, error) {
	return s.tokenSvc.IssueToken(ParticipantGrants{
		Identity: "server",
		Name:     "Server",
		TTL:      2 * time.Minute,
	})
}

func (s *RoomService) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	token, err := s.adminToken()
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.host+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("livekit request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("livekit %s %s: status %d — %s", method, path, resp.StatusCode, string(data))
	}
	return data, nil
}

// CreateRoom creates (or joins existing) a LiveKit room.
func (s *RoomService) CreateRoom(ctx context.Context, name string, maxParticipants uint32, emptyTimeout uint32) (*LiveKitRoom, error) {
	if emptyTimeout == 0 {
		emptyTimeout = 300
	}
	payload := map[string]interface{}{
		"name":             name,
		"max_participants": maxParticipants,
		"empty_timeout":    emptyTimeout,
	}
	data, err := s.do(ctx, http.MethodPost, "/twirp/livekit.RoomService/CreateRoom", payload)
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}
	var room LiveKitRoom
	if err := json.Unmarshal(data, &room); err != nil {
		return nil, fmt.Errorf("parse room: %w", err)
	}
	return &room, nil
}

// ListRooms lists all active rooms, optionally filtering by name.
func (s *RoomService) ListRooms(ctx context.Context, names ...string) ([]LiveKitRoom, error) {
	payload := map[string]interface{}{"names": names}
	data, err := s.do(ctx, http.MethodPost, "/twirp/livekit.RoomService/ListRooms", payload)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	var resp struct {
		Rooms []LiveKitRoom `json:"rooms"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse rooms: %w", err)
	}
	return resp.Rooms, nil
}

// GetRoom returns info about a single room, or nil if not found.
func (s *RoomService) GetRoom(ctx context.Context, name string) (*LiveKitRoom, error) {
	rooms, err := s.ListRooms(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(rooms) == 0 {
		return nil, nil
	}
	return &rooms[0], nil
}

// EndRoom terminates a room and disconnects all participants.
func (s *RoomService) EndRoom(ctx context.Context, name string) error {
	_, err := s.do(ctx, http.MethodPost, "/twirp/livekit.RoomService/DeleteRoom",
		map[string]string{"room": name})
	if err != nil {
		return fmt.Errorf("end room: %w", err)
	}
	return nil
}

// ListParticipants returns all current participants in a room.
func (s *RoomService) ListParticipants(ctx context.Context, roomName string) ([]LiveKitParticipant, error) {
	data, err := s.do(ctx, http.MethodPost, "/twirp/livekit.RoomService/ListParticipants",
		map[string]string{"room": roomName})
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	var resp struct {
		Participants []LiveKitParticipant `json:"participants"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse participants: %w", err)
	}
	return resp.Participants, nil
}

// RemoveParticipant kicks a participant from the room.
func (s *RoomService) RemoveParticipant(ctx context.Context, roomName, identity string) error {
	_, err := s.do(ctx, http.MethodPost, "/twirp/livekit.RoomService/RemoveParticipant",
		map[string]string{"room": roomName, "identity": identity})
	if err != nil {
		return fmt.Errorf("remove participant: %w", err)
	}
	return nil
}

// MuteTrack mutes/unmutes a published track.
func (s *RoomService) MuteTrack(ctx context.Context, roomName, identity, trackSid string, muted bool) error {
	_, err := s.do(ctx, http.MethodPost, "/twirp/livekit.RoomService/MutePublishedTrack",
		map[string]interface{}{
			"room": roomName, "identity": identity,
			"track_sid": trackSid, "muted": muted,
		})
	if err != nil {
		return fmt.Errorf("mute track: %w", err)
	}
	return nil
}
