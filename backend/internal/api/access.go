package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
)

// canAccessEvent reports whether the requester may view/use an event.
// Admins see everything; users only events they are invited to.
func (s *Server) canAccessEvent(r *http.Request, eventID uuid.UUID) bool {
	if isStaff(r) {
		return true
	}
	var count int64
	s.DB.Model(&models.EventParticipant{}).
		Where("event_id = ? AND user_id = ?", eventID, userIDFrom(r)).
		Count(&count)
	return count > 0
}

// effectiveParticipantCount returns the participant count used for every
// computation (quantities, per-person costs): the number defined on the event at
// creation — NOT how many invitees accepted.
func (s *Server) effectiveParticipantCount(_ uuid.UUID, initial int) int {
	return initial
}
