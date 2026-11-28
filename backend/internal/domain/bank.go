package domain

import "context"

// Bank is one bank branch from the reference list partners pick from when
// capturing an employee's account details. The demo-era bin/code/swift columns
// stay retired; the employee contract only needs id + branch_name.
type Bank struct {
	ID         uint   `json:"id" gorm:"primarykey;type:bigint unsigned"`
	BranchName string `json:"branch_name" gorm:"type:varchar(255);uniqueIndex;not null"`
}

// BankRepository reads the bank branch reference list. Read-only on purpose:
// the list is a frozen seed snapshot, managed outside the product until a
// bank CRUD surface returns.
type BankRepository interface {
	List(ctx context.Context) ([]Bank, error)
	GetByID(ctx context.Context, id uint) (*Bank, error)
	SearchByBranchName(ctx context.Context, searchTerm string, limit int) ([]Bank, error)
}
