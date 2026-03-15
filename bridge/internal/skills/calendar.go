package skills

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CalDAVConfig represents CalDAV configuration
type CalDAVConfig struct {
	BaseURL   string
	Username  string
	Password  string
	Principal string
}

// CalendarParams represents parameters for calendar operations
type CalendarParams struct {
	Operation   string                 `json:"operation"`
	Username    string                 `json:"username"`
	Password    string                 `json:"password"`
	CalendarURL string                 `json:"calendar_url"`
	EventData   map[string]interface{} `json:"event_data,omitempty"`
	StartTime   string                 `json:"start_time,omitempty"`
	EndTime     string                 `json:"end_time,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Location    string                 `json:"location,omitempty"`
	Attendees   []string               `json:"attendees,omitempty"`
}

// CalendarResult represents the result of calendar operations
type CalendarResult struct {
	Operation         string            `json:"operation"`
	Status            string            `json:"status"`
	Message           string            `json:"message"`
	CalendarID        string            `json:"calendar_id,omitempty"`
	EventID           string            `json:"event_id,omitempty"`
	Events            []CalendarEvent   `json:"events,omitempty"`
	ConflictsDetected bool              `json:"conflicts_detected,omitempty"`
	Timestamp         time.Time         `json:"timestamp"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// CalendarEvent represents a calendar event
type CalendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location"`
	Description string    `json:"description,omitempty"`
	Attendees   []string  `json:"attendees,omitempty"`
	UID         string    `json:"uid"`
}

// mockCalendarClient is a mock CalDAV client for Phase 1
// In production, this will be replaced with github.com/emersion/go-webdav/caldav
type mockCalendarClient struct {
	events       map[string]*mockCalendarEvent
	calendars    map[string]*mockCalendar
	config       *CalDAVConfig
	eventCounter int
}

// mockCalendar represents a mock CalDAV calendar
type mockCalendar struct {
	ID          string
	Description string
	Events      []*mockCalendarEvent
}

// mockCalendarEvent represents a mock calendar event
type mockCalendarEvent struct {
	UID         string
	Title       string
	Description string
	Location    string
	Attendees   []string
	StartTime   time.Time
	EndTime     time.Time
}

// ExecuteCalendar handles calendar operations
func ExecuteCalendar(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	calendarParams, err := parseCalendarParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid calendar parameters: %w", err)
	}

	// Validate parameters
	if err := validateCalendarParams(calendarParams); err != nil {
		return nil, fmt.Errorf("calendar validation failed: %w", err)
	}

	// Get CalDAV configuration
	config, err := getCalDAVConfig(ctx, calendarParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get CalDAV configuration: %w", err)
	}

	// Create mock client
	client := &mockCalendarClient{
		config: config,
		events: make(map[string]*mockCalendarEvent),
	}

	// Execute operation based on operation type
	result, err := executeCalDAVOperation(ctx, client, calendarParams)
	if err != nil {
		return nil, fmt.Errorf("CalDAV operation failed: %w", err)
	}

	return result, nil
}

