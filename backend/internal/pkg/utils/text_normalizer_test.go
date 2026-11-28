package utils

import (
	"testing"
)

func TestNewVietnameseNormalizer(t *testing.T) {
	normalizer := NewVietnameseNormalizer()
	if normalizer == nil {
		t.Fatal("NewVietnameseNormalizer() returned nil")
		return
	}

	if normalizer.cache == nil {
		t.Error("NewVietnameseNormalizer() created normalizer with nil cache")
	}

	if len(normalizer.cache) != 0 {
		t.Errorf("NewVietnameseNormalizer() cache should be empty, got %d items", len(normalizer.cache))
	}
}

func TestVietnameseNormalizer_Normalize(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "english text",
			input: "Hello World",
			want:  "hello world",
		},
		{
			name:  "vietnamese with accents",
			input: "Nguyễn Văn A",
			want:  "nguyen van a",
		},
		{
			name:  "mixed case vietnamese",
			input: "HỒNG HÀ",
			want:  "hong ha",
		},
		{
			name:  "already normalized",
			input: "abc",
			want:  "abc",
		},
		{
			name:  "numbers and text",
			input: "Số 123",
			want:  "so 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVietnameseNormalizer_NormalizeForSearch(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple search",
			input: "Nguyễn",
			want:  "%nguyen%",
		},
		{
			name:  "empty search",
			input: "",
			want:  "%%",
		},
		{
			name:  "english search",
			input: "Test",
			want:  "%test%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.NormalizeForSearch(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeForSearch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Normalize a string

// Check cache size

// Normalize the same string again (should use cache)

// Cache size should still be 1

// Add another string

// Cache size should be 2

// Add some items to cache

// Clear cache

func TestNormalizeVietnamese(t *testing.T) {
	// Test global convenience function
	result := NormalizeVietnamese("Hà Nội")
	expected := "ha noi"

	if result != expected {
		t.Errorf("NormalizeVietnamese() = %q, want %q", result, expected)
	}
}

func TestNormalizeVietnameseForSearch(t *testing.T) {
	// Test global convenience function
	result := NormalizeVietnameseForSearch("Hà Nội")
	expected := "%ha noi%"

	if result != expected {
		t.Errorf("NormalizeVietnameseForSearch() = %q, want %q", result, expected)
	}
}

// Simulate concurrent access

// Wait for all goroutines

// Should not panic and cache should have the item
