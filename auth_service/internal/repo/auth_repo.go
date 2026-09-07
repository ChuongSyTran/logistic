package repo

import (
	"context"
	"fmt"

	"auth_service/ent"
	"auth_service/ent/users"
	"auth_service/internal/biz"
	"auth_service/internal/entity"
	"auth_service/internal/mapper"

	"github.com/google/uuid"
	"github.com/logistic/pkg/authn"
)

type authRepoImpl struct {
	client *ent.Client
	mapper mapper.AuthMapper
}

func NewAuthRepo(client *ent.Client, mapper mapper.AuthMapper) biz.AuthRepo {
	return &authRepoImpl{
		client: client,
		mapper: mapper,
	}
}

func (r *authRepoImpl) FindByEmail(ctx context.Context, email string) (*entity.UserProfile, string, error) {
	u, err := r.client.Users.
		Query().
		Where(users.EmailEQ(email)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, "", fmt.Errorf("repo findByEmail: %w", biz.ErrInvalidCredentials)
		}
		return nil, "", fmt.Errorf("repo findByEmail: unexpected db error: %w", err)
	}

	var hashedPassword string
	if u.Password != nil {
		hashedPassword = *u.Password
	}

	profile := r.mapper.ToUserProfile(u)
	return profile, hashedPassword, nil
}

func (r *authRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.UserProfile, error) {
	u, err := r.client.Users.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("repo findByID: %w", biz.ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("repo findByID: unexpected db error: %w", err)
	}
	return r.mapper.ToUserProfile(u), nil
}

var roleFromAuthn = map[string]users.Role{
	authn.RoleDriver:  users.RoleDriver,
	authn.RoleShipper: users.RoleShipper,
	authn.RoleAdmin:   users.RoleAdmin,
}

func (r *authRepoImpl) Save(ctx context.Context, user entity.UserRegister, hashedPassword string) (*entity.UserProfile, error) {
	if user.Role == "" {
		user.Role = authn.RoleShipper
	}
	role, ok := roleFromAuthn[user.Role]
	if !ok {
		return nil, fmt.Errorf("repo save: vai trò %q không hợp lệ", user.Role)
	}

	createBuilder := r.client.Users.
		Create().
		SetEmail(user.Email).
		SetFullName(user.FullName).
		SetPassword(hashedPassword).
		SetRole(role)

	if user.GoogleID != "" {
		createBuilder.SetGoogleID(user.GoogleID)
	}

	u, err := createBuilder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("repo save: %w", biz.ErrEmailAlreadyExists)
		}
		return nil, fmt.Errorf("repo save: unexpected db error: %w", err)
	}

	return r.mapper.ToUserProfile(u), nil
}

func (r *authRepoImpl) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	exists, err := r.client.Users.
		Query().
		Where(users.EmailEQ(email)).
		Exist(ctx)

	if err != nil {
		return false, fmt.Errorf("repo existsByEmail: %w", err)
	}

	return exists, nil
}