// parseCalendarParams parses calendar parameters from input
func parseCalendarParams(params map[string]interface{}) (*CalendarParams, error) {
	calendarParams := &CalendarParams{}

	// Extract required parameters
	if operation, ok := params["operation"].(string); ok {
		calendarParams.Operation = strings.ToLower(operation)
	} else {
		return nil, fmt.Errorf("operation parameter is required and must be a string")
	}

	if url, ok := params["calendar_url"].(string); ok {
		calendarParams.CalendarURL = strings.TrimSpace(url)
	} else {
		return nil, fmt.Errorf("calendar_url parameter is required and must be a string")
	}

	if username, ok := params["username"].(string); ok {
		calendarParams.Username = strings.TrimSpace(username)
	}

	if password, ok := params["password"].(string); ok {
		calendarParams.Password = password
	}

	// Extract event data
	if eventData, ok := params["event_data"].(map[string]interface{}); ok {
		calendarParams.EventData = eventData
	}

	// Parse optional fields from event_data if not provided directly
	if title, ok := params["title"].(string); ok {
		calendarParams.Title = strings.TrimSpace(title)
	} else if eventData, ok := calendarParams.EventData["title"].(string); ok {
		calendarParams.Title = strings.TrimSpace(eventData)
	}

	if description, ok := params["description"].(string); ok {
		calendarParams.Description = strings.TrimSpace(description)
	} else if eventData, ok := calendarParams.EventData["description"].(string); ok {
		calendarParams.Description = strings.TrimSpace(eventData)
	}

	if location, ok := params["location"].(string); ok {
		calendarParams.Location = strings.TrimSpace(location)
	} else if eventData, ok := calendarParams.EventData["location"].(string); ok {
		calendarParams.Location = strings.TrimSpace(eventData)
	}

	if attendees, ok := params["attendees"].([]interface{}); ok {
		calendarParams.Attendees = make([]string, len(attendees))
		for i, attendee := range attendees {
			if attendeeStr, ok := attendee.(string); ok {
				calendarParams.Attendees[i] = strings.TrimSpace(attendeeStr)
			}
		}
	} else if eventData, ok := calendarParams.EventData["attendees"].([]interface{}); ok {
		calendarParams.Attendees = make([]string, len(eventData))
		for i, attendee := range eventData {
			if attendeeStr, ok := attendee.(string); ok {
				calendarParams.Attendees[i] = strings.TrimSpace(attendeeStr)
			}
		}
	}

	if startTime, ok := params["start_time"].(string); ok {
		calendarParams.StartTime = strings.TrimSpace(startTime)
	} else if eventData, ok := calendarParams.EventData["start_time"].(string); ok {
		calendarParams.StartTime = strings.TrimSpace(eventData)
	}

	if endTime, ok := params["end_time"].(string); ok {
		calendarParams.EndTime = strings.TrimSpace(endTime)
	} else if eventData, ok := calendarParams.EventData["end_time"].(string); ok {
		calendarParams.EndTime = strings.TrimSpace(eventData)
	}

	return calendarParams, nil
}

// validateCalendarParams validates calendar parameters
func validateCalendarParams(params *CalendarParams) error {
	// Validate operation
	if params.Operation == "" {
		return fmt.Errorf("operation is required")
	}

	validOperations := map[string]bool{
		"list_calendars": true,
		"create_event":   true,
		"get_events":     true,
		"delete_event":   true,
		"get_event":      true,
		"update_event":   true,
	}

	if !validOperations[params.Operation] {
		return fmt.Errorf("invalid operation '%s'. Valid operations: list_calendars, create_event, get_events, delete_event, get_event, update_event", params.Operation)
	}

	// Validate calendar_url
	if params.CalendarURL == "" {
		return fmt.Errorf("calendar_url is required")
	}

	if !strings.HasPrefix(params.CalendarURL, "http://") && !strings.HasPrefix(params.CalendarURL, "https://") {
		return fmt.Errorf("calendar_url must start with http:// or https://")
	}

	// Validate required fields for specific operations
	switch params.Operation {
	case "create_event":
		if params.Title == "" {
			return fmt.Errorf("title is required for create_event operation")
		}
		if params.StartTime == "" {
			return fmt.Errorf("start_time is required for create_event operation")
		}
		if params.EndTime == "" {
			return fmt.Errorf("end_time is required for create_event operation")
		}
	}

	return nil
}

// getCalDAVConfig gets CalDAV configuration
func getCalDAVConfig(ctx context.Context, params *CalendarParams) (*CalDAVConfig, error) {
	// In production, this would load from secure storage or environment variables
	config := &CalDAVConfig{
		BaseURL:   params.CalendarURL,
		Username:  params.Username,
		Password:  params.Password,
		Principal: params.CalendarURL, // Default principal to calendar URL
	}

	// Set principal from username if username provided
	if config.Username != "" {
		config.Principal = fmt.Sprintf("%s/principal/", strings.TrimSuffix(params.CalendarURL, "/"))
	}

	return config, nil
}

