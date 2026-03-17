package secretary

import (
	"context"
	"github.com/armorclaw/bridge/internal/skills"
)

type WebDAVService struct{}

func NewWebDAVService() *WebDAVService {
	return &WebDAVService{}
}

func (s *WebDAVService) ExecuteWebDAV(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return skills.ExecuteWebDAV(ctx, params)
}
