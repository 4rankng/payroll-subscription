package dto

// BankItem is one bank branch in the reference-list response.
type BankItem struct {
	ID         uint   `json:"id"`
	BranchName string `json:"branch_name"`
}

// BanksResponse is the body of GET /banks. The list is read-only reference
// data; there is no pagination until a bank CRUD surface returns.
type BanksResponse struct {
	Banks []BankItem `json:"banks"`
}
