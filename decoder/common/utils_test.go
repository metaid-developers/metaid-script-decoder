package common

import "testing"

func TestGetParentPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/protocols/simplebuzz", "/protocols"},
		{"/info/name", "/info"},
		{"/a/b/c/d", "/a/b/c"},
		{"/a", ""},
		{"", ""},
		{"/", ""},
	}

	for _, test := range tests {
		result := GetParentPath(test.input)
		if result != test.expected {
			t.Errorf("GetParentPath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestValidateOperation(t *testing.T) {
	validOps := []string{"create", "modify", "revoke", "CREATE", "MODIFY", "REVOKE"}
	for _, op := range validOps {
		if !ValidateOperation(op) {
			t.Errorf("ValidateOperation(%q) = false, expected true", op)
		}
	}

	invalidOps := []string{"init", "delete", "update", "remove", ""}
	for _, op := range invalidOps {
		if ValidateOperation(op) {
			t.Errorf("ValidateOperation(%q) = true, expected false", op)
		}
	}
}

func TestNormalizeContentType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"application/json", "application/json"},
		{"APPLICATION/JSON", "application/json"},
		{"  text/plain  ", "text/plain"},
		{"", "application/json"},
	}

	for _, test := range tests {
		result := NormalizeContentType(test.input)
		if result != test.expected {
			t.Errorf("NormalizeContentType(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Protocols/SimpleBuzz", "/protocols/simplebuzz"},
		{"  /Info/Name  ", "/info/name"},
		{"/test", "/test"},
	}

	for _, test := range tests {
		result := NormalizePath(test.input)
		if result != test.expected {
			t.Errorf("NormalizePath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
