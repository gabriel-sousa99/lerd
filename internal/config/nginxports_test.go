package config

import "testing"

func TestNginxPorts(t *testing.T) {
	// HOME alone does not move the config dir where XDG_CONFIG_HOME is already
	// set, which is how CI runs, and the write lands on the real config.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	t.Run("an install with no config gets the defaults", func(t *testing.T) {
		http, https := NginxPorts()
		if http != 80 || https != 443 {
			t.Errorf("got %d/%d, want 80/443", http, https)
		}
	})

	t.Run("configured ports win", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Nginx.HTTPPort = 10080
		cfg.Nginx.HTTPSPort = 10443
		if err := SaveGlobal(cfg); err != nil {
			t.Fatalf("saving config: %v", err)
		}
		http, https := NginxPorts()
		if http != 10080 || https != 10443 {
			t.Errorf("got %d/%d, want 10080/10443", http, https)
		}
	})
}
