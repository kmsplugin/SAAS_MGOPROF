// Package service provides LiveKit-compatible token generation.
// Implements the LiveKit JWT spec without the SDK dependency:
// https://docs.livekit.io/home/get-started/authentication/
package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// VideoGrant holds LiveKit room access permissions.
// These map 1:1 to LiveKit's JWT video grant claims.
type VideoGrant struct {
	RoomCreate     bool   `json:"roomCreate,omitempty"`
	RoomList       bool   `json:"roomList,omitempty"`
	RoomRecord     bool   `json:"roomRecord,omitempty"`
	RoomAdmin      bool   `json:"roomAdmin,omitempty"`
	RoomJoin       bool   `json:"roomJoin,omitempty"`
	Room           string `json:"room,omitempty"`
	CanPublish     *bool  `json:"canPublish,omitempty"`
	CanSubscribe   *bool  `json:"canSubscribe,omitempty"`
	CanPublishData *bool  `json:"canPublishData,omitempty"`
	Hidden         bool   `json:"hidden,omitempty"`
	Recorder       bool   `json:"recorder,omitempty"`
}

// livekitClaims are the JWT claims for a LiveKit access token.
type livekitClaims struct {
	Video    VideoGrant `json:"video"`
	Identity string     `json:"sub,omitempty"`
	Name     string     `json:"name,omitempty"`
	jwt.RegisteredClaims
}

// TokenService issues LiveKit-compatible access tokens.
type TokenService struct {
	apiKey    string
	apiSecret []byte
}

func NewTokenService(apiKey, apiSecret string) *TokenService {
	return &TokenService{apiKey: apiKey, apiSecret: []byte(apiSecret)}
}

// ParticipantGrants defines what a token holder can do in the room.
type ParticipantGrants struct {
	RoomName       string
	Identity       string
	Name           string
	CanPublish     bool
	CanSubscribe   bool
	CanPublishData bool
	IsHidden       bool
	TTL            time.Duration
}

// IssueToken creates a signed JWT for LiveKit room access.
func (s *TokenService) IssueToken(g ParticipantGrants) (string, error) {
	if g.TTL == 0 {
		g.TTL = 4 * time.Hour
	}

	canPub := g.CanPublish
	canSub := g.CanSubscribe
	canData := g.CanPublishData

	claims := livekitClaims{
		Video: VideoGrant{
			RoomJoin:       true,
			Room:           g.RoomName,
			CanPublish:     &canPub,
			CanSubscribe:   &canSub,
			CanPublishData: &canData,
			Hidden:         g.IsHidden,
		},
		Identity: g.Identity,
		Name:     g.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.apiKey,
			Subject:   g.Identity,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(g.TTL)),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := t.SignedString(s.apiSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return token, nil
}

// IssueHostToken issues a full-permission token for event hosts/co-hosts.
func (s *TokenService) IssueHostToken(roomName, identity, name string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName: roomName, Identity: identity, Name: name,
		CanPublish: true, CanSubscribe: true, CanPublishData: true,
	})
}

// IssueSpeakerToken issues publish+subscribe token for speakers.
func (s *TokenService) IssueSpeakerToken(roomName, identity, name string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName: roomName, Identity: identity, Name: name,
		CanPublish: true, CanSubscribe: true, CanPublishData: true,
	})
}

// IssueViewerToken issues subscribe-only hidden token for passive viewers.
// Used for webinar audience (up to 10k viewers).
func (s *TokenService) IssueViewerToken(roomName, identity, name string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName: roomName, Identity: identity, Name: name,
		CanPublish: false, CanSubscribe: true, CanPublishData: false,
		IsHidden: true, TTL: 6 * time.Hour,
	})
}

// IssueModeratorToken issues subscribe+data token for moderators.
func (s *TokenService) IssueModeratorToken(roomName, identity, name string) (string, error) {
	return s.IssueToken(ParticipantGrants{
		RoomName: roomName, Identity: identity, Name: name,
		CanPublish: false, CanSubscribe: true, CanPublishData: true,
	})
}
