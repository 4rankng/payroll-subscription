package domain

import (
	"fmt"
	"strings"
)

// AuditMatrix provides clean Vietnamese audit message templates keyed by (AuditAction, EntityType).
// Templates include the actor as the first parameter and produce complete, ready-to-use sentences.
var AuditMatrix = map[AuditAction]map[EntityType]string{
	AuditActionCreate: {
		EntityTypeUser:            "%s đã tạo người dùng %s",
		EntityTypeEmployee:        "%s đã tạo hồ sơ nhân viên %s",
		EntityTypeProject:         "%s đã tạo dự án %s",
		EntityTypeProjectEmployee: "%s đã phân công %s vào dự án %s với vai trò %s",
		EntityTypeProjectUser:     "%s đã chia sẻ dự án %s với người dùng %s",
		EntityTypeEmployeeUser:    "%s đã cấp quyền truy cập cho %s vào dự án %s với vai trò %s",
		EntityTypeTimesheet:       "%s đã tạo bảng công cho %s trong dự án %s",
		EntityTypeAsset:           "%s đã tải lên tài liệu %s",
		EntityTypeSettings:        "%s đã tạo thiết lập %s",
		EntityTypePayrate:         "%s đã tạo mức lương cho dự án %s",
	},
	AuditActionUpdate: {
		EntityTypeUser:            "%s đã cập nhật thông tin người dùng %s",
		EntityTypeEmployee:        "%s đã cập nhật hồ sơ nhân viên %s",
		EntityTypeProject:         "%s đã cập nhật dự án %s",
		EntityTypeProjectEmployee: "%s đã cập nhật phân công của nhân viên %s trong dự án %s",
		EntityTypeProjectUser:     "%s đã cập nhật chia sẻ dự án %s với %s",
		EntityTypeEmployeeUser:    "%s đã cập nhật quyền truy cập nhân viên %s trong dự án %s với vai trò %s",
		EntityTypeTimesheet:       "%s đã cập nhật bảng công của nhân viên %s trong dự án %s",
		EntityTypeAsset:           "%s đã cập nhật tài liệu %s",
		EntityTypeSettings:        "%s đã cập nhật thiết lập %s",
		EntityTypePayrate:         "%s đã cập nhật mức lương cho dự án %s",
	},
	AuditActionDelete: {
		EntityTypeUser:            "%s đã xóa người dùng %s",
		EntityTypeEmployee:        "%s đã xóa hồ sơ nhân viên %s",
		EntityTypeProject:         "%s đã xóa dự án %s",
		EntityTypeProjectEmployee: "%s đã xóa phân công nhân viên %s khỏi dự án %s",
		EntityTypeProjectUser:     "%s đã xóa chia sẻ dự án %s với %s",
		EntityTypeEmployeeUser:    "%s đã xóa quyền truy cập nhân viên %s khỏi dự án %s",
		EntityTypeTimesheet:       "%s đã xóa bảng công của nhân viên %s trong dự án %s",
		EntityTypeAsset:           "%s đã xóa tài liệu %s",
		EntityTypeSettings:        "%s đã xóa thiết lập %s",
		EntityTypePayrate:         "%s đã xóa mức lương cho dự án %s",
	},
	AuditActionApprove: {
		EntityTypeTimesheet: "%s đã duyệt bảng công của nhân viên %s trong dự án %s",
	},
	AuditActionReject: {
		EntityTypeTimesheet: "%s đã từ chối bảng công của nhân viên %s trong dự án %s",
	},
	AuditActionBulkApprove: {
		EntityTypeTimesheet: "%s đã duyệt hàng loạt %s bảng công",
	},
	AuditActionBulkReject: {
		EntityTypeTimesheet: "%s đã từ chối hàng loạt %s bảng công",
	},
	AuditActionBulkReset: {
		EntityTypeTimesheet: "%s đã đặt lại hàng loạt %s bảng công",
	},
	AuditActionBulkCreate: {
		EntityTypeTimesheet: "%s đã tạo hàng loạt %s bảng công",
	},
	AuditActionLogin: {
		EntityTypeUser: "%s đã đăng nhập vào hệ thống",
	},
	AuditActionLogout: {
		EntityTypeUser: "%s đã đăng xuất khỏi hệ thống",
	},
	AuditActionChangePassword: {
		EntityTypeUser: "%s đã đổi mật khẩu",
	},
	AuditActionExport: {
		EntityTypeAsset:     "%s đã xuất dữ liệu ra tệp %s",
		EntityTypeTimesheet: "%s đã xuất bảng công dự án %s",
		EntityTypeEmployee:  "%s đã xuất danh sách nhân viên",
	},
	AuditActionImport: {
		EntityTypeAsset:     "%s đã nhập dữ liệu từ tệp %s",
		EntityTypeTimesheet: "%s đã nhập bảng công từ tệp %s",
		EntityTypeEmployee:  "%s đã nhập danh sách nhân viên từ tệp %s",
	},
}

// LookupTemplate returns the audit message template for an (action, entity) pair.
func LookupTemplate(action AuditAction, entityType EntityType) (string, bool) {
	entities, ok := AuditMatrix[action]
	if !ok {
		return "", false
	}
	template, ok := entities[entityType]
	return template, ok
}

// BuildEventAuditMessage renders the template for (action, entityType) with the
// actor as the first parameter followed by the event's context arguments.
func BuildEventAuditMessage(action AuditAction, entityType EntityType, actorName string, contextArgs ...interface{}) string {
	template, ok := LookupTemplate(action, entityType)
	if !ok {
		return ""
	}

	// Actor is the first parameter, followed by context-specific parameters
	allArgs := append([]interface{}{actorName}, contextArgs...)
	return fmt.Sprintf(template, allArgs...)
}

// MapRoleToVietnamese converts a role value to its Vietnamese display name.
func MapRoleToVietnamese(role string) string {
	switch strings.ToLower(role) {
	case "admin":
		return "Quản trị viên"
	case "quanly":
		return "Quản lý"
	case "giamdoc":
		return "Giám đốc"
	case "nhanvien":
		return "Nhân viên"
	default:
		return "Người dùng"
	}
}
