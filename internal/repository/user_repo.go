package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"

	"github.com/jellydator/ttlcache/v3"
	"gorm.io/gorm"
)

type UserRepository struct {
	db    *gorm.DB
	cache *repoCache
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db:    db,
		cache: newRepoCache(UserCacheTTL),
	}
}

// GetUserByUsername retrieves a user by their username
func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	key := userCacheKey(username)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			// Return a shallow copy to prevent callers from mutating cached data
			original := item.Value().(*model.User)
			cp := *original
			return &cp, nil
		}
	}

	var user model.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(key, &user, ttlcache.DefaultTTL)
		// Return a copy so the caller cannot mutate the cached object
		cp := user
		return &cp, nil
	}
	return &user, nil
}

// GetUserByUploadToken retrieves a user by their upload token
func (r *UserRepository) GetUserByUploadToken(token string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("upload_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(user *model.User) (*model.User, error) {
	// Set default nickname if not provided
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser updates an existing user's information (PUT semantics)
func (r *UserRepository) UpdateUser(user *model.User) (*model.User, error) {
	if err := r.db.Save(user).Error; err != nil {
		return nil, err
	}
	// Invalidate cache for this user after successful DB write
	if r.cache != nil {
		r.cache.Delete(userCacheKey(user.Username))
	}
	return user, nil
}

// WithTransaction executes fn within a database transaction, passing a transactional
// copy of UserRepository. If fn returns an error the transaction is rolled back.
// The transactional repo shares the same cache so writes inside the TX trigger
// invalidation on the shared cache.
func (r *UserRepository) WithTransaction(fn func(txRepo *UserRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&UserRepository{db: tx, cache: r.cache})
	})
}