// executeCalDAVOperation executes the appropriate CalDAV operation
func executeCalDAVOperation(ctx context.Context, client *mockCalendarClient, params *CalendarParams) (*CalendarResult, error) {
	switch params.Operation {
	case "list_calendars":
		return listCalendars(ctx, client)
	case "create_event":
		return createEvent(ctx, client, params)
	case "get_events":
		return getEvents(ctx, client)
	case "get_event":
		return getEvent(ctx, client, params)
	case "delete_event":
		return deleteEvent(ctx, client, params)
	case "update_event":
		return updateEvent(ctx, client, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", params.Operation)
	}
}

// listCalendars lists all available calendars
func listCalendars(ctx context.Context, client *mockCalendarClient) (*CalendarResult, error) {
	// Create a mock calendar
	calendar := &mockCalendar{
		ID:          "default-calendar",
		Description: "Default CalDAV Calendar",
	}

	events := &CalendarEvent{
		ID:        calendar.ID,
		Title:     calendar.Description,
		StartTime: time.Now(),
		EndTime:   time.Now().AddDate(0, 0, 1),
		Location:  "",
	}

	return &CalendarResult{
		Operation:         "list_calendars",
		Status:            "success",
		Message:           "Found 1 calendar",
		Events:            []CalendarEvent{*events},
		ConflictsDetected: false,
		Timestamp:         time.Now(),
		Metadata: map[string]string{
			"calendar_count": "1",
		},
	}, nil
}

// createEvent creates a new calendar event with conflict detection
func createEvent(ctx context.Context, client *mockCalendarClient, params *CalendarParams) (*CalendarResult, error) {
	// Parse time strings
	startTime, err := time.Parse(time.RFC3339, params.StartTime)
	if err != nil {
		// Try alternative format
		startTime, err = time.Parse("2006-01-02 15:04:05", params.StartTime)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time format: %w", err)
		}
	}

	endTime, err := time.Parse(time.RFC3339, params.EndTime)
	if err != nil {
		// Try alternative format
		endTime, err = time.Parse("2006-01-02 15:04:05", params.EndTime)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time format: %w", err)
		}
	}

	// Conflict detection: check for overlapping events
	conflicts := checkForConflicts(startTime, endTime, client.events)

	// Generate event UID
	eventUID := generateEventUID()

	// Create event
	event := &mockCalendarEvent{
		UID:         eventUID,
		Title:       params.Title,
		Description: params.Description,
		Location:    params.Location,
		Attendees:   params.Attendees,
		StartTime:   startTime,
		EndTime:     endTime,
	}

	client.events[eventUID] = event

	return &CalendarResult{
		Operation:         "create_event",
		Status:            "success",
		Message:           "Event created successfully",
		EventID:           eventUID,
		ConflictsDetected: len(conflicts) > 0,
		Timestamp:         time.Now(),
		Metadata: map[string]string{
			"event_uid":      eventUID,
			"conflict_count": fmt.Sprintf("%d", len(conflicts)),
		},
	}, nil
}

// checkForConflicts checks for overlapping events
func checkForConflicts(startTime, endTime time.Time, events map[string]*mockCalendarEvent) []CalendarEvent {
	var conflicts []CalendarEvent

	for _, event := range events {
		if hasOverlap(startTime, endTime, event.StartTime, event.EndTime) {
			conflicts = append(conflicts, CalendarEvent{
				ID:        event.UID,
				Title:     event.Title,
				StartTime: event.StartTime,
				EndTime:   event.EndTime,
				Location:  event.Location,
				UID:       event.UID,
			})
		}
	}

	return conflicts
}

// hasOverlap checks if two time ranges overlap
func hasOverlap(start1, end1, start2, end2 time.Time) bool {
	// Normalize to start of day
	start1 = time.Date(start1.Year(), start1.Month(), start1.Day(), 0, 0, 0, 0, start1.Location())
	end1 = time.Date(end1.Year(), end1.Month(), end1.Day(), 0, 0, 0, 0, end1.Location())
	start2 = time.Date(start2.Year(), start2.Month(), start2.Day(), 0, 0, 0, 0, start2.Location())
	end2 = time.Date(end2.Year(), end2.Month(), end2.Day(), 0, 0, 0, 0, end2.Location())

	return start1.Before(end2) && end1.After(start2)
}

// getEvents retrieves all events from calendars
func getEvents(ctx context.Context, client *mockCalendarClient) (*CalendarResult, error) {
	events := make([]CalendarEvent, 0, len(client.events))

	for _, event := range client.events {
		events = append(events, CalendarEvent{
			ID:          event.UID,
			Title:       event.Title,
			StartTime:   event.StartTime,
			EndTime:     event.EndTime,
			Location:    event.Location,
			Description: event.Description,
			Attendees:   event.Attendees,
			UID:         event.UID,
		})
	}

	return &CalendarResult{
		Operation:         "get_events",
		Status:            "success",
		Message:           fmt.Sprintf("Found %d events", len(events)),
		Events:            events,
		ConflictsDetected: false,
		Timestamp:         time.Now(),
		Metadata: map[string]string{
			"event_count": fmt.Sprintf("%d", len(events)),
		},
	}, nil
}

