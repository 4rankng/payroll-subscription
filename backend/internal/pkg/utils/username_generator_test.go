package utils

import (
	"testing"
)

func TestGenerateUsername(t *testing.T) {
	tests := []struct {
		name     string
		fullname string
		want     string
	}{
		{
			name:     "standard vietnamese name",
			fullname: "Nguyen Thi Hong Tham",
			want:     "thamnth",
		},
		{
			name:     "another vietnamese name",
			fullname: "Dang Quan Khai",
			want:     "khaidq",
		},
		{
			name:     "single name",
			fullname: "Tham",
			want:     "tham",
		},
		{
			name:     "two part name",
			fullname: "Nguyen Tham",
			want:     "thamn",
		},
		{
			name:     "empty string",
			fullname: "",
			want:     "",
		},
		{
			name:     "name with extra spaces",
			fullname: "Nguyen  Thi  Tham",
			want:     "thamnt",
		},
		{
			name:     "lowercase input",
			fullname: "nguyen van a",
			want:     "anv",
		},
		{
			name:     "uppercase input",
			fullname: "NGUYEN VAN B",
			want:     "bnv",
		},
		{
			name:     "name with accents",
			fullname: "Trần Thị Hồng",
			want:     "hongtt",
		},
		{
			name:     "complex vietnamese name",
			fullname: "Lê Hoàng Phúc An",
			want:     "anlhp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUsername(tt.fullname)
			if got != tt.want {
				t.Errorf("GenerateUsername(%q) = %q, want %q", tt.fullname, got, tt.want)
			}
		})
	}
}

// Create a checker function that returns true if username exists

// Test with a checker that always returns false (all usernames available)

func TestGenerateUsernameEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		fullname string
		wantLen  int
	}{
		{
			name:     "very long name",
			fullname: "Nguyen Thi Hong Tham Anh Mai Phuong",
			wantLen:  12, // "phuong" + 6 initials
		},
		{
			name:     "single character parts",
			fullname: "A B C D",
			wantLen:  4, // "d" + "abc"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUsername(tt.fullname)
			if len(got) != tt.wantLen {
				t.Errorf("GenerateUsername(%q) length = %d, want %d (got: %q)", tt.fullname, len(got), tt.wantLen, got)
			}
		})
	}
}
