package minio

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Endpoint != "localhost:9000" {
		t.Errorf("expected default endpoint localhost:9000, got %s", cfg.Endpoint)
	}
	if cfg.AccessKey != "minioadmin" {
		t.Errorf("expected default AccessKey minioadmin, got %s", cfg.AccessKey)
	}
	if cfg.SecretKey != "minioadmin" {
		t.Errorf("expected default SecretKey minioadmin, got %s", cfg.SecretKey)
	}
	if cfg.UseSSL {
		t.Error("expected default UseSSL=false")
	}
	if cfg.MaxIdleConns != 100 {
		t.Errorf("expected default MaxIdleConns 100, got %d", cfg.MaxIdleConns)
	}
}

func TestOptions(t *testing.T) {
	cfg := DefaultConfig()
	WithEndpoint("minio:9000")(&cfg)
	WithCredentials("key", "secret")(&cfg)
	WithSSL()(&cfg)
	WithRegion("eu-west-1")(&cfg)

	if cfg.Endpoint != "minio:9000" {
		t.Errorf("Endpoint = %s", cfg.Endpoint)
	}
	if cfg.AccessKey != "key" || cfg.SecretKey != "secret" {
		t.Errorf("credentials = %s/%s", cfg.AccessKey, cfg.SecretKey)
	}
	if !cfg.UseSSL {
		t.Error("UseSSL should be true")
	}
	if cfg.Region != "eu-west-1" {
		t.Errorf("Region = %s", cfg.Region)
	}
}

func TestDefaultBucket(t *testing.T) {
	if DefaultBucket != "dem-files" {
		t.Errorf("DefaultBucket = %s", DefaultBucket)
	}
}
