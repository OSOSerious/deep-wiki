package db

import (
	"context"
	"database/sql"
	"fmt"
)

// SetTenantContext sets the current tenant for RLS
func SetTenantContext(ctx context.Context, tx *sql.Tx, tenantID string) error {
	_, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// WithTenant executes a function with tenant isolation
func WithTenant(ctx context.Context, db *sql.DB, tenantID string, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := SetTenantContext(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// SetUserContext sets both tenant and user context for RLS
func SetUserContext(ctx context.Context, tx *sql.Tx, tenantID, userID string) error {
	// Set tenant context
	if err := SetTenantContext(ctx, tx, tenantID); err != nil {
		return err
	}
	
	// Set user context
	_, err := tx.ExecContext(ctx, "SET LOCAL app.current_user = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to set user context: %w", err)
	}
	
	return nil
}

// WithUserContext executes a function with both tenant and user isolation
func WithUserContext(ctx context.Context, db *sql.DB, tenantID, userID string, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := SetUserContext(ctx, tx, tenantID, userID); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}