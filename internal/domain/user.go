package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleOwner      UserRole = "owner"
	RoleManager    UserRole = "manager"
	RoleCashier    UserRole = "cashier"
	RoleStaff      UserRole = "staff"
	RoleViewer     UserRole = "viewer"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Email       string    `json:"email"`
	FullName    string    `json:"full_name"`
	Role        UserRole  `json:"role"`
	AvatarURL   string    `json:"avatar_url"`
	Phone       string    `json:"phone"`
	IsActive    bool      `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (u User) DisplayName() string {
	if u.FullName != "" {
		return u.FullName
	}
	return u.Email
}

func (u User) RoleLabel() string {
	switch u.Role {
	case RoleSuperAdmin:
		return "Super Admin"
	case RoleOwner:
		return "Pemilik"
	case RoleManager:
		return "Manager"
	case RoleCashier:
		return "Kasir"
	case RoleStaff:
		return "Staf"
	case RoleViewer:
		return "Viewer"
	default:
		return string(u.Role)
	}
}

func (u User) AvatarOrDefault() string {
	if u.AvatarURL != "" {
		return u.AvatarURL
	}
	return "/static/img/avatar-default.svg"
}

func (u User) CanAccessPOS() bool {
	switch u.Role {
	case RoleSuperAdmin, RoleOwner, RoleManager, RoleCashier:
		return true
	default:
		return false
	}
}

func (u User) CanManageProducts() bool {
	switch u.Role {
	case RoleSuperAdmin, RoleOwner, RoleManager:
		return true
	default:
		return false
	}
}
