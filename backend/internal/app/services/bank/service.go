package bank

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/domain"
)

// BankService reads the bank branch reference list and resolves raw bank
// names (Excel imports) to reference rows.
type BankService struct {
	BankRepo domain.BankRepository
}

func NewBankService(repo domain.BankRepository) *BankService {
	return &BankService{BankRepo: repo}
}

// List returns every seeded branch for the reference-list endpoint.
func (s *BankService) List(ctx context.Context) ([]domain.Bank, error) {
	return s.BankRepo.List(ctx)
}

// GetByID fetches one branch; used to embed bank info in employee responses.
func (s *BankService) GetByID(ctx context.Context, id uint) (*domain.Bank, error) {
	return s.BankRepo.GetByID(ctx, id)
}

// ResolveName maps a raw bank name (Excel import) to a branch id. Returns nil
// when the name is empty or no seeded branch matches — non-fatal by design so
// an unknown bank name never blocks an import row.
func (s *BankService) ResolveName(ctx context.Context, bankName string) *uint {
	if bankName == "" {
		return nil
	}
	banks, err := s.BankRepo.SearchByBranchName(ctx, bankName, 1)
	if err != nil {
		slog.Warn("bank search failed", "bank_name", bankName, "error", err)
		return nil
	}
	if len(banks) == 0 {
		slog.Warn("bank not found in reference list", "bank_name", bankName)
		return nil
	}
	return &banks[0].ID
}

// ValidateExists confirms a create request references a seeded branch, so a
// bogus bank id surfaces as a 400 instead of a foreign-key failure.
func (s *BankService) ValidateExists(ctx context.Context, id uint) error {
	if _, err := s.BankRepo.GetByID(ctx, id); err != nil {
		return domain.NewValidationError(fmt.Sprintf("bank %d does not exist", id))
	}
	return nil
}
