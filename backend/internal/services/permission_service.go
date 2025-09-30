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

type PermissionService struct {
	db *mongo.Database
}

func NewPermissionService(db *mongo.Database) *PermissionService {
	return &PermissionService{db: db}
}

// InitializeDefaultPermissions creates the default permission set
func (s *PermissionService) InitializeDefaultPermissions(ctx context.Context) error {
	permissions := []models.Permission{
		// User management
		{ID: primitive.NewObjectID(), Name: "users.create", Description: "Twórz nowych użytkowników", Category: "users"},
		{ID: primitive.NewObjectID(), Name: "users.read", Description: "Przeglądaj informacje o użytkownikach", Category: "users"},
		{ID: primitive.NewObjectID(), Name: "users.update", Description: "Aktualizuj informacje o użytkownikach", Category: "users"},
		{ID: primitive.NewObjectID(), Name: "users.delete", Description: "Usuń użytkowników", Category: "users"},

		// Group management
		{ID: primitive.NewObjectID(), Name: "groups.create", Description: "Twórz nowe grupy", Category: "groups"},
		{ID: primitive.NewObjectID(), Name: "groups.read", Description: "Przeglądaj grupy", Category: "groups"},
		{ID: primitive.NewObjectID(), Name: "groups.update", Description: "Aktualizuj grupy", Category: "groups"},
		{ID: primitive.NewObjectID(), Name: "groups.delete", Description: "Usuń grupy", Category: "groups"},

		// Bill management
		{ID: primitive.NewObjectID(), Name: "bills.create", Description: "Twórz nowe rachunki", Category: "bills"},
		{ID: primitive.NewObjectID(), Name: "bills.read", Description: "Przeglądaj rachunki", Category: "bills"},
		{ID: primitive.NewObjectID(), Name: "bills.update", Description: "Aktualizuj rachunki", Category: "bills"},
		{ID: primitive.NewObjectID(), Name: "bills.delete", Description: "Usuń rachunki", Category: "bills"},
		{ID: primitive.NewObjectID(), Name: "bills.post", Description: "Opublikuj rachunki", Category: "bills"},
		{ID: primitive.NewObjectID(), Name: "bills.close", Description: "Zamknij rachunki", Category: "bills"},

		// Chore management
		{ID: primitive.NewObjectID(), Name: "chores.create", Description: "Twórz nowe obowiązki", Category: "chores"},
		{ID: primitive.NewObjectID(), Name: "chores.read", Description: "Przeglądaj obowiązki", Category: "chores"},
		{ID: primitive.NewObjectID(), Name: "chores.update", Description: "Aktualizuj obowiązki", Category: "chores"},
		{ID: primitive.NewObjectID(), Name: "chores.delete", Description: "Usuń obowiązki", Category: "chores"},
		{ID: primitive.NewObjectID(), Name: "chores.assign", Description: "Przypisz obowiązki do użytkowników", Category: "chores"},

		// Supplies management
		{ID: primitive.NewObjectID(), Name: "supplies.create", Description: "Dodaj artykuły zaopatrzeniowe", Category: "supplies"},
		{ID: primitive.NewObjectID(), Name: "supplies.read", Description: "Przeglądaj zaopatrzenie", Category: "supplies"},
		{ID: primitive.NewObjectID(), Name: "supplies.update", Description: "Aktualizuj artykuły zaopatrzeniowe", Category: "supplies"},
		{ID: primitive.NewObjectID(), Name: "supplies.delete", Description: "Usuń artykuły zaopatrzeniowe", Category: "supplies"},

		// Role management
		{ID: primitive.NewObjectID(), Name: "roles.create", Description: "Twórz niestandardowe role", Category: "roles"},
		{ID: primitive.NewObjectID(), Name: "roles.read", Description: "Przeglądaj role", Category: "roles"},
		{ID: primitive.NewObjectID(), Name: "roles.update", Description: "Aktualizuj role", Category: "roles"},
		{ID: primitive.NewObjectID(), Name: "roles.delete", Description: "Usuń role", Category: "roles"},

		// Approval management
		{ID: primitive.NewObjectID(), Name: "approvals.review", Description: "Przeglądaj i zatwierdź/odrzuć oczekujące akcje", Category: "approvals"},

		// Audit logs
		{ID: primitive.NewObjectID(), Name: "audit.read", Description: "Przeglądaj logi audytu", Category: "audit"},

		// Loan management
		{ID: primitive.NewObjectID(), Name: "loans.read", Description: "Przeglądaj pożyczki", Category: "loans"},
		{ID: primitive.NewObjectID(), Name: "loans.delete", Description: "Usuń pożyczki", Category: "loans"},

		// Reading management
		{ID: primitive.NewObjectID(), Name: "readings.delete", Description: "Usuń odczyty liczników", Category: "readings"},

		// Backup management
		{ID: primitive.NewObjectID(), Name: "backup.export", Description: "Eksportuj kopię zapasową", Category: "backup"},
		{ID: primitive.NewObjectID(), Name: "backup.import", Description: "Importuj kopię zapasową", Category: "backup"},
	}

	// Insert or update permissions (upsert missing ones)
	for _, perm := range permissions {
		filter := bson.M{"name": perm.Name}
		update := bson.M{
			"$setOnInsert": perm,
		}
		opts := options.Update().SetUpsert(true)
		_, err := s.db.Collection("permissions").UpdateOne(ctx, filter, update, opts)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAllPermissions retrieves all permissions
func (s *PermissionService) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	cursor, err := s.db.Collection("permissions").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []models.Permission
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetPermissionsByCategory retrieves permissions grouped by category
func (s *PermissionService) GetPermissionsByCategory(ctx context.Context) (map[string][]models.Permission, error) {
	permissions, err := s.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]models.Permission)
	for _, perm := range permissions {
		grouped[perm.Category] = append(grouped[perm.Category], perm)
	}
	return grouped, nil
}

type RoleService struct {
	db *mongo.Database
}

func NewRoleService(db *mongo.Database) *RoleService {
	return &RoleService{db: db}
}

// InitializeDefaultRoles creates the default ADMIN and RESIDENT roles
func (s *RoleService) InitializeDefaultRoles(ctx context.Context) error {
	now := time.Now()

	// ADMIN role with all permissions
	adminPermissions := []string{
		"users.create", "users.read", "users.update", "users.delete",
		"groups.create", "groups.read", "groups.update", "groups.delete",
		"bills.create", "bills.read", "bills.update", "bills.delete", "bills.post", "bills.close",
		"chores.create", "chores.read", "chores.update", "chores.delete", "chores.assign",
		"supplies.create", "supplies.read", "supplies.update", "supplies.delete",
		"roles.create", "roles.read", "roles.update", "roles.delete",
		"approvals.review",
		"audit.read",
		"loans.read", "loans.delete",
		"readings.delete",
		"backup.export", "backup.import",
	}

	// MIESZKANIEC role with limited permissions
	residentPermissions := []string{
		"users.read",
		"groups.read",
		"bills.read",
		"chores.read",
		"supplies.read", "supplies.update",
	}

	// Upsert ADMIN role - always update permissions to include new ones
	adminFilter := bson.M{"name": "ADMIN"}
	adminUpdate := bson.M{
		"$setOnInsert": bson.M{
			"_id":         primitive.NewObjectID(),
			"name":        "ADMIN",
			"displayName": "Administrator",
			"isSystem":    true,
			"createdAt":   now,
		},
		"$set": bson.M{
			"permissions": adminPermissions,
			"updatedAt":   now,
		},
	}
	adminOpts := options.Update().SetUpsert(true)
	_, err := s.db.Collection("roles").UpdateOne(ctx, adminFilter, adminUpdate, adminOpts)
	if err != nil {
		return err
	}

	// Upsert MIESZKANIEC role - only create once, never update
	residentFilter := bson.M{"name": "MIESZKANIEC"}
	residentUpdate := bson.M{
		"$setOnInsert": bson.M{
			"_id":         primitive.NewObjectID(),
			"name":        "MIESZKANIEC",
			"displayName": "Mieszkaniec",
			"isSystem":    true,
			"createdAt":   now,
			"updatedAt":   now,
			"permissions": residentPermissions,
		},
	}
	residentOpts := options.Update().SetUpsert(true)
	_, err = s.db.Collection("roles").UpdateOne(ctx, residentFilter, residentUpdate, residentOpts)
	return err
}

