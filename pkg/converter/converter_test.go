package converter

import (
	"testing"
)

func TestNewPFXConverter(t *testing.T) {
	converter := NewPFXConverter()
	if converter == nil {
		t.Error("NewPFXConverter() returned nil")
	}
}

func TestAnnotationConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"AnnotationPFXConvert", AnnotationPFXConvert, "pfx-tls.kubernetes.io/convert"},
		{"AnnotationPFXPassword", AnnotationPFXPassword, "pfx-tls.kubernetes.io/password"},
		{"AnnotationPFXPasswordSecretName", AnnotationPFXPasswordSecretName, "pfx-tls.kubernetes.io/password-secret-name"},
		{"AnnotationPFXPasswordSecretKey", AnnotationPFXPasswordSecretKey, "pfx-tls.kubernetes.io/password-secret-key"},
		{"AnnotationPFXDataKey", AnnotationPFXDataKey, "pfx-tls.kubernetes.io/pfx-key"},
		{"AnnotationConverted", AnnotationConverted, "pfx-tls.kubernetes.io/converted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be %s, got %s", tt.name, tt.expected, tt.constant)
			}
		})
	}
}
