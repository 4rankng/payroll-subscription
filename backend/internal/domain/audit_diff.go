package domain

// CompareUsers compares auditable fields of two users and returns changed fields.
func CompareUsers(original, updated *User) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.Username != updated.Username {
		changes["username"] = FieldChange{Before: original.Username, After: updated.Username}
	}
	if original.Fullname != updated.Fullname {
		changes["fullname"] = FieldChange{Before: original.Fullname, After: updated.Fullname}
	}
	if !equalStringPtr(original.Email, updated.Email) {
		changes["email"] = FieldChange{Before: strPtrToStr(original.Email), After: strPtrToStr(updated.Email)}
	}
	if !equalStringPtr(original.Mobile, updated.Mobile) {
		changes["mobile"] = FieldChange{Before: strPtrToStr(original.Mobile), After: strPtrToStr(updated.Mobile)}
	}
	if original.Role != updated.Role {
		changes["role"] = FieldChange{Before: string(original.Role), After: string(updated.Role)}
	}
	return changes
}

// CompareProjects compares auditable fields of two projects and returns changed fields.
func CompareProjects(original, updated *Project) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.Name != updated.Name {
		changes["name"] = FieldChange{Before: original.Name, After: updated.Name}
	}
	if original.Code != updated.Code {
		changes["code"] = FieldChange{Before: original.Code, After: updated.Code}
	}
	if original.ClientName != updated.ClientName {
		changes["client_name"] = FieldChange{Before: original.ClientName, After: updated.ClientName}
	}
	if original.Description != updated.Description {
		changes["description"] = FieldChange{Before: original.Description, After: updated.Description}
	}
	if original.ProjectStatus != updated.ProjectStatus {
		changes["project_status"] = FieldChange{Before: string(original.ProjectStatus), After: string(updated.ProjectStatus)}
	}
	if original.OffDays != updated.OffDays {
		changes["off_days"] = FieldChange{Before: original.OffDays, After: updated.OffDays}
	}
	return changes
}

// CompareTimesheets compares auditable fields of two timesheets and returns changed fields.
func CompareTimesheets(original, updated *Timesheet) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.HoursWorked != updated.HoursWorked {
		changes["hours_worked"] = FieldChange{Before: original.HoursWorked, After: updated.HoursWorked}
	}
	if original.Amount != updated.Amount {
		changes["amount"] = FieldChange{Before: original.Amount, After: updated.Amount}
	}
	if original.PayRate != updated.PayRate {
		changes["pay_rate"] = FieldChange{Before: original.PayRate, After: updated.PayRate}
	}
	if original.Status != updated.Status {
		changes["status"] = FieldChange{Before: string(original.Status), After: string(updated.Status)}
	}
	if original.PaymentStatus != updated.PaymentStatus {
		changes["payment_status"] = FieldChange{Before: string(original.PaymentStatus), After: string(updated.PaymentStatus)}
	}
	if original.Date != updated.Date {
		changes["date"] = FieldChange{Before: original.Date.Format("2006-01-02"), After: updated.Date.Format("2006-01-02")}
	}
	return changes
}
