package model

// Well-known role names seeded by the migrations.
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// Role is a named permission group.
type Role struct {
	Base
	Name        string `gorm:"size:64;uniqueIndex;not null" json:"name"`
	Description string `gorm:"size:255" json:"description"`
}

// User is an account managed by the identity service.
type User struct {
	Base
	Email        string `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	Name         string `gorm:"size:255" json:"name"`
	IsActive     bool   `gorm:"not null;default:true" json:"is_active"`
	Roles        []Role `gorm:"many2many:user_roles;constraint:OnDelete:CASCADE" json:"roles,omitempty"`
}

// RoleNames flattens the loaded roles into their names.
func (u *User) RoleNames() []string {
	names := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		names = append(names, r.Name)
	}
	return names
}
