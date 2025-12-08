package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type ChoreService struct {
	chores           repository.ChoreRepository
	choreAssignments repository.ChoreAssignmentRepository
	users            repository.UserRepository
}

func NewChoreService(
	chores repository.ChoreRepository,
	choreAssignments repository.ChoreAssignmentRepository,
	users repository.UserRepository,
) *ChoreService {
	return &ChoreService{
		chores:           chores,
		choreAssignments: choreAssignments,
		users:            users,
	}
}

type CreateChoreRequest struct {
	Name                 string  `json:"name"`
	Description          *string `json:"description,omitempty"`
	Frequency            string  `json:"frequency"` // daily, weekly, monthly, custom, irregular
	CustomInterval       *int    `json:"customInterval,omitempty"`
	Difficulty           int     `json:"difficulty"`     // 1-5
	Priority             int     `json:"priority"`       // 1-5
	AssignmentMode       string  `json:"assignmentMode"` // manual, round_robin, random
	NotificationsEnabled bool    `json:"notificationsEnabled"`
	ReminderHours        *int    `json:"reminderHours,omitempty"`
}

type AssignChoreRequest struct {
	ChoreID        string    `json:"choreId"`
	AssigneeUserID string    `json:"assigneeUserId"`
	DueDate        time.Time `json:"dueDate"`
}

type UpdateChoreAssignmentRequest struct {
	Status string `json:"status"` // pending, in_progress, done, overdue
}

type ChoreWithAssignment struct {
	Chore      models.Chore            `json:"chore"`
	Assignment *models.ChoreAssignment `json:"assignment,omitempty"`
}

// CreateChore creates a new chore (ADMIN only)
func (s *ChoreService) CreateChore(ctx context.Context, req CreateChoreRequest) (*models.Chore, error) {
	if req.Name == "" {
		return nil, errors.New("chore name is required")
	}

	// Set defaults
	if req.Frequency == "" {
		req.Frequency = "weekly"
	}
	if req.Difficulty < 1 {
		req.Difficulty = 1
	}
	if req.Priority < 1 {
		req.Priority = 1
	}
	if req.AssignmentMode == "" {
		req.AssignmentMode = "round_robin"
	}

	chore := models.Chore{
		ID:                   uuid.New().String(),
		Name:                 req.Name,
		Description:          req.Description,
		Frequency:            req.Frequency,
		CustomInterval:       req.CustomInterval,
		Difficulty:           req.Difficulty,
		Priority:             req.Priority,
		AssignmentMode:       req.AssignmentMode,
		NotificationsEnabled: req.NotificationsEnabled,
		ReminderHours:        req.ReminderHours,
		IsActive:             true,
		CreatedAt:            time.Now(),
	}

	if err := s.chores.Create(ctx, &chore); err != nil {
		return nil, fmt.Errorf("failed to create chore: %w", err)
	}

	return &chore, nil
}

// GetChores retrieves all chores
func (s *ChoreService) GetChores(ctx context.Context) ([]models.Chore, error) {
	chores, err := s.chores.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	return chores, nil
}

// GetChore retrieves a chore by ID
func (s *ChoreService) GetChore(ctx context.Context, choreID string) (*models.Chore, error) {
	chore, err := s.chores.GetByID(ctx, choreID)
	if err != nil {
		return nil, errors.New("chore not found")
	}
	return chore, nil
}

