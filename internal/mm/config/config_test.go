package config

import "testing"

func TestDefaultUsesModeloManToken(t *testing.T) {
	cfg := Default()
	if cfg.TokenEnvVar != "MODELOMAN_TOKEN" {
		t.Fatalf("unexpected token env var %q", cfg.TokenEnvVar)
	}
}