// getEvent retrieves a specific event
func getEvent(ctx context.Context, client *mockCalendarClient, params *CalendarParams) (*CalendarResult, error) {
	if params.EventData == nil || params.EventData["uid"] == nil {
		return nil, fmt.Errorf("uid is required to retrieve a specific event")
	}

	eventUID, ok := params.EventData["uid"].(string)
	if !ok {
		return nil, fmt.Errorf("uid must be a string")
	}

	event, exists := client.events[eventUID]
	if !exists {
		return &CalendarResult{
			Operation: "get_event",
			Status:    "not_found",
			Message:   fmt.Sprintf("Event with UID %s not found", eventUID),
			Timestamp: time.Now(),
		}, nil
	}

	return &CalendarResult{
		Operation: "get_event",
		Status:    "success",
		Message:   "Event retrieved successfully",
		EventID:   eventUID,
		Events: []CalendarEvent{{
			ID:          event.UID,
			Title:       event.Title,
			StartTime:   event.StartTime,
			EndTime:     event.EndTime,
			Location:    event.Location,
			Description: event.Description,
			Attendees:   event.Attendees,
			UID:         event.UID,
		}},
		ConflictsDetected: false,
		Timestamp:         time.Now(),
		Metadata: map[string]string{
			"event_uid": eventUID,
		},
	}, nil
}

// deleteEvent deletes a specific event
func deleteEvent(ctx context.Context, client *mockCalendarClient, params *CalendarParams) (*CalendarResult, error) {
	if params.EventData == nil || params.EventData["uid"] == nil {
		return nil, fmt.Errorf("uid is required to delete an event")
	}

	eventUID, ok := params.EventData["uid"].(string)
	if !ok {
		return nil, fmt.Errorf("uid must be a string")
	}

	_, exists := client.events[eventUID]
	if !exists {
		return &CalendarResult{
			Operation: "delete_event",
			Status:    "not_found",
			Message:   fmt.Sprintf("Event with UID %s not found", eventUID),
			Timestamp: time.Now(),
		}, nil
	}

	delete(client.events, eventUID)

	return &CalendarResult{
		Operation: "delete_event",
		Status:    "success",
		Message:   "Event deleted successfully",
		EventID:   eventUID,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"event_uid": eventUID,
		},
	}, nil
}

// updateEvent updates an existing event
func updateEvent(ctx context.Context, client *mockCalendarClient, params *CalendarParams) (*CalendarResult, error) {
	if params.EventData == nil || params.EventData["uid"] == nil {
		return nil, fmt.Errorf("uid is required to update an event")
	}

	eventUID, ok := params.EventData["uid"].(string)
	if !ok {
		return nil, fmt.Errorf("uid must be a string")
	}

	event, exists := client.events[eventUID]
	if !exists {
		return &CalendarResult{
			Operation: "update_event",
			Status:    "not_found",
			Message:   fmt.Sprintf("Event with UID %s not found", eventUID),
			Timestamp: time.Now(),
		}, nil
	}

	// Parse new time strings
	var startTime, endTime time.Time
	var err error

	if params.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, params.StartTime)
		if err != nil {
			startTime, err = time.Parse("2006-01-02 15:04:05", params.StartTime)
			if err != nil {
				return nil, fmt.Errorf("invalid start_time format: %w", err)
			}
		}
		event.StartTime = startTime
	}

	if params.EndTime != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndTime)
		if err != nil {
			endTime, err = time.Parse("2006-01-02 15:04:05", params.EndTime)
			if err != nil {
				return nil, fmt.Errorf("invalid end_time format: %w", err)
			}
		}
		event.EndTime = endTime
	}

	if params.Title != "" {
		event.Title = params.Title
	}

	if params.Description != "" {
		event.Description = params.Description
	}

	if params.Location != "" {
		event.Location = params.Location
	}

	return &CalendarResult{
		Operation: "update_event",
		Status:    "success",
		Message:   "Event updated successfully",
		EventID:   eventUID,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"event_uid": eventUID,
		},
	}, nil
}

// generateEventUID generates a unique event ID
func generateEventUID() string {
	return fmt.Sprintf("cal-%d", time.Now().UnixNano())
}

// ValidateCalendarParams validates calendar parameters (public API)
func ValidateCalendarParams(params map[string]interface{}) error {
	calendarParams, err := parseCalendarParams(params)
	if err != nil {
		return err
	}
	return validateCalendarParams(calendarParams)
}
