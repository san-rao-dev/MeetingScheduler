package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Event endpoints
	router.POST("/api/v1/events", createEvent)
	router.GET("/api/v1/events/:eventId", getEvent)
	router.PUT("/api/v1/events/:eventId", updateEvent)
	router.DELETE("/api/v1/events/:eventId", deleteEvent)

	// TimeSlot endpoints
	router.POST("/api/v1/events/:eventId/timeslots", createTimeSlot)
	router.GET("/api/v1/events/:eventId/timeslots", listTimeSlots)

	// UserAvailability endpoints
	router.POST("/api/v1/events/:eventId/users/:userId/availability", createUserAvailability)

	// Recommendations endpoint
	router.GET("/api/v1/events/:eventId/recommendations", getRecommendations)

	return router
}

func TestCreateEvent(t *testing.T) {
	// Clear data
	events = make(map[string]Event)
	
	router := setupRouter()
	
	// Create event request
	eventReq := CreateEventRequest{
		Title:            "Team Meeting",
		Description:      "Weekly team sync",
		OrganizerID:      "user1",
		RequiredDuration: 60,
	}
	
	reqBody, _ := json.Marshal(eventReq)
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response Event
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, eventReq.Title, response.Title)
	assert.Equal(t, eventReq.Description, response.Description)
	assert.Equal(t, eventReq.OrganizerID, response.OrganizerID)
	assert.Equal(t, eventReq.RequiredDuration, response.RequiredDuration)
	assert.Equal(t, "active", response.Status)
	assert.NotEmpty(t, response.ID)
	
	// Verify event was stored
	assert.Equal(t, 1, len(events))
}

func TestCreateTimeSlot(t *testing.T) {
	// Clear data
	events = make(map[string]Event)
	timeSlots = make(map[string]TimeSlot)
	
	router := setupRouter()
	
	// First create an event
	eventReq := CreateEventRequest{
		Title:            "Team Meeting",
		Description:      "Weekly team sync",
		OrganizerID:      "user1",
		RequiredDuration: 60,
	}
	
	reqBody, _ := json.Marshal(eventReq)
	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	var event Event
	_ = json.Unmarshal(w.Body.Bytes(), &event)
	
	// Now create a time slot for this event
	startTime := time.Now().Add(24 * time.Hour)  // Tomorrow
	endTime := startTime.Add(2 * time.Hour)      // 2 hours later
	
	timeSlotReq := CreateTimeSlotRequest{
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	reqBody