// GetRole retrieves a role by name
func (s *RoleService) GetRole(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := s.db.Collection("roles").FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// GetRoleByID retrieves a role by ID
func (s *RoleService) GetRoleByID(ctx context.Context, id primitive.ObjectID) (*models.Role, error) {
	var role models.Role
	err := s.db.Collection("roles").FindOne(ctx, bson.M{"_id": id}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// GetAllRoles retrieves all roles
func (s *RoleService) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	cursor, err := s.db.Collection("roles").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []models.Role
	if err := cursor.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// CreateRole creates a new custom role
func (s *RoleService) CreateRole(ctx context.Context, name, displayName string, permissions []string) (*models.Role, error) {
	// Check if role already exists
	count, err := s.db.Collection("roles").CountDocuments(ctx, bson.M{"name": name})
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("role with this name already exists")
	}

	now := time.Now()
	role := models.Role{
		ID:          primitive.NewObjectID(),
		Name:        name,
		DisplayName: displayName,
		IsSystem:    false,
		Permissions: permissions,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = s.db.Collection("roles").InsertOne(ctx, role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// UpdateRole updates a role's permissions
func (s *RoleService) UpdateRole(ctx context.Context, roleID primitive.ObjectID, displayName string, permissions []string) error {
	// Check if role exists and is ADMIN role
	var role models.Role
	err := s.db.Collection("roles").FindOne(ctx, bson.M{"_id": roleID}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("role not found")
		}
		return err
	}

	// Only ADMIN role is protected from modification
	if role.Name == "ADMIN" {
		return errors.New("cannot modify ADMIN role")
	}

	update := bson.M{
		"$set": bson.M{
			"permissions": permissions,
			"display_name": displayName,
			"updated_at":  time.Now(),
		},
	}

	_, err = s.db.Collection("roles").UpdateOne(ctx, bson.M{"_id": roleID}, update)
	return err
}

// DeleteRole deletes a custom role
func (s *RoleService) DeleteRole(ctx context.Context, roleID primitive.ObjectID) error {
	// Check if role exists
	var role models.Role
	err := s.db.Collection("roles").FindOne(ctx, bson.M{"_id": roleID}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("role not found")
		}
		return err
	}

	// Only ADMIN role is protected from deletion
	if role.Name == "ADMIN" {
		return errors.New("cannot delete ADMIN role")
	}

	// Check if any users have this role
	userCount, err := s.db.Collection("users").CountDocuments(ctx, bson.M{"role": role.Name})
	if err != nil {
		return err
	}
	if userCount > 0 {
		return fmt.Errorf("cannot delete role: %d users are assigned to this role", userCount)
	}

	_, err = s.db.Collection("roles").DeleteOne(ctx, bson.M{"_id": roleID})
	return err
}

// HasPermission checks if a role has a specific permission
func (s *RoleService) HasPermission(ctx context.Context, roleName, permission string) (bool, error) {
	role, err := s.GetRole(ctx, roleName)
	if err != nil {
		return false, err
	}

	for _, perm := range role.Permissions {
		if perm == permission {
			return true, nil
		}
	}
	return false, nil
}

// GetRolePermissions returns all permissions for a given role
func (s *RoleService) GetRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	role, err := s.GetRole(ctx, roleName)
	if err != nil {
		return nil, err
	}
	return role.Permissions, nil
}
