package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"platform/api/internal/model"
	"platform/api/internal/repository"
)

// RoomService manages event rooms and delegates media operations to media-service.
type RoomService struct {
	roomRepo          *repository.RoomRepository
	eventRepo         *repository.EventRepository
	regRepo           *repository.RegistrationRepository
	mediaServiceURL   string
	mediaServiceToken string
	livekitURL        string
	logger            *zap.Logger
	httpClient        *http.Client
}

func NewRoomService(
	roomRepo *repository.RoomRepository,
	eventRepo *repository.EventRepository,
	regRepo *repository.RegistrationRepository,
	mediaServiceURL, mediaServiceToken, livekitURL string,
	logger *zap.Logger,
) *RoomService {
	return &RoomService{
		roomRepo:          roomRepo,
		eventRepo:         eventRepo,
		regRepo:           regRepo,
		mediaServiceURL:   mediaServiceURL,
		mediaServiceToken: mediaServiceToken,
		livekitURL:        livekitURL,
		logger:            logger,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateRoom creates a DB room record and a LiveKit room via media-service.
func (s *RoomService) CreateRoom(ctx context.Context, tenantID, userID string, req model.CreateRoomRequest) (*model.Room, error) {
	// Verify event belongs to tenant
	event, err := s.eventRepo.FindByID(ctx, tenantID, req.EventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("мероприятие не найдено")
	}

	maxP := req.MaxParticipants
	if maxP <= 0 {
		maxP = 100
	}

	// Generate a unique room name
	livekitName := fmt.Sprintf("%s-%s-%d", tenantID[:8], req.EventID[:8], time.Now().Unix())

	// Create the room in DB first
	room, err := s.roomRepo.Create(ctx, tenantID, req.EventID, livekitName, req.Mode, maxP)
	if err != nil {
		return nil, fmt.Errorf("создание комнаты: %w", err)
	}

	// Ask media-service to create the LiveKit room
	if err := s.createLiveKitRoom(ctx, livekitName, uint32(maxP)); err != nil {
		s.logger.Warn("livekit room creation failed", zap.String("room", livekitName), zap.Error(err))
		// Non-fatal: LiveKit room is created lazily when first participant joins
	}

	// Mark room as active
	_ = s.roomRepo.SetStatus(ctx, room.ID, "active")
	room.Status = "active"

	return room, nil
}

// JoinRoom issues a LiveKit token for a participant to join a room.
func (s *RoomService) JoinRoom(ctx context.Context, tenantID, roomID, userID, displayName, role string) (*model.RoomTokenResponse, error) {
	room, err := s.roomRepo.FindByID(ctx, tenantID, roomID)
	if err != nil || room == nil {
		return nil, fmt.Errorf("комната не найдена")
	}
	if room.Status == "ended" {
		return nil, fmt.Errorf("комната завершена")
	}

	// Validate registration for non-host roles
	if role == "viewer" || role == "speaker" || role == "moderator" {
		reg, err := s.regRepo.Find(ctx, tenantID, room.EventID, userID)
		if err != nil {
			return nil, fmt.Errorf("проверка регистрации: %w", err)
		}
		if reg == nil {
			return nil, fmt.Errorf("необходима регистрация на мероприятие")
		}
	}

	identity := fmt.Sprintf("%s-%s", role, userID)
	token, err := s.getMediaToken(ctx, room.LiveKitRoomName, identity, displayName, role)
	if err != nil {
		return nil, fmt.Errorf("получение токена: %w", err)
	}

	return &model.RoomTokenResponse{
		Token:      token,
		RoomName:   room.LiveKitRoomName,
		LiveKitURL: s.livekitURL,
	}, nil
}

// GetRooms returns all rooms for an event.
func (s *RoomService) GetRooms(ctx context.Context, tenantID, eventID string) ([]model.Room, error) {
	return s.roomRepo.FindByEvent(ctx, tenantID, eventID)
}

// EndRoom closes a room.
func (s *RoomService) EndRoom(ctx context.Context, tenantID, roomID string) error {
	room, err := s.roomRepo.FindByID(ctx, tenantID, roomID)
	if err != nil || room == nil {
		return fmt.Errorf("комната не найдена")
	}
	return s.roomRepo.SetStatus(ctx, roomID, "ended")
}

// ── media-service calls ────────────────────────────────────────────────────────

func (s *RoomService) createLiveKitRoom(ctx context.Context, name string, maxParticipants uint32) error {
	body, _ := json.Marshal(map[string]interface{}{
		"name":             name,
		"max_participants": maxParticipants,
		"empty_timeout":    300,
	})
	_, err := s.mediaCall(ctx, http.MethodPost, "/media/rooms", body)
	return err
}

func (s *RoomService) getMediaToken(ctx context.Context, roomName, identity, displayName, role string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"room_name":    roomName,
		"identity":     identity,
		"display_name": displayName,
		"role":         role,
	})
	data, err := s.mediaCall(ctx, http.MethodPost, "/media/token", body)
	if err != nil {
		return "", err
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse media token response: %w", err)
	}
	return resp.Token, nil
}

func (s *RoomService) mediaCall(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.mediaServiceURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.mediaServiceToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("media service: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("media service %s: %d — %s", path, resp.StatusCode, string(data))
	}
	return data, nil
}