// GetUserByID retrieves a user by ID
func (s *ChoreService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// AssignChore assigns a chore to a user (ADMIN only)
func (s *ChoreService) AssignChore(ctx context.Context, req AssignChoreRequest) (*models.ChoreAssignment, error) {
	// Verify chore exists and get it
	chore, err := s.GetChore(ctx, req.ChoreID)
	if err != nil {
		return nil, err
	}

	// Verify user exists
	_, err = s.users.GetByID(ctx, req.AssigneeUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Calculate base points (difficulty * 10)
	points := chore.Difficulty * 10

	assignment := models.ChoreAssignment{
		ID:             uuid.New().String(),
		ChoreID:        req.ChoreID,
		AssigneeUserID: req.AssigneeUserID,
		DueDate:        req.DueDate,
		Status:         "pending",
		Points:         points,
		IsOnTime:       false,
	}

	if err := s.choreAssignments.Create(ctx, &assignment); err != nil {
		return nil, fmt.Errorf("failed to create chore assignment: %w", err)
	}

	return &assignment, nil
}

// GetChoreAssignments retrieves all chore assignments
func (s *ChoreService) GetChoreAssignments(ctx context.Context, userID *string, status *string) ([]models.ChoreAssignment, error) {
	var assignments []models.ChoreAssignment
	var err error

	if userID != nil && status != nil {
		// Get by user then filter by status
		assignments, err = s.choreAssignments.ListByAssigneeID(ctx, *userID)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		// Filter by status
		filtered := []models.ChoreAssignment{}
		for _, a := range assignments {
			if a.Status == *status {
				filtered = append(filtered, a)
			}
		}
		assignments = filtered
	} else if userID != nil {
		assignments, err = s.choreAssignments.ListByAssigneeID(ctx, *userID)
	} else if status != nil {
		assignments, err = s.choreAssignments.ListByStatus(ctx, *status)
	} else {
		// List all - we'll use chores to get all assignments
		chores, err := s.chores.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		for _, chore := range chores {
			choreAssignments, err := s.choreAssignments.ListByChoreID(ctx, chore.ID)
			if err != nil {
				continue
			}
			assignments = append(assignments, choreAssignments...)
		}
		return assignments, nil
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return assignments, nil
}

// GetChoreAssignment retrieves a chore assignment by ID
func (s *ChoreService) GetChoreAssignment(ctx context.Context, assignmentID string) (*models.ChoreAssignment, error) {
	assignment, err := s.choreAssignments.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, errors.New("chore assignment not found")
	}
	return assignment, nil
}

// UpdateChoreAssignment updates a chore assignment status
func (s *ChoreService) UpdateChoreAssignment(ctx context.Context, assignmentID string, req UpdateChoreAssignmentRequest) error {
	validStatuses := map[string]bool{
		"pending": true, "in_progress": true, "done": true, "overdue": true,
	}
	if !validStatuses[req.Status] {
		return errors.New("invalid status")
	}

	// Get current assignment
	assignment, err := s.GetChoreAssignment(ctx, assignmentID)
	if err != nil {
		return err
	}

	assignment.Status = req.Status

	if req.Status == "done" {
		now := time.Now()
		assignment.CompletedAt = &now

		// Check if completed on time and award bonus points
		isOnTime := now.Before(assignment.DueDate) || now.Equal(assignment.DueDate)
		assignment.IsOnTime = isOnTime

		if isOnTime {
			// 50% bonus for on-time completion
			assignment.Points = int(float64(assignment.Points) * 1.5)
		}
	} else {
		assignment.CompletedAt = nil
	}

	if err := s.choreAssignments.Update(ctx, assignment); err != nil {
		return fmt.Errorf("failed to update chore assignment: %w", err)
	}

	return nil
}

// SwapChoreAssignment swaps two chore assignments (ADMIN only)
func (s *ChoreService) SwapChoreAssignment(ctx context.Context, assignment1ID, assignment2ID string) error {
	// Get both assignments
	assignment1, err := s.GetChoreAssignment(ctx, assignment1ID)
	if err != nil {
		return err
	}

	assignment2, err := s.GetChoreAssignment(ctx, assignment2ID)
	if err != nil {
		return err
	}

	// Swap assignees
	assignment1.AssigneeUserID, assignment2.AssigneeUserID = assignment2.AssigneeUserID, assignment1.AssigneeUserID

	if err := s.choreAssignments.Update(ctx, assignment1); err != nil {
		return fmt.Errorf("failed to update first assignment: %w", err)
	}

	if err := s.choreAssignments.Update(ctx, assignment2); err != nil {
		return fmt.Errorf("failed to update second assignment: %w", err)
	}

	return nil
}

// RotateChore creates a new assignment based on a rotating schedule (ADMIN only)
func (s *ChoreService) RotateChore(ctx context.Context, choreID string, dueDate time.Time) (*models.ChoreAssignment, error) {
	// Get all active users
	users, err := s.users.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if len(users) == 0 {
		return nil, errors.New("no active users to assign chore to")
	}

	// Get the last assignment for this chore
	lastAssignment, err := s.choreAssignments.GetLatestByChoreID(ctx, choreID)

	var nextUserID string

	if err != nil {
		// No previous assignment, assign to first user
		nextUserID = users[0].ID
	} else {
		// Find next user in rotation
		lastUserIndex := -1
		for i, u := range users {
			if u.ID == lastAssignment.AssigneeUserID {
				lastUserIndex = i
				break
			}
		}

		if lastUserIndex == -1 {
			// Last user no longer exists, start from beginning
			nextUserID = users[0].ID
		} else {
			// Move to next user (circular)
			nextUserIndex := (lastUserIndex + 1) % len(users)
			nextUserID = users[nextUserIndex].ID
		}
	}

	// Create new assignment
	return s.AssignChore(ctx, AssignChoreRequest{
		ChoreID:        choreID,
		AssigneeUserID: nextUserID,
		DueDate:        dueDate,
	})
}

// GetChoresWithAssignments retrieves chores with their current assignments
func (s *ChoreService) GetChoresWithAssignments(ctx context.Context) ([]ChoreWithAssignment, error) {
	chores, err := s.GetChores(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]ChoreWithAssignment, 0, len(chores))

	for _, chore := range chores {
		// Get most recent pending assignment for this chore
		pendingAssignments, err := s.choreAssignments.ListPendingByAssignee(ctx, "")
		if err != nil {
			pendingAssignments = []models.ChoreAssignment{}
		}

		choreWithAssignment := ChoreWithAssignment{
			Chore: chore,
		}

		// Find pending assignment for this chore
		for _, assignment := range pendingAssignments {
			if assignment.ChoreID == chore.ID {
				a := assignment // Create a copy to avoid pointer issues
				choreWithAssignment.Assignment = &a
				break
			}
		}

		// If no pending, try to get latest assignment
		if choreWithAssignment.Assignment == nil {
			latest, err := s.choreAssignments.GetLatestByChoreID(ctx, chore.ID)
			if err == nil && latest.Status == "pending" {
				choreWithAssignment.Assignment = latest
			}
		}

		result = append(result, choreWithAssignment)
	}

	return result, nil
}

// UserStats represents user statistics for chores
type UserStats struct {
	UserID          string  `json:"userId"`
	UserName        string  `json:"userName"`
	TotalPoints     int     `json:"totalPoints"`
	CompletedChores int     `json:"completedChores"`
	OnTimeRate      float64 `json:"onTimeRate"`
	PendingChores   int     `json:"pendingChores"`
}

// AutoAssignChore automatically assigns a chore to the user with least workload
func (s *ChoreService) AutoAssignChore(ctx context.Context, choreID string, dueDate time.Time) (*models.ChoreAssignment, error) {
	// Get all active users
	users, err := s.users.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if len(users) == 0 {
		return nil, errors.New("no active users to assign chore to")
	}

	// Calculate workload for each user (pending chores + their difficulty)
	type userWorkload struct {
		UserID   string
		Workload int // Sum of difficulty points from pending chores
		Count    int // Number of pending chores
	}

	workloads := make([]userWorkload, 0, len(users))

	for _, user := range users {
		// Get user's pending assignments
		pendingAssignments, err := s.choreAssignments.ListPendingByAssignee(ctx, user.ID)
		if err != nil {
			pendingAssignments = []models.ChoreAssignment{}
		}

		totalWorkload := 0
		for _, assignment := range pendingAssignments {
			// Get the chore to find its difficulty
			assignedChore, err := s.GetChore(ctx, assignment.ChoreID)
			if err == nil {
				totalWorkload += assignedChore.Difficulty
			}
		}

		workloads = append(workloads, userWorkload{
			UserID:   user.ID,
			Workload: totalWorkload,
			Count:    len(pendingAssignments),
		})
	}

	// Find user with minimum workload (prioritize by difficulty sum, then by count)
	minWorkload := workloads[0]
	for _, wl := range workloads[1:] {
		if wl.Workload < minWorkload.Workload || (wl.Workload == minWorkload.Workload && wl.Count < minWorkload.Count) {
			minWorkload = wl
		}
	}

	// Assign to user with minimum workload
	return s.AssignChore(ctx, AssignChoreRequest{
		ChoreID:        choreID,
		AssigneeUserID: minWorkload.UserID,
		DueDate:        dueDate,
	})
}

// GetUserLeaderboard retrieves user rankings based on points
func (s *ChoreService) GetUserLeaderboard(ctx context.Context) ([]UserStats, error) {
	// Get all active users
	users, err := s.users.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	result := make([]UserStats, 0, len(users))

	for _, user := range users {
		// Get all completed assignments for this user
		allAssignments, err := s.choreAssignments.ListByAssigneeID(ctx, user.ID)
		if err != nil {
			continue
		}

		// Filter completed
		completedAssignments := []models.ChoreAssignment{}
		pendingCount := 0
		for _, a := range allAssignments {
			if a.Status == "done" {
				completedAssignments = append(completedAssignments, a)
			} else if a.Status == "pending" {
				pendingCount++
			}
		}

		// Calculate stats
		totalPoints := 0
		onTimeCount := 0
		for _, assignment := range completedAssignments {
			totalPoints += assignment.Points
			if assignment.IsOnTime {
				onTimeCount++
			}
		}

		onTimeRate := 0.0
		if len(completedAssignments) > 0 {
			onTimeRate = float64(onTimeCount) / float64(len(completedAssignments)) * 100
		}

		result = append(result, UserStats{
			UserID:          user.ID,
			UserName:        user.Name,
			TotalPoints:     totalPoints,
			CompletedChores: len(completedAssignments),
			OnTimeRate:      onTimeRate,
			PendingChores:   pendingCount,
		})
	}

	// Sort by total points descending
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if result[j].TotalPoints < result[j+1].TotalPoints {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result, nil
}

// DeleteChore deletes a chore and all its assignments
func (s *ChoreService) DeleteChore(ctx context.Context, choreID string) error {
	// Delete all assignments for this chore
	assignments, err := s.choreAssignments.ListByChoreID(ctx, choreID)
	if err != nil {
		return fmt.Errorf("failed to list chore assignments: %w", err)
	}

	for _, assignment := range assignments {
		if err := s.choreAssignments.Delete(ctx, assignment.ID); err != nil {
			return fmt.Errorf("failed to delete chore assignment: %w", err)
		}
	}

	// Delete the chore
	if err := s.chores.Delete(ctx, choreID); err != nil {
		return fmt.Errorf("failed to delete chore: %w", err)
	}

	return nil
}
