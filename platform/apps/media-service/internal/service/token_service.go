// Package service provides LiveKit token generation and room management.
package service

import (
	"fmt"
	"time"

	lkauth "github.com/livekit/protocol/auth"
)

// ParticipantGrants defines what a token holder can do in the room.
type ParticipantGrants struct {
	RoomName    string
	Identity    string // unique per participant per room
	Name        string // display name
	CanPublish  bool   // send audio/video (speakers, hosts)
	CanSubscribe bool  // receive audio/video (always true for viewers)
	CanPublishData bool // send data channel messages (chat, reactions)
	IsHidden    bool   // viewer mode: not visible in participant list
	TTL         time.Duration
}

// TokenService issues LiveKit access tokens.
type TokenService struct {
	apiKey    string
	apiSecret string
}

func NewTokenService(apiKey, apiSecret string) *TokenService {
	return &TokenService{apiKey: apiKey, apiSecret: apiSecret}
}

// IssueToken creates a signed JWT for LiveKit room access.
func (s *TokenService) IssueToken(g ParticipantGrants) (string, error) {
	if g.TTL == 0 {
		g.TTL = 4 * time.Hour
	}

	at := lkauth.NewAccessToken(s.apiKey, s.apiSecret)
	grant := &lkauth.VideoGrant{
		RoomJoin:     true,
		Room:         g.RoomName,
		CanPublish:   &g.CanPublish,
		CanSubscribe: &g.CanSubscribe,
		CanPublishData: &g.CanPublishData,
		Hidden:       g.IsHidden,
	}

	at.AddGrant(grant).
		SetIdentity(g.Identity).
		SetName(g.Name).
		SetValidFor(g.TTL)

	token, err := at.ToJWT()
	if err != nil {
		return "", fmt.Errorf("token sign: %w", err)
	}
	return token, nil
}

// IssueHostToken issues a full-permission token for event hosts/co-hosts.
func (s *TokenService) IssueHostToken(roomName, identity, displayName string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName:       roomName,
		Identity:       identity,
		Name:           displayName,
		CanPublish:     true,
		CanSubscribe:   true,
		CanPublishData: true,
		IsHidden:       false,
	})
}

// IssueSpeakerToken issues a publish+subscribe token for speakers.
func (s *TokenService) IssueSpeakerToken(roomName, identity, displayName string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName:       roomName,
		Identity:       identity,
		Name:           displayName,
		CanPublish:     true,
		CanSubscribe:   true,
		CanPublishData: true,
		IsHidden:       false,
	})
}

// IssueViewerToken issues a subscribe-only hidden token for passive viewers.
// Used for webinar audience and broadcast viewers (up to 10k).
func (s *TokenService) IssueViewerToken(roomName, identity, displayName string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName:       roomName,
		Identity:       identity,
		Name:           displayName,
		CanPublish:     false,
		CanSubscribe:   true,
		CanPublishData: false,
		IsHidden:       true,
		TTL:            6 * time.Hour,
	})
}

// IssueModeratorToken issues a subscribe+data token for moderators (no A/V publish).
func (s *TokenService) IssueModeratorToken(roomName, identity, displayName string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName:       roomName,
		Identity:       identity,
		Name:           displayName,
		CanPublish:     false,
		CanSubscribe:   true,
		CanPublishData: true,
		IsHidden:       false,
	})
}
