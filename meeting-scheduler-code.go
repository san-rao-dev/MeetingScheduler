// Main application entry point
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Domain Models
type Event struct {
	ID               string    `json:"id"`
	Title            string    `json:"title" binding:"required"`
	Description      string    `json:"description"`
	OrganizerID      string    `json:"organizerId" binding:"required"`
	RequiredDuration int       `json:"requiredDuration" binding:"required"` // in minutes
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type TimeSlot struct {
	ID        string    `json:"id"`
	EventID   string    `json:"eventId"`
	StartTime time.Time `json:"startTime" binding:"required"`
	EndTime   time.Time `json:"endTime" binding:"required"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserAvailability struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	EventID    string    `json:"eventId"`
	TimeSlotID string    `json:"timeslotId" binding:"required"`
	Status     string    `json:"status" binding:"required"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Recommendation struct {
	TimeSlot              TimeSlot `json:"timeslot"`
	AvailableUsers        []string `json:"availableUsers"`
	UnavailableUsers      []string `json:"unavailableUsers"`
	AvailabilityPercentage float64  `json:"availabilityPercentage"`
}

// Request/Response models
type CreateEventRequest struct {
	Title            string `json:"title" binding:"required"`
	Description      string `json:"description"`
	OrganizerID      string `json:"organizerId" binding:"required"`
	RequiredDuration int    `json:"requiredDuration" binding:"required"`
}

type CreateTimeSlotRequest struct {
	StartTime time.Time `json:"startTime" binding:"required"`
	EndTime   time.Time `json:"endTime" binding:"required"`
}

type UserAvailabilityRequest struct {
	TimeSlotID string `json:"timeslotId" binding:"required"`
	Status     string `json:"status" binding:"required,oneof=available unavailable"`
}

type RecommendationsResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
}

// In-memory storage (would use a database in production)
var events = make(map[string]Event)
var timeSlots = make(map[string]TimeSlot)
var userAvailability = make(map[string]UserAvailability)

func main() {
	router := gin.Default()

	// Event endpoints
	router.POST("/api/v1/events", createEvent)
	router.GET("/api/v1/events", listEvents)
	router.GET("/api/v1/events/:eventId", getEvent)
	router.PUT("/api/v1/events/:eventId", updateEvent)
	router.DELETE("/api/v1/events/:eventId", deleteEvent)

	// TimeSlot endpoints
	router.POST("/api/v1/events/:eventId/timeslots", createTimeSlot)
	router.GET("/api/v1/events/:eventId/timeslots", listTimeSlots)
	router.PUT("/api/v1/events/:eventId/timeslots/:timeslotId", updateTimeSlot)
	router.DELETE("/api/v1/events/:eventId/timeslots/:timeslotId", deleteTimeSlot)

	// UserAvailability endpoints
	router.POST("/api/v1/events/:eventId/users/:userId/availability", createUserAvailability)
	router.GET("/api/v1/events/:eventId/users/:userId/availability", getUserAvailability)
	router.PUT("/api/v1/events/:eventId/users/:userId/availability/:timeslotId", updateUserAvailability)
	router.DELETE("/api/v1/events/:eventId/users/:userId/availability/:timeslotId", deleteUserAvailability)

	// Recommendations endpoint
	router.GET("/api/v1/events/:eventId/recommendations", getRecommendations)

	// Start the server
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Event handlers
func createEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	event := Event{
		ID:               uuid.New().String(),
		Title:            req.Title,
		Description:      req.Description,
		OrganizerID:      req.OrganizerID,
		RequiredDuration: req.RequiredDuration,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	events[event.ID] = event
	c.JSON(http.StatusCreated, event)
}

func listEvents(c *gin.Context) {
	var eventList []Event
	for _, event := range events {
		eventList = append(eventList, event)
	}
	c.JSON(http.StatusOK, eventList)
}

func getEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	event, exists := events[eventID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	c.JSON(http.StatusOK, event)
}

func updateEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	_, exists := events[eventID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event := events[eventID]
	event.Title = req.Title
	event.Description = req.Description
	event.OrganizerID = req.OrganizerID
	event.RequiredDuration = req.RequiredDuration
	event.UpdatedAt = time.Now()
	
	events[eventID] = event
	c.JSON(http.StatusOK, event)
}

func deleteEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	_, exists := events[eventID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	delete(events, eventID)
	c.JSON(http.StatusNoContent, nil)
}

// TimeSlot handlers
func createTimeSlot(c *gin.Context) {
	eventID := c.Param("eventId")
	_, exists := events[eventID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	var req CreateTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate time range
	if req.EndTime.Before(req.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End time must be after start time"})
		return
	}

	now := time.Now()
	timeSlot := TimeSlot{
		ID:        uuid.New().String(),
		EventID:   eventID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		CreatedAt: now,
		UpdatedAt: now,
	}

	timeSlots[timeSlot.ID] = timeSlot
	c.JSON(http.StatusCreated, timeSlot)
}

func listTimeSlots(c *gin.Context) {
	eventID := c.Param("eventId")
	var slotList []TimeSlot
	for _, slot := range timeSlots {
		if slot.EventID == eventID {
			slotList = append(slotList, slot)
		}
	}
	c.JSON(http.StatusOK, slotList)
}

func updateTimeSlot(c *gin.Context) {
	timeslotID := c.Param("timeslotId")
	slot, exists := timeSlots[timeslotID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Time slot not found"})
		return
	}

	var req CreateTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate time range
	if req.EndTime.Before(req.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End time must be after start time"})
		return
	}

	slot.StartTime = req.StartTime
	slot.EndTime = req.EndTime
	slot.UpdatedAt = time.Now()
	
	timeSlots[timeslotID] = slot
	c.JSON(http.StatusOK, slot)
}

func deleteTimeSlot(c *gin.Context) {
	timeslotID := c.Param("timeslotId")
	_, exists := timeSlots[timeslotID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Time slot not found"})
		return
	}

	delete(timeSlots, timeslotID)
	c.JSON(http.StatusNoContent, nil)
}

// UserAvailability handlers
func createUserAvailability(c *gin.Context) {
	eventID := c.Param("eventId")
	userID := c.Param("userId")
	
	_, eventExists := events[eventID]
	if !eventExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	var req UserAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	_, slotExists := timeSlots[req.TimeSlotID]
	if !slotExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Time slot not found"})
		return
	}

	now := time.Now()
	availability := UserAvailability{
		ID:         uuid.New().String(),
		UserID:     userID,
		EventID:    eventID,
		TimeSlotID: req.TimeSlotID,
		Status:     req.Status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	userAvailability[availability.ID] = availability
	c.JSON(http.StatusCreated, availability)
}

func getUserAvailability(c *gin.Context) {
	eventID := c.Param("eventId")
	userID := c.Param("userId")
	
	var availabilityList []UserAvailability
	for _, avail := range userAvailability {
		if avail.EventID == eventID && avail.UserID == userID {
			availabilityList = append(availabilityList, avail)
		}
	}
	c.JSON(http.StatusOK, availabilityList)
}

func updateUserAvailability(c *gin.Context) {
	eventID := c.Param("eventId")
	userID := c.Param("userId")
	timeslotID := c.Param("timeslotId")
	
	// Find the availability record
	var targetAvail UserAvailability
	var found bool
	
	for _, avail := range userAvailability {
		if avail.EventID == eventID && avail.UserID == userID && avail.TimeSlotID == timeslotID {
			targetAvail = avail
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Availability record not found"})
		return
	}
	
	var req UserAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Update the record
	targetAvail.Status = req.Status
	targetAvail.UpdatedAt = time.Now()
	
	userAvailability[targetAvail.ID] = targetAvail
	c.JSON(http.StatusOK, targetAvail)
}

func deleteUserAvailability(c *gin.Context) {
	eventID := c.Param("eventId")
	userID := c.Param("userId")
	timeslotID := c.Param("timeslotId")
	
	// Find the availability record
	var targetID string
	var found bool
	
	for id, avail := range userAvailability {
		if avail.EventID == eventID && avail.UserID == userID && avail.TimeSlotID == timeslotID {
			targetID = id
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Availability record not found"})
		return
	}
	
	delete(userAvailability, targetID)
	c.JSON(http.StatusNoContent, nil)
}

// Recommendation handler
func getRecommendations(c *gin.Context) {
	eventID := c.Param("eventId")
	event, exists := events[eventID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	
	// Get all time slots for this event
	var eventSlots []TimeSlot
	for _, slot := range timeSlots {
		if slot.EventID == eventID {
			eventSlots = append(eventSlots, slot)
		}
	}
	
	if len(eventSlots) == 0 {
		c.JSON(http.StatusOK, RecommendationsResponse{Recommendations: []Recommendation{}})
		return
	}
	
	// Get all unique users for this event
	uniqueUsers := make(map[string]bool)
	for _, avail := range userAvailability {
		if avail.EventID == eventID {
			uniqueUsers[avail.UserID] = true
		}
	}
	
	// If no users have provided availability
	if len(uniqueUsers) == 0 {
		c.JSON(http.StatusOK, RecommendationsResponse{Recommendations: []Recommendation{}})
		return
	}
	
	// For each time slot, calculate user availability
	var recommendations []Recommendation
	for _, slot := range eventSlots {
		// Check if slot duration is sufficient for the meeting
		slotDuration := slot.EndTime.Sub(slot.StartTime).Minutes()
		if slotDuration < float64(event.RequiredDuration) {
			continue // Skip slots that are too short
		}
		
		var availableUsers []string
		var unavailableUsers []string
		
		// For each user, check if they've indicated availability for this slot
		for userID := range uniqueUsers {
			isAvailable := false
			
			// Check if user has explicitly marked availability for this slot
			for _, avail := range userAvailability {
				if avail.EventID == eventID && avail.UserID == userID && avail.TimeSlotID == slot.ID && avail.Status == "available" {
					isAvailable = true
					break
				}
			}
			
			if isAvailable {
				availableUsers = append(availableUsers, userID)
			} else {
				unavailableUsers = append(unavailableUsers, userID)
			}
		}
		
		availabilityPercentage := float64(len(availableUsers)) / float64(len(uniqueUsers)) * 100
		
		recommendations = append(recommendations, Recommendation{
			TimeSlot:              slot,
			AvailableUsers:        availableUsers,
			UnavailableUsers:      unavailableUsers,
			AvailabilityPercentage: availabilityPercentage,
		})
	}
	
	// Sort recommendations by availability percentage (highest first)
	// In a real implementation, we'd use sort.Slice here
	// This is a simplified bubble sort
	for i := 0; i < len(recommendations); i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[i].AvailabilityPercentage < recommendations[j].AvailabilityPercentage {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}
	
	c.JSON(http.StatusOK, RecommendationsResponse{Recommendations: recommendations})
}
