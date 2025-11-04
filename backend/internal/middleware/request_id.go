package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDKey ContextKey = "requestId"

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Locals(RequestIDKey, requestID)
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}

// GetRequestID extracts the request ID from context
func GetRequestID(c *fiber.Ctx) string {
	requestID, ok := c.Locals(RequestIDKey).(string)
	if !ok {
		return ""
	}
	return requestID
}
