package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"
)

// BankRepository reads the bank branch reference table. Write operations are
// intentionally absent: the list is a frozen seed snapshot managed outside the
// product until a bank CRUD surface returns.
type BankRepository struct {
	db *Database
}

func NewBankRepository(db *Database) domain.BankRepository {
	return &BankRepository{db: db}
}

func (r *BankRepository) List(ctx context.Context) ([]domain.Bank, error) {
	var banks []domain.Bank
	if err := r.db.WithContext(ctx).Order("branch_name ASC").Find(&banks).Error; err != nil {
		return nil, fmt.Errorf("failed to list banks: %w", err)
	}
	return banks, nil
}

func (r *BankRepository) GetByID(ctx context.Context, id uint) (*domain.Bank, error) {
	var bank domain.Bank
	if err := r.db.WithContext(ctx).First(&bank, id).Error; err != nil {
		return nil, err
	}
	return &bank, nil
}

func (r *BankRepository) SearchByBranchName(ctx context.Context, searchTerm string, limit int) ([]domain.Bank, error) {
	var banks []domain.Bank
	query := r.db.WithContext(ctx)
	if searchTerm != "" {
		pattern := escapeLike(searchTerm)
		query = query.Where("branch_name LIKE ? ESCAPE '\\\\'", "%"+pattern+"%")
	}
	if limit <= 0 {
		limit = 50
	}
	if err := query.Limit(limit).Order("branch_name ASC").Find(&banks).Error; err != nil {
		return nil, fmt.Errorf("failed to search banks: %w", err)
	}
	return banks, nil
}
