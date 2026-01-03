package db

import (
	"context"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DB struct {
	cli *gorm.DB
	mu  sync.Mutex
}

func NewDB(cli *gorm.DB) *DB {
	return &DB{
		cli: cli,
		mu:  sync.Mutex{},
	}
}

func (db *DB) Create(ctx context.Context, value any, clauses []clause.Expression) *gorm.DB {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.cli.WithContext(ctx).Clauses(clauses...).Create(value)
}

func (db *DB) Save(ctx context.Context, value any, clauses []clause.Expression) *gorm.DB {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.cli.WithContext(ctx).Clauses(clauses...).Save(value)
}

func (db *DB) Exec(ctx context.Context, sql string, clauses []clause.Expression, values ...any) *gorm.DB {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.cli.WithContext(ctx).Clauses(clauses...).Exec(sql, values...)
}

func (db *DB) Raw(ctx context.Context, sql string, clauses []clause.Expression, values ...any) *gorm.DB {
	return db.cli.WithContext(ctx).Clauses(clauses...).Raw(sql, values...)
}

func (db *DB) AutoMigrate(models ...any) error {
	return db.cli.AutoMigrate(models...)
}

func (db *DB) Delete(ctx context.Context, value any, clauses []clause.Expression) *gorm.DB {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.cli.WithContext(ctx).Clauses(clauses...).Delete(value)
}

func (db *DB) First(ctx context.Context, dest any, conds ...any) *gorm.DB {
	return db.cli.WithContext(ctx).First(dest, conds...)
}

func (db *DB) BeginDangerously(ctx context.Context) *gorm.DB {
	return db.cli.WithContext(ctx).Begin()
}

func (db *DB) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.cli.WithContext(ctx).Transaction(fn)
}

func (db *DB) Lock() {
	db.mu.Lock()
}

func (db *DB) Unlock() {
	db.mu.Unlock()
}
