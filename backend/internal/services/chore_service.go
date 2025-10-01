package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sainaif/holy-home/internal/models"
)

type ChoreService struct {
	db *mongo.Database
}

func NewChoreService(db *mongo.Database) *ChoreService {
	return &ChoreService{db: db}
}

type CreateChoreRequest struct {
	Name                 string  `json:"name"`
	Description          *string `json:"description,omitempty"`
	Frequency            string  `json:"frequency"` // daily, weekly, monthly, custom, irregular
	CustomInterval       *int    `json:"customInterval,omitempty"`
	Difficulty           int     `json:"difficulty"` // 1-5
	Priority             int     `json:"priority"` // 1-5
	AssignmentMode       string  `json:"assignmentMode"` // manual, round_robin, random
	NotificationsEnabled bool    `json:"notificationsEnabled"`
	ReminderHours        *int    `json:"reminderHours,omitempty"`
}

type AssignChoreRequest struct {
	ChoreID        primitive.ObjectID `json:"choreId"`
	AssigneeUserID primitive.ObjectID `json:"assigneeUserId"`
	DueDate        time.Time          `json:"dueDate"`
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
		ID:                   primitive.NewObjectID(),
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

	_, err := s.db.Collection("chores").InsertOne(ctx, chore)
	if err != nil {
		return nil, fmt.Errorf("failed to create chore: %w", err)
	}

	return &chore, nil
}

// GetChores retrieves all chores
func (s *ChoreService) GetChores(ctx context.Context) ([]models.Chore, error) {
	cursor, err := s.db.Collection("chores").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var chores []models.Chore
	if err := cursor.All(ctx, &chores); err != nil {
		return nil, fmt.Errorf("failed to decode chores: %w", err)
	}

	return chores, nil
}

