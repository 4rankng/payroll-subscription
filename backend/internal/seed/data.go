package seed

import (
	"api-server/internal/pkg/clock"

	"github.com/brianvoe/gofakeit/v7"
)

// UserProfile represents a complete user profile for generation
type UserProfile struct {
	FullName    string
	Username    string
	Email       string
	Phone       string
	Address     string
	CCCD        string
	BankAccount string
}

// Vietnamese data generators with weighted frequencies
var vietnameseFirstNames = map[string]int{
	"Nguyễn": 40, "Trần": 11, "Lê": 9, "Phạm": 7, "Hoàng": 4, "Huỳnh": 4,
	"Phan": 3, "Vũ": 3, "Võ": 3, "Đặng": 2, "Bùi": 2, "Đỗ": 2,
	"Hồ": 2, "Ngô": 2, "Dương": 2, "Lý": 2, "Mai": 2, "Đinh": 2, "Vương": 2,
	"Chu": 1, "Cao": 1, "Trương": 1, "Tô": 1, "Lưu": 1, "Hà": 1,
}

var vietnameseLastNames = []string{
	"Văn Minh", "Thị Hương", "Văn Hùng", "Thị Lan", "Quốc Dũng", "Thị Mai",
	"Văn Thành", "Thị Linh", "Đức Anh", "Thị Nga", "Văn Long", "Thị Thu",
	"Quang Huy", "Thị Vân", "Văn Khoa", "Thị Hoa", "Đình Trọng", "Thị Xuân",
	"Văn Tài", "Thị Bích", "Quốc Việt", "Thị Phương", "Văn Đức", "Thị Trang",
	"Minh Tuấn", "Thị Yến", "Văn Nam", "Thị Thùy", "Đức Thắng", "Thị Loan",
	"Thanh Huyền", "Anh Tuấn", "Thị Kim", "Văn Hải", "Quang Minh", "Thị Nhung",
	"Đức Long", "Thị Hạnh", "Văn Quý", "Thị Tuyết", "Minh Đức", "Thị Hòa",
}

var VietnameseCompanies = []string{
	"Công ty TNHH Sản xuất Hóa mỹ phẩm VERICO",
	"Công ty Cổ phần Dệt may Việt Nam",
	"Công ty TNHH Sản xuất Giày dép Thành Công",
	"Công ty Cổ phần Thực phẩm Sạch Việt",
	"Công ty TNHH Sản xuất Đồ gỗ Nội thất",
	"Công ty Cổ phần Điện tử Việt Nam",
	"Công ty TNHH Sản xuất Nhựa Thái Bình",
	"Công ty Cổ phần Dược phẩm Hà Nội",
}

var VietnameseBanks = []struct {
	Name string
}{
	{"Ngân hàng TMCP Ngoại thương Việt Nam - Chi nhánh Hà Nội"},
	{"Ngân hàng Đầu tư và Phát triển Việt Nam - Chi nhánh TPHCM"},
	{"Ngân hàng Nông nghiệp và Phát triển Nông thôn - Chi nhánh Đống Đa"},
	{"Ngân hàng TMCP Kỹ thương Việt Nam - Chi nhánh Cầu Giấy"},
	{"Ngân hàng TMCP Sài Gòn Thương Tín - Chi nhánh Hoàn Kiếm"},
	{"Ngân hàng TMCP Á Châu - Chi nhánh Tây Hồ"},
	{"Ngân hàng TMCP Quân đội - Chi nhánh Ba Đình"},
	{"Ngân hàng TMCP Việt Nam Thịnh Vượng - Chi nhánh Hai Bà Trưng"},
}

// Generate weighted random first name

// Generate 12-digit CCCD starting with valid province codes

// Generate remaining 9 digits

// Fallback to random generation if all numbered options are taken

// Try basic email first

// Try with numbers

// Fallback to random suffix

// Should never reach here in normal cases

// Fallback to random generation

// GenerateRealisticUserProfile creates a complete user profile with collision avoidance

// GenerateUniqueUsernameWithDB provides enhanced collision detection with database verification

// Double-check against database

// Fallback to UUID-based generation for ultimate uniqueness

// GenerateUniqueEmailWithDB provides enhanced collision detection with database verification

// Try basic email first

