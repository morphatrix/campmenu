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

// effectiveParticipantCount returns the number of counted participants for an
// event, falling back to InitialParticipants when no roster is set yet.
func (s *Server) effectiveParticipantCount(eventID uuid.UUID, initial int) int {
	var count int64
	s.DB.Model(&models.EventParticipant{}).
		Where("event_id = ? AND counted = ?", eventID, true).
		Count(&count)
	if count == 0 {
		return initial
	}
	return int(count)
}