// GetChore retrieves a chore by ID
func (s *ChoreService) GetChore(ctx context.Context, choreID primitive.ObjectID) (*models.Chore, error) {
	var chore models.Chore
	err := s.db.Collection("chores").FindOne(ctx, bson.M{"_id": choreID}).Decode(&chore)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("chore not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &chore, nil
}

// AssignChore assigns a chore to a user (ADMIN only)
func (s *ChoreService) AssignChore(ctx context.Context, req AssignChoreRequest) (*models.ChoreAssignment, error) {
	// Verify chore exists and get it
	chore, err := s.GetChore(ctx, req.ChoreID)
	if err != nil {
		return nil, err
	}

	// Verify user exists
	var user models.User
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": req.AssigneeUserID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Calculate base points (difficulty * 10)
	points := chore.Difficulty * 10

	assignment := models.ChoreAssignment{
		ID:             primitive.NewObjectID(),
		ChoreID:        req.ChoreID,
		AssigneeUserID: req.AssigneeUserID,
		DueDate:        req.DueDate,
		Status:         "pending",
		Points:         points,
		IsOnTime:       false,
	}

	_, err = s.db.Collection("chore_assignments").InsertOne(ctx, assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to create chore assignment: %w", err)
	}

	return &assignment, nil
}

// GetChoreAssignments retrieves all chore assignments
func (s *ChoreService) GetChoreAssignments(ctx context.Context, userID *primitive.ObjectID, status *string) ([]models.ChoreAssignment, error) {
	filter := bson.M{}

	if userID != nil {
		filter["assignee_user_id"] = *userID
	}

	if status != nil {
		filter["status"] = *status
	}

	cursor, err := s.db.Collection("chore_assignments").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var assignments []models.ChoreAssignment
	if err := cursor.All(ctx, &assignments); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	return assignments, nil
}

// GetChoreAssignment retrieves a chore assignment by ID
func (s *ChoreService) GetChoreAssignment(ctx context.Context, assignmentID primitive.ObjectID) (*models.ChoreAssignment, error) {
	var assignment models.ChoreAssignment
	err := s.db.Collection("chore_assignments").FindOne(ctx, bson.M{"_id": assignmentID}).Decode(&assignment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("chore assignment not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &assignment, nil
}

// UpdateChoreAssignment updates a chore assignment status
func (s *ChoreService) UpdateChoreAssignment(ctx context.Context, assignmentID primitive.ObjectID, req UpdateChoreAssignmentRequest) error {
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

	update := bson.M{"status": req.Status}

	if req.Status == "done" {
		now := time.Now()
		update["completed_at"] = now

		// Check if completed on time and award bonus points
		isOnTime := now.Before(assignment.DueDate) || now.Equal(assignment.DueDate)
		update["is_on_time"] = isOnTime

		if isOnTime {
			// 50% bonus for on-time completion
			update["points"] = int(float64(assignment.Points) * 1.5)
		}
	} else {
		update["completed_at"] = nil
	}

	result, err := s.db.Collection("chore_assignments").UpdateOne(
		ctx,
		bson.M{"_id": assignmentID},
		bson.M{"$set": update},
	)
	if err != nil {
		return fmt.Errorf("failed to update chore assignment: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("chore assignment not found")
	}

	return nil
}

// SwapChoreAssignment swaps two chore assignments (ADMIN only)
func (s *ChoreService) SwapChoreAssignment(ctx context.Context, assignment1ID, assignment2ID primitive.ObjectID) error {
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
	_, err = s.db.Collection("chore_assignments").UpdateOne(
		ctx,
		bson.M{"_id": assignment1ID},
		bson.M{"$set": bson.M{"assignee_user_id": assignment2.AssigneeUserID}},
	)
	if err != nil {
		return fmt.Errorf("failed to update first assignment: %w", err)
	}

	_, err = s.db.Collection("chore_assignments").UpdateOne(
		ctx,
		bson.M{"_id": assignment2ID},
		bson.M{"$set": bson.M{"assignee_user_id": assignment1.AssigneeUserID}},
	)
	if err != nil {
		return fmt.Errorf("failed to update second assignment: %w", err)
	}

	return nil
}

// RotateChore creates a new assignment based on a rotating schedule (ADMIN only)
func (s *ChoreService) RotateChore(ctx context.Context, choreID primitive.ObjectID, dueDate time.Time) (*models.ChoreAssignment, error) {
	// Get all users
	cursor, err := s.db.Collection("users").Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	if len(users) == 0 {
		return nil, errors.New("no active users to assign chore to")
	}

	// Get the last assignment for this chore
	var lastAssignment models.ChoreAssignment
	opts := options.FindOne().SetSort(bson.M{"due_date": -1})
	err = s.db.Collection("chore_assignments").
		FindOne(ctx, bson.M{"chore_id": choreID}, opts).
		Decode(&lastAssignment)

	var nextUserID primitive.ObjectID

	if err == mongo.ErrNoDocuments {
		// No previous assignment, assign to first user
		nextUserID = users[0].ID
	} else if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
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
		var assignment models.ChoreAssignment
		opts := options.FindOne().SetSort(bson.M{"due_date": -1})
		err := s.db.Collection("chore_assignments").
			FindOne(ctx, bson.M{
				"chore_id": chore.ID,
				"status":   "pending",
			}, opts).
			Decode(&assignment)

		choreWithAssignment := ChoreWithAssignment{
			Chore: chore,
		}

		if err == nil {
			choreWithAssignment.Assignment = &assignment
		}

		result = append(result, choreWithAssignment)
	}

	return result, nil
}
// UserStats represents user statistics for chores
type UserStats struct {
	UserID           primitive.ObjectID `json:"userId"`
	UserName         string             `json:"userName"`
	TotalPoints      int                `json:"totalPoints"`
	CompletedChores  int                `json:"completedChores"`
	OnTimeRate       float64            `json:"onTimeRate"`
	PendingChores    int                `json:"pendingChores"`
}

// AutoAssignChore automatically assigns a chore to the user with least workload
func (s *ChoreService) AutoAssignChore(ctx context.Context, choreID primitive.ObjectID, dueDate time.Time) (*models.ChoreAssignment, error) {
	// Get all active users
	usersCursor, err := s.db.Collection("users").Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer usersCursor.Close(ctx)

	var users []models.User
	if err := usersCursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	if len(users) == 0 {
		return nil, errors.New("no active users to assign chore to")
	}

	// Calculate workload for each user (pending chores + their difficulty)
	type userWorkload struct {
		UserID   primitive.ObjectID
		Workload int // Sum of difficulty points from pending chores
		Count    int // Number of pending chores
	}

	workloads := make([]userWorkload, 0, len(users))

	for _, user := range users {
		// Get user's pending assignments
		pendingAssignments, err := s.GetChoreAssignments(ctx, &user.ID, stringPtr("pending"))
		if err != nil {
			return nil, err
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
	// Get all users
	usersCursor, err := s.db.Collection("users").Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer usersCursor.Close(ctx)

	var users []models.User
	if err := usersCursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	result := make([]UserStats, 0, len(users))

	for _, user := range users {
		// Get all completed assignments for this user
		completedAssignments, err := s.GetChoreAssignments(ctx, &user.ID, stringPtr("done"))
		if err != nil {
			return nil, err
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

		// Get pending assignments
		pendingAssignments, err := s.GetChoreAssignments(ctx, &user.ID, stringPtr("pending"))
		if err != nil {
			return nil, err
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
			PendingChores:   len(pendingAssignments),
		})
	}

	// Sort by total points descending
	// Simple bubble sort for now
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if result[j].TotalPoints < result[j+1].TotalPoints {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result, nil
}

func stringPtr(s string) *string {
	return &s
}