// Try with numbers

// Fallback to UUID-based generation

// EmployeeProfile represents a complete employee profile with career progression
type EmployeeProfile struct {
	*UserProfile
	Position       string
	Experience     string
	Seniority      int // years of experience
	BasePayRate    int // VND per hour
	IsReliable     bool
	AttendanceRate float64
}

// ProjectProfile represents realistic project data
type ProjectProfile struct {
	Name             string
	ClientName       string
	Code             string
	Description      string
	Status           string
	Budget           float64
	ExpectedDuration int // months
	TeamSize         int
}

// Vietnamese position hierarchy with typical pay rates (VND per hour)
var vietnamesePositions = map[string]struct {
	BaseRate    int
	MinExp      int
	MaxExp      int
	Description string
}{
	"thực tập":       {15000, 0, 1, "Thực tập sinh - người mới bắt đầu"},
	"phổ thông":      {25000, 0, 3, "Công nhân phổ thông - không yêu cầu kinh nghiệm đặc biệt"},
	"có kinh nghiệm": {35000, 2, 8, "Công nhân có kinh nghiệm - đã quen với công việc"},
	"chuyên môn":     {45000, 3, 10, "Công nhân chuyên môn - có kỹ năng đặc biệt"},
	"tổ trưởng":      {55000, 5, 15, "Tổ trưởng - quản lý nhóm nhỏ"},
	"giám sát":       {65000, 7, 20, "Giám sát viên - quản lý ca làm việc"},
	"kỹ thuật":       {75000, 4, 15, "Kỹ thuật viên - vận hành máy móc phức tạp"},
}

// Vietnamese project types based on common manufacturing industries
var vietnameseProjectTypes = []struct {
	Industry     string
	ProjectTypes []string
	AvgBudget    float64 // million VND
	AvgDuration  int     // months
}{
	{
		"Dệt may",
		[]string{"Sản xuất áo thun", "Sản xuất quần jean", "Sản xuất đồng phục", "Dệt vải cotton"},
		2500,
		8,
	},
	{
		"Hóa mỹ phẩm",
		[]string{"Sản xuất xà phòng", "Sản xuất kem dưỡng da", "Sản xuất dầu gội", "Sản xuất nước hoa"},
		1800,
		6,
	},
	{
		"Thực phẩm",
		[]string{"Chế biến thực phẩm đông lạnh", "Sản xuất bánh kẹo", "Chế biến hải sản", "Sản xuất nước giải khát"},
		3200,
		10,
	},
	{
		"Điện tử",
		[]string{"Lắp ráp linh kiện điện tử", "Sản xuất cáp điện", "Lắp ráp thiết bị điện tử", "Sản xuất bo mạch"},
		4500,
		12,
	},
	{
		"Đồ gỗ",
		[]string{"Sản xuất nội thất", "Chế biến gỗ xuất khẩu", "Sản xuất đồ chơi gỗ", "Sản xuất đồ gia dụng"},
		2000,
		9,
	},
}

// GenerateRealisticEmployeeProfile creates a comprehensive employee profile

// Determine seniority and position based on age-like factors
// 20-60 years old
// assuming start working at 22

// Select position based on experience

// Add variation based on individual performance
// ±15% variation

// Determine reliability and attendance
// 75% of workers are reliable

// 85-98%

// 60-85%

// GenerateRealisticProjectProfile creates a comprehensive project profile

// Generate realistic client company name

// Generate budget with realistic variation
// 0.5x to 1.5x base budget
// convert to VND

// Generate duration with variation
// 0.7x to 1.3x

// Team size based on budget and duration
// rough calculation based on average worker cost

// Generate project status based on realistic distribution

// GenerateRealisticPayrates creates realistic pay rate structures

// 150% for overtime (Vietnamese labor law)
// 200% for weekend work
// 300% for holiday work

// Add some position-specific variations

// Supervisors get slightly higher weekend/holiday premiums

// Interns get lower premiums

// GenerateVietnameseHolidays returns major Vietnamese holidays for a given year

// New Year
// Liberation Day
// Labor Day
// National Day

// Add Tet holidays (approximate - 3-5 days in Jan/Feb)

func init() {
	_ = gofakeit.Seed(clock.Now().UnixNano())
}
