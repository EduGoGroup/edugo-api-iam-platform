package repository

import (
	"context"
	"errors"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== Role ====================

type postgresRoleRepository struct{ db *gorm.DB }

func NewPostgresRoleRepository(db *gorm.DB) repository.RoleRepository {
	return &postgresRoleRepository{db: db}
}

func (r *postgresRoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Role, error) {
	var role entities.Role
	if err := r.db.WithContext(ctx).Where("is_active = true").First(&role, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *postgresRoleRepository) FindAll(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Role, error) {
	query := r.db.WithContext(ctx).Where("is_active = true")
	query = filters.ApplySearch(query)
	var roles []*entities.Role
	err := query.Order("name").Find(&roles).Error
	return roles, err
}

func (r *postgresRoleRepository) FindByScope(ctx context.Context, scope string, filters sharedrepo.ListFilters) ([]*entities.Role, error) {
	query := r.db.WithContext(ctx).Where("scope = ? AND is_active = true", scope)
	query = filters.ApplySearch(query)
	var roles []*entities.Role
	err := query.Order("name").Find(&roles).Error
	return roles, err
}

// ==================== Permission ====================

type postgresPermissionRepository struct{ db *gorm.DB }

func NewPostgresPermissionRepository(db *gorm.DB) repository.PermissionRepository {
	return &postgresPermissionRepository{db: db}
}

func (r *postgresPermissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Permission, error) {
	var p entities.Permission
	if err := r.db.WithContext(ctx).Where("is_active = true").First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *postgresPermissionRepository) FindAll(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, error) {
	query := r.db.WithContext(ctx).Where("is_active = true")
	query = filters.ApplySearch(query)
	var perms []*entities.Permission
	err := query.Order("name").Find(&perms).Error
	return perms, err
}

func (r *postgresPermissionRepository) FindByRole(ctx context.Context, roleID uuid.UUID) ([]*entities.Permission, error) {
	var perms []*entities.Permission
	err := r.db.WithContext(ctx).
		Joins("INNER JOIN iam.role_permissions rp ON iam.permissions.id = rp.permission_id").
		Where("rp.role_id = ? AND iam.permissions.is_active = true", roleID).
		Order("iam.permissions.name").
		Find(&perms).Error
	return perms, err
}

// ==================== UserRole ====================

type postgresUserRoleRepository struct{ db *gorm.DB }

func NewPostgresUserRoleRepository(db *gorm.DB) repository.UserRoleRepository {
	return &postgresUserRoleRepository{db: db}
}

func (r *postgresUserRoleRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error) {
	var userRoles []*entities.UserRole
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = true", userID).Order("id").Find(&userRoles).Error
	return userRoles, err
}

func (r *postgresUserRoleRepository) FindByUserInContext(ctx context.Context, userID uuid.UUID, schoolID *uuid.UUID, unitID *uuid.UUID) ([]*entities.UserRole, error) {
	query := r.db.WithContext(ctx).Where("user_id = ? AND is_active = true", userID)
	if schoolID != nil {
		query = query.Where("school_id = ?", *schoolID)
	}
	if unitID != nil {
		query = query.Where("academic_unit_id = ?", *unitID)
	}
	query = query.Order("created_at")
	var userRoles []*entities.UserRole
	err := query.Find(&userRoles).Error
	return userRoles, err
}

func (r *postgresUserRoleRepository) Grant(ctx context.Context, ur *entities.UserRole) error {
	return r.db.WithContext(ctx).Create(ur).Error
}

func (r *postgresUserRoleRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entities.UserRole{}).Where("id = ?", id).
		Updates(map[string]interface{}{"is_active": false, "updated_at": time.Now()}).Error
}

func (r *postgresUserRoleRepository) RevokeByUserAndRole(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entities.UserRole{}).
		Where("user_id = ? AND role_id = ? AND is_active = true", userID, roleID).
		Updates(map[string]interface{}{"is_active": false, "updated_at": time.Now()}).Error
}

func (r *postgresUserRoleRepository) UserHasRole(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&entities.UserRole{}).Where("user_id = ? AND role_id = ? AND is_active = true", userID, roleID)
	if schoolID != nil {
		query = query.Where("school_id = ?", *schoolID)
	}
	if unitID != nil {
		query = query.Where("academic_unit_id = ?", *unitID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *postgresUserRoleRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID, schoolID, unitID *uuid.UUID) ([]string, error) {
	query := `SELECT DISTINCT p.name FROM iam.permissions p
		INNER JOIN iam.role_permissions rp ON p.id = rp.permission_id
		INNER JOIN iam.user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ? AND ur.is_active = true AND p.is_active = true
		ORDER BY p.name`
	args := []interface{}{userID}
	if schoolID != nil {
		query += ` AND ur.school_id = ?`
		args = append(args, *schoolID)
	}
	if unitID != nil {
		query += ` AND ur.academic_unit_id = ?`
		args = append(args, *unitID)
	}
	var perms []string
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&perms).Error
	return perms, err
}

// ==================== Resource ====================

type postgresResourceRepository struct{ db *gorm.DB }

func NewPostgresResourceRepository(db *gorm.DB) repository.ResourceRepository {
	return &postgresResourceRepository{db: db}
}

func (r *postgresResourceRepository) FindAll(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Resource, error) {
	query := r.db.WithContext(ctx).Where("is_active = true")
	query = filters.ApplySearch(query)
	var resources []*entities.Resource
	err := query.Order("sort_order").Find(&resources).Error
	return resources, err
}

func (r *postgresResourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Resource, error) {
	var res entities.Resource
	if err := r.db.WithContext(ctx).Where("is_active = true").First(&res, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *postgresResourceRepository) FindMenuVisible(ctx context.Context) ([]*entities.Resource, error) {
	var resources []*entities.Resource
	err := r.db.WithContext(ctx).Where("is_menu_visible = true AND is_active = true").Order("sort_order").Find(&resources).Error
	return resources, err
}

func (r *postgresResourceRepository) Create(ctx context.Context, res *entities.Resource) error {
	return r.db.WithContext(ctx).Create(res).Error
}

func (r *postgresResourceRepository) Update(ctx context.Context, res *entities.Resource) error {
	return r.db.WithContext(ctx).Save(res).Error
}
