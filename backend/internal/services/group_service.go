package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type GroupService struct {
	groups      repository.GroupRepository
	users       repository.UserRepository
	allocations repository.AllocationRepository
}

func NewGroupService(groups repository.GroupRepository, users repository.UserRepository, allocations repository.AllocationRepository) *GroupService {
	return &GroupService{
		groups:      groups,
		users:       users,
		allocations: allocations,
	}
}

type CreateGroupRequest struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type UpdateGroupRequest struct {
	Name   *string  `json:"name,omitempty"`
	Weight *float64 `json:"weight,omitempty"`
}

// CreateGroup creates a new household group (ADMIN only)
func (s *GroupService) CreateGroup(ctx context.Context, req CreateGroupRequest) (*models.Group, error) {
	if req.Name == "" {
		return nil, errors.New("group name is required")
	}

	weight := req.Weight
	if weight == 0 {
		weight = 1.0 // Default weight
	}

	group := models.Group{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Weight:    weight,
		CreatedAt: time.Now(),
	}

	if err := s.groups.Create(ctx, &group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return &group, nil
}

// GetGroups retrieves all groups
func (s *GroupService) GetGroups(ctx context.Context) ([]models.Group, error) {
	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	return groups, nil
}

// GetGroup retrieves a group by ID
func (s *GroupService) GetGroup(ctx context.Context, groupID string) (*models.Group, error) {
	group, err := s.groups.GetByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	return group, nil
}

// UpdateGroup updates a group (ADMIN only)
func (s *GroupService) UpdateGroup(ctx context.Context, groupID string, req UpdateGroupRequest) error {
	// Get the existing group first
	group, err := s.groups.GetByID(ctx, groupID)
	if err != nil {
		return errors.New("group not found")
	}

	if req.Name != nil {
		if *req.Name == "" {
			return errors.New("group name cannot be empty")
		}
		group.Name = *req.Name
	}

	if req.Weight != nil {
		if *req.Weight <= 0 {
			return errors.New("weight must be positive")
		}
		group.Weight = *req.Weight
	}

	if err := s.groups.Update(ctx, group); err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// DeleteGroup deletes a group (ADMIN only)
// Note: Should check if any users are still assigned to this group
func (s *GroupService) DeleteGroup(ctx context.Context, groupID string) error {
	// Check if any users are in this group
	users, err := s.users.ListByGroupID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if len(users) > 0 {
		return errors.New("cannot delete group: users are still assigned to it")
	}

	// Delete all allocations for this group
	if err := s.allocations.DeleteByBillID(ctx, groupID); err != nil {
		log.Printf("[WARN] Failed to delete allocations for group %s: %v", groupID, err)
	}

	if err := s.groups.Delete(ctx, groupID); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	log.Printf("[INFO] Deleted group %s and its allocations", groupID)
	return nil
}
