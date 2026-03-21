package service

import (
	"context"
	"fmt"

	lksdk "github.com/livekit/server-sdk-go/v2"
	lkproto "github.com/livekit/protocol/livekit"
)

// RoomService manages LiveKit rooms via the Server API.
type RoomService struct {
	client *lksdk.RoomServiceClient
}

func NewRoomService(host, apiKey, apiSecret string) *RoomService {
	return &RoomService{
		client: lksdk.NewRoomServiceClient(host, apiKey, apiSecret),
	}
}

// CreateRoom creates a LiveKit room (idempotent — returns existing room if already exists).
func (s *RoomService) CreateRoom(ctx context.Context, name string, maxParticipants uint32, emptyTimeout uint32) (*lkproto.Room, error) {
	room, err := s.client.CreateRoom(ctx, &lkproto.CreateRoomRequest{
		Name:            name,
		MaxParticipants: maxParticipants,
		EmptyTimeout:    emptyTimeout, // seconds before room auto-closes when empty
	})
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}
	return room, nil
}

// GetRoom returns room info if it exists.
func (s *RoomService) GetRoom(ctx context.Context, name string) (*lkproto.Room, error) {
	rooms, err := s.client.ListRooms(ctx, &lkproto.ListRoomsRequest{Names: []string{name}})
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	if len(rooms.Rooms) == 0 {
		return nil, nil
	}
	return rooms.Rooms[0], nil
}

// EndRoom terminates a room and disconnects all participants.
func (s *RoomService) EndRoom(ctx context.Context, name string) error {
	_, err := s.client.DeleteRoom(ctx, &lkproto.DeleteRoomRequest{Room: name})
	if err != nil {
		return fmt.Errorf("end room: %w", err)
	}
	return nil
}

// ListParticipants returns all current participants in a room.
func (s *RoomService) ListParticipants(ctx context.Context, roomName string) ([]*lkproto.ParticipantInfo, error) {
	resp, err := s.client.ListParticipants(ctx, &lkproto.ListParticipantsRequest{Room: roomName})
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	return resp.Participants, nil
}

// MuteParticipant mutes a track (audio or video) for a participant.
func (s *RoomService) MuteParticipant(ctx context.Context, roomName, identity, trackSid string, muted bool) error {
	_, err := s.client.MutePublishedTrack(ctx, &lkproto.MuteRoomTrackRequest{
		Room:     roomName,
		Identity: identity,
		TrackSid: trackSid,
		Muted:    muted,
	})
	if err != nil {
		return fmt.Errorf("mute participant: %w", err)
	}
	return nil
}

// RemoveParticipant kicks a participant from the room.
func (s *RoomService) RemoveParticipant(ctx context.Context, roomName, identity string) error {
	_, err := s.client.RemoveParticipant(ctx, &lkproto.RoomParticipantIdentity{
		Room:     roomName,
		Identity: identity,
	})
	if err != nil {
		return fmt.Errorf("remove participant: %w", err)
	}
	return nil
}

// SendData sends a data message to all or specific participants (for chat, reactions, etc.).
func (s *RoomService) SendData(ctx context.Context, roomName string, payload []byte, reliable bool, destinationIdentities []string) error {
	_, err := s.client.SendData(ctx, &lkproto.SendDataRequest{
		Room:                  roomName,
		Data:                  payload,
		Kind:                  reliableKind(reliable),
		DestinationIdentities: destinationIdentities,
	})
	if err != nil {
		return fmt.Errorf("send data: %w", err)
	}
	return nil
}

func reliableKind(reliable bool) lkproto.DataPacket_Kind {
	if reliable {
		return lkproto.DataPacket_RELIABLE
	}
	return lkproto.DataPacket_LOSSY
}
