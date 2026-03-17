package secretary

import (
	"context"
	"github.com/armorclaw/bridge/internal/skills"
)

type CalendarService struct{}

func NewCalendarService() *CalendarService {
	return &CalendarService{}
}

func (s *CalendarService) ExecuteCalendar(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return skills.ExecuteCalendar(ctx, params)
}
