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

type PermissionService struct {
	permissions repository.PermissionRepository
}

func NewPermissionService(permissions repository.PermissionRepository) *PermissionService {
	return &PermissionService{permissions: permissions}
}

// InitializeDefaultPermissions creates the default permission set
func (s *PermissionService) InitializeDefaultPermissions(ctx context.Context) error {
	permissions := []models.Permission{
		// User management
		{ID: uuid.New().String(), Name: "users.create", Description: "Twórz nowych użytkowników", Category: "users"},
		{ID: uuid.New().String(), Name: "users.read", Description: "Przeglądaj informacje o użytkownikach", Category: "users"},
		{ID: uuid.New().String(), Name: "users.update", Description: "Aktualizuj informacje o użytkownikach", Category: "users"},
		{ID: uuid.New().String(), Name: "users.delete", Description: "Usuń użytkowników", Category: "users"},

		// Group management
		{ID: uuid.New().String(), Name: "groups.create", Description: "Twórz nowe grupy", Category: "groups"},
		{ID: uuid.New().String(), Name: "groups.read", Description: "Przeglądaj grupy", Category: "groups"},
		{ID: uuid.New().String(), Name: "groups.update", Description: "Aktualizuj grupy", Category: "groups"},
		{ID: uuid.New().String(), Name: "groups.delete", Description: "Usuń grupy", Category: "groups"},

		// Bill management
		{ID: uuid.New().String(), Name: "bills.create", Description: "Twórz nowe rachunki", Category: "bills"},
		{ID: uuid.New().String(), Name: "bills.read", Description: "Przeglądaj rachunki", Category: "bills"},
		{ID: uuid.New().String(), Name: "bills.update", Description: "Aktualizuj rachunki", Category: "bills"},
		{ID: uuid.New().String(), Name: "bills.delete", Description: "Usuń rachunki", Category: "bills"},
		{ID: uuid.New().String(), Name: "bills.post", Description: "Opublikuj rachunki", Category: "bills"},
		{ID: uuid.New().String(), Name: "bills.close", Description: "Zamknij rachunki", Category: "bills"},

		// Chore management
		{ID: uuid.New().String(), Name: "chores.create", Description: "Twórz nowe obowiązki", Category: "chores"},
		{ID: uuid.New().String(), Name: "chores.read", Description: "Przeglądaj obowiązki", Category: "chores"},
		{ID: uuid.New().String(), Name: "chores.update", Description: "Aktualizuj obowiązki", Category: "chores"},
		{ID: uuid.New().String(), Name: "chores.delete", Description: "Usuń obowiązki", Category: "chores"},
		{ID: uuid.New().String(), Name: "chores.assign", Description: "Przypisz obowiązki do użytkowników", Category: "chores"},

		// Supplies management
		{ID: uuid.New().String(), Name: "supplies.create", Description: "Dodaj artykuły zaopatrzeniowe", Category: "supplies"},
		{ID: uuid.New().String(), Name: "supplies.read", Description: "Przeglądaj zaopatrzenie", Category: "supplies"},
		{ID: uuid.New().String(), Name: "supplies.update", Description: "Aktualizuj artykuły zaopatrzeniowe", Category: "supplies"},
		{ID: uuid.New().String(), Name: "supplies.delete", Description: "Usuń artykuły zaopatrzeniowe", Category: "supplies"},

		// Role management
		{ID: uuid.New().String(), Name: "roles.create", Description: "Twórz niestandardowe role", Category: "roles"},
		{ID: uuid.New().String(), Name: "roles.read", Description: "Przeglądaj role", Category: "roles"},
		{ID: uuid.New().String(), Name: "roles.update", Description: "Aktualizuj role", Category: "roles"},
		{ID: uuid.New().String(), Name: "roles.delete", Description: "Usuń role", Category: "roles"},

		// Approval management
		{ID: uuid.New().String(), Name: "approvals.review", Description: "Przeglądaj i zatwierdź/odrzuć oczekujące akcje", Category: "approvals"},

		// Audit logs
		{ID: uuid.New().String(), Name: "audit.read", Description: "Przeglądaj logi audytu", Category: "audit"},

		// Loan management
		{ID: uuid.New().String(), Name: "loans.create", Description: "Twórz pożyczki", Category: "loans"},
		{ID: uuid.New().String(), Name: "loans.read", Description: "Przeglądaj pożyczki", Category: "loans"},
		{ID: uuid.New().String(), Name: "loans.update", Description: "Edytuj pożyczki", Category: "loans"},
		{ID: uuid.New().String(), Name: "loans.delete", Description: "Usuń pożyczki", Category: "loans"},
		{ID: uuid.New().String(), Name: "loan-payments.create", Description: "Dodaj spłaty pożyczek", Category: "loans"},
		{ID: uuid.New().String(), Name: "loan-payments.read", Description: "Przeglądaj spłaty pożyczek", Category: "loans"},
		{ID: uuid.New().String(), Name: "loan-payments.update", Description: "Edytuj spłaty pożyczek", Category: "loans"},
		{ID: uuid.New().String(), Name: "loan-payments.delete", Description: "Usuń spłaty pożyczek", Category: "loans"},

		// Reading management
		{ID: uuid.New().String(), Name: "readings.delete", Description: "Usuń odczyty liczników", Category: "readings"},

		// Backup management
		{ID: uuid.New().String(), Name: "backup.export", Description: "Eksportuj kopię zapasową", Category: "backup"},
		{ID: uuid.New().String(), Name: "backup.import", Description: "Importuj kopię zapasową", Category: "backup"},

		// App settings
		{ID: uuid.New().String(), Name: "settings.app.update", Description: "Zmień ustawienia aplikacji", Category: "settings"},

		// Reminders
		{ID: uuid.New().String(), Name: "reminders.send", Description: "Wysyłaj przypomnienia użytkownikom", Category: "reminders"},
	}

	// Insert permissions (skip if already exists)
	for _, perm := range permissions {
		// Check if permission already exists
		existing, _ := s.permissions.GetByName(ctx, perm.Name)
		if existing != nil {
			continue // Already exists, skip
		}
		if err := s.permissions.Create(ctx, &perm); err != nil {
			return err
		}
	}
	return nil
}

