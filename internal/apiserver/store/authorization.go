package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"gorm.io/gorm"
)

var errBuiltinRolesCannotBeEdited = errors.New("builtin roles cannot be edited")
var errCannotRevokeLastAdministrator = errors.New("cannot revoke last administrator")
var errUnsupportedRoleMutation = errors.New("unsupported role mutation")

// AuthorizationStore queries live database grants. JWTs never carry authoritative roles.
type AuthorizationStore struct{ core IStore }

// NewAuthorizationStore constructs an authorization store with an explicit transaction source.
func NewAuthorizationStore(core IStore) *AuthorizationStore { return &AuthorizationStore{core} }

// Allowed returns false for missing, disabled or unprivileged users.
func (s *AuthorizationStore) Allowed(ctx context.Context, user, permission string) (bool, error) {
	var allowed bool
	err := s.core.DB(ctx).WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM "user" u JOIN user_role ur ON ur.user_id=u.user_id JOIN role_permission rp ON rp.role_id=ur.role_id WHERE u.user_id=? AND u.status=0 AND rp.permission_code=?)`, user, permission).Scan(&allowed).Error
	if err != nil {
		return false, fmt.Errorf("lookup permission: %w", err)
	}
	return allowed, nil
}

// Audit writes a sanitized management outcome in the caller's transaction.
func (s *AuthorizationStore) Audit(ctx context.Context, actor, action, target, outcome string, summary map[string]any) error {
	b, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("marshal audit summary: %w", err)
	}
	if err := s.core.DB(ctx).WithContext(ctx).Exec(`INSERT INTO audit_log(user_id,action,resource,details,outcome,request_id) VALUES(?,?,?,?,?,?)`, actor, action, target, string(b), outcome, contextx.RequestID(ctx)).Error; err != nil {
		return fmt.Errorf("write management audit: %w", err)
	}
	return nil
}

// AssignDefaultRole is invoked with account creation's transaction context.
func (s *AuthorizationStore) AssignDefaultRole(ctx context.Context, user string) error {
	return s.core.DB(ctx).WithContext(ctx).Exec("INSERT INTO user_role(user_id,role_id) VALUES(?,'role_user') ON CONFLICT DO NOTHING", user).Error
}

// ManageRole applies a bounded role or grant mutation. Permission catalogs are fixed by migrations.
func (s *AuthorizationStore) ManageRole(ctx context.Context, actor, action, id, target string, permissions []string) error {
	return s.core.TX(ctx, func(tx context.Context) error {
		db := s.core.DB(tx).WithContext(tx)
		if err := db.Exec("SELECT pg_advisory_xact_lock(21120261015)").Error; err != nil {
			return err
		}
		allowed, err := s.Allowed(tx, actor, "rbac.manage")
		if err != nil {
			return err
		}
		if !allowed {
			return errno.ErrPermissionDenied
		}
		if err := mutateRole(db, action, id, target, permissions); err != nil {
			return err
		}
		return s.Audit(tx, actor, "rbac."+action, id, "success", map[string]any{"target": target, "permissions": permissions})
	})
}

func mutateRole(db *gorm.DB, action, id, target string, permissions []string) error {
	switch action {
	case "create":
		return db.Exec("INSERT INTO role(role_id,name) VALUES(?,?)", id, target).Error
	case "permissions":
		return setRolePermissions(db, id, permissions)
	case "assign":
		return db.Exec("INSERT INTO user_role(user_id,role_id) VALUES(?,?) ON CONFLICT DO NOTHING", target, id).Error
	case "revoke":
		return revokeRole(db, id, target)
	default:
		return errUnsupportedRoleMutation
	}
}
func setRolePermissions(db *gorm.DB, id string, permissions []string) error {
	var builtin bool
	result := db.Raw("SELECT builtin FROM role WHERE role_id=? FOR UPDATE", id).Scan(&builtin)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	if builtin {
		return errBuiltinRolesCannotBeEdited
	}
	if err := db.Exec("DELETE FROM role_permission WHERE role_id=?", id).Error; err != nil {
		return err
	}
	for _, p := range permissions {
		if err := db.Exec("INSERT INTO role_permission(role_id,permission_code) VALUES(?,?)", id, p).Error; err != nil {
			return err
		}
	}
	return nil
}
func revokeRole(db *gorm.DB, id, target string) error {
	if id == "role_admin" {
		var n int64
		if err := db.Table("user_role ur").Joins(`JOIN "user" u ON u.user_id=ur.user_id`).Where("role_id='role_admin' AND u.status=0").Count(&n).Error; err != nil {
			return err
		}
		if n <= 1 {
			return errCannotRevokeLastAdministrator
		}
	}
	return db.Exec("DELETE FROM user_role WHERE user_id=? AND role_id=?", target, id).Error
}
