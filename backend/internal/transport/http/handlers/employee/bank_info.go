package employee

import (
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// formatOptionalDate renders an optional date as YYYY-MM-DD for responses.
func formatOptionalDate(date *time.Time) *string {
	if date == nil {
		return nil
	}
	formatted := date.Format("2006-01-02")
	return &formatted
}

// buildEmployeeBankInfo projects the preloaded branch reference for employee
// responses; nil when the employee has no bank captured.
func buildEmployeeBankInfo(bank *domain.Bank) *dto.EmployeeBankInfo {
	if bank == nil {
		return nil
	}
	return &dto.EmployeeBankInfo{ID: bank.ID, BranchName: bank.BranchName}
}

// ListBanks serves the seeded branch reference list used by the bank capture
// UI. Read-only reference data — the seed is managed outside the product.
func (h *Handler) ListBanks(c *gin.Context) {
	banks, err := h.employeeService.ListBanks(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Không thể lấy danh sách ngân hàng")
		return
	}
	items := make([]dto.BankItem, 0, len(banks))
	for _, bank := range banks {
		items = append(items, dto.BankItem{ID: bank.ID, BranchName: bank.BranchName})
	}
	response.Success(c, dto.BanksResponse{Banks: items}, "Danh sách ngân hàng")
}