// GetAllPermissions retrieves all permissions
func (s *PermissionService) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	return s.permissions.List(ctx)
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
	roles repository.RoleRepository
	users repository.UserRepository
}

func NewRoleService(roles repository.RoleRepository, users repository.UserRepository) *RoleService {
	return &RoleService{roles: roles, users: users}
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
		"loans.create", "loans.read", "loans.update", "loans.delete",
		"loan-payments.create", "loan-payments.read", "loan-payments.update", "loan-payments.delete",
		"readings.delete",
		"backup.export", "backup.import",
		"settings.app.update",
		"reminders.send",
	}

	// MIESZKANIEC role with limited permissions
	residentPermissions := []string{
		"users.read",
		"groups.read",
		"bills.create", "bills.read", "bills.update", "bills.delete", "bills.post", "bills.close",
		"chores.read",
		"supplies.read", "supplies.update",
		"loans.read",
		"loan-payments.read",
	}

	// Upsert ADMIN role
	adminRole, _ := s.roles.GetByName(ctx, "ADMIN")
	if adminRole == nil {
		adminRole = &models.Role{
			ID:          uuid.New().String(),
			Name:        "ADMIN",
			DisplayName: "Administrator",
			IsSystem:    true,
			Permissions: adminPermissions,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.roles.Create(ctx, adminRole); err != nil {
			return err
		}
	} else {
		// Update permissions
		adminRole.Permissions = adminPermissions
		adminRole.UpdatedAt = now
		if err := s.roles.Update(ctx, adminRole); err != nil {
			return err
		}
	}

	// Upsert MIESZKANIEC role
	residentRole, _ := s.roles.GetByName(ctx, "MIESZKANIEC")
	if residentRole == nil {
		residentRole = &models.Role{
			ID:          uuid.New().String(),
			Name:        "MIESZKANIEC",
			DisplayName: "Mieszkaniec",
			IsSystem:    true,
			Permissions: residentPermissions,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.roles.Create(ctx, residentRole); err != nil {
			return err
		}
	} else {
		// Update permissions
		residentRole.Permissions = residentPermissions
		residentRole.UpdatedAt = now
		if err := s.roles.Update(ctx, residentRole); err != nil {
			return err
		}
	}

	return nil
}

// GetRole retrieves a role by name
func (s *RoleService) GetRole(ctx context.Context, name string) (*models.Role, error) {
	role, err := s.roles.GetByName(ctx, name)
	if err != nil {
		return nil, errors.New("role not found")
	}
	return role, nil
}

// GetRoleByID retrieves a role by ID
func (s *RoleService) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	role, err := s.roles.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("role not found")
	}
	return role, nil
}

// GetAllRoles retrieves all roles
func (s *RoleService) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	return s.roles.List(ctx)
}

// CreateRole creates a new custom role
func (s *RoleService) CreateRole(ctx context.Context, name, displayName string, permissions []string) (*models.Role, error) {
	// Check if role already exists
	existing, _ := s.roles.GetByName(ctx, name)
	if existing != nil {
		return nil, errors.New("role with this name already exists")
	}

	now := time.Now()
	role := &models.Role{
		ID:          uuid.New().String(),
		Name:        name,
		DisplayName: displayName,
		IsSystem:    false,
		Permissions: permissions,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.roles.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole updates a role's permissions
func (s *RoleService) UpdateRole(ctx context.Context, roleID string, displayName string, permissions []string) error {
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return errors.New("role not found")
	}

	// Only ADMIN role is protected from modification
	if role.Name == "ADMIN" {
		return errors.New("cannot modify ADMIN role")
	}

	role.DisplayName = displayName
	role.Permissions = permissions
	role.UpdatedAt = time.Now()

	return s.roles.Update(ctx, role)
}

// DeleteRole deletes a custom role
func (s *RoleService) DeleteRole(ctx context.Context, roleID string) error {
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return errors.New("role not found")
	}

	// Only ADMIN role is protected from deletion
	if role.Name == "ADMIN" {
		return errors.New("cannot delete ADMIN role")
	}

	// Check if any users have this role
	users, err := s.users.List(ctx)
	if err != nil {
		return err
	}

	userCount := 0
	for _, u := range users {
		if u.Role == role.Name {
			userCount++
		}
	}

	if userCount > 0 {
		return fmt.Errorf("cannot delete role: %d users are assigned to this role", userCount)
	}

	return s.roles.Delete(ctx, roleID)
}

// HasPermission checks if a role has a specific permission
func (s *RoleService) HasPermission(ctx context.Context, roleName, permission string) (bool, error) {
	// ADMIN role always has all permissions
	if roleName == "ADMIN" {
		return true, nil
	}

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
