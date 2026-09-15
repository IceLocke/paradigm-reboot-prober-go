package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		db      func(*Config)
		wantErr string
	}{
		{
			name:    "sqlite with dsn",
			db:      func(c *Config) { c.Database.Type = "sqlite"; c.Database.DSN = "prober.db" },
			wantErr: "",
		},
		{
			name:    "sqlite without dsn",
			db:      func(c *Config) { c.Database.Type = "sqlite" },
			wantErr: `database.dsn must be set when database.type is "sqlite"`,
		},
		{
			name: "postgres complete",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.User = "postgres"
				c.Database.DBName = "prp"
				c.Database.SSLMode = "disable"
			},
			wantErr: "",
		},
		{
			name: "postgres empty sslmode falls back to libpq default",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.User = "postgres"
				c.Database.DBName = "prp"
			},
			wantErr: "",
		},
		{
			name: "postgres missing host",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Port = 5432
				c.Database.User = "postgres"
				c.Database.DBName = "prp"
			},
			wantErr: `database.host must be set when database.type is "postgres"`,
		},
		{
			name: "postgres missing user",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.DBName = "prp"
			},
			wantErr: `database.user must be set when database.type is "postgres"`,
		},
		{
			name: "postgres missing dbname",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.User = "postgres"
			},
			wantErr: `database.dbname must be set when database.type is "postgres"`,
		},
		{
			name: "postgres zero port",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 0
				c.Database.User = "postgres"
				c.Database.DBName = "prp"
			},
			wantErr: `database.port must be in 1-65535 when database.type is "postgres", got 0`,
		},
		{
			name: "postgres port out of range",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 70000
				c.Database.User = "postgres"
				c.Database.DBName = "prp"
			},
			wantErr: `database.port must be in 1-65535 when database.type is "postgres", got 70000`,
		},
		{
			name: "postgres host with whitespace breaks DSN",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db host"
				c.Database.Port = 5432
				c.Database.User = "postgres"
				c.Database.DBName = "prp"
			},
			wantErr: `database.host must not contain whitespace, quotes, or backslashes: "db host"`,
		},
		{
			name: "postgres user with quote breaks DSN",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.User = "pg'user"
				c.Database.DBName = "prp"
			},
			wantErr: `database.user must not contain whitespace, quotes, or backslashes: "pg'user"`,
		},
		{
			name: "postgres dbname with backslash breaks DSN",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.User = "postgres"
				c.Database.DBName = `pr\p`
			},
			wantErr: `database.dbname must not contain whitespace, quotes, or backslashes: "pr\\p"`,
		},
		{
			name: "postgres typo'd sslmode",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.User = "postgres"
				c.Database.DBName = "prp"
				c.Database.SSLMode = "requier"
			},
			wantErr: `database.sslmode must be one of disable, allow, prefer, require, verify-ca, verify-full, got "requier"`,
		},
		{
			name: "postgres password with special chars is escaped downstream, not rejected",
			db: func(c *Config) {
				c.Database.Type = "postgres"
				c.Database.Host = "db"
				c.Database.Port = 5432
				c.Database.User = "postgres"
				c.Database.Password = `p'a\s s`
				c.Database.DBName = "prp"
			},
			wantErr: "",
		},
		{
			name:    "unsupported type",
			db:      func(c *Config) { c.Database.Type = "mysql" },
			wantErr: `unsupported database.type "mysql": must be "sqlite" or "postgres"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			tt.db(c)
			err := c.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
