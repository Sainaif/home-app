package handlers

import (
	"bufio"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

type EventHandler struct {
	eventService *services.EventService
}

func NewEventHandler(eventService *services.EventService) *EventHandler {
	return &EventHandler{eventService: eventService}
}

// StreamEvents handles SSE connections
func (h *EventHandler) StreamEvents(c *fiber.Ctx) error {
	// Get user ID from context
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Set SSE headers - CRITICAL for keeping connection alive
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// IMPORTANT: SetBodyStreamWriter runs in a goroutine
	// All resource management must happen INSIDE the callback
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Subscribe to events INSIDE the stream writer
		eventChan := h.eventService.Subscribe(userID)
		defer h.eventService.Unsubscribe(userID)

		// Create heartbeat ticker INSIDE the stream writer
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		// Send initial connection message
		fmt.Fprintf(w, "data: {\"type\":\"connected\",\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
		if err := w.Flush(); err != nil {
			return
		}

		// Main event loop - this keeps the connection open
		for {
			select {
			case event, ok := <-eventChan:
				if !ok {
					return
				}

				fmt.Fprint(w, event.FormatSSE())
				if err := w.Flush(); err != nil {
					return
				}

			case <-heartbeat.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})

	return nil
}
