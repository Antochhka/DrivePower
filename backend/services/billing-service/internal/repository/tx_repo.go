package repository

import (
	"context"
	"database/sql"

	"drivepower/backend/services/billing-service/internal/models"
)

// TransactionRepository persists billing transactions.
type TransactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository returns repository.
func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create inserts a new transaction.
func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	const query = `
		INSERT INTO billing_transactions (session_id, user_id, energy_kwh, price_per_kwh, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		tx.SessionID,
		tx.UserID,
		tx.EnergyKWh,
		tx.PricePerKWh,
		tx.Amount,
		tx.Status,
	).Scan(&tx.ID, &tx.CreatedAt)
}

// ListByUser returns latest transactions for user.
func (r *TransactionRepository) ListByUser(ctx context.Context, userID int64, limit int) ([]models.Transaction, error) {
	if limit <= 0 {
		limit = 50
	}
	const query = `
		SELECT id, session_id, user_id, energy_kwh, price_per_kwh, amount, status, created_at
		FROM billing_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.SessionID,
			&tx.UserID,
			&tx.EnergyKWh,
			&tx.PricePerKWh,
			&tx.Amount,
			&tx.Status,
			&tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return txs, nil
}

