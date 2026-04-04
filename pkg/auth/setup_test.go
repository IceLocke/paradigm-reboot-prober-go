package auth

import (
	"os"
	"paradigm-reboot-prober-go/config"
	"testing"
)

func TestMain(m *testing.M) {
	config.InitDefaults()
	config.GlobalConfig.Auth.SecretKey = "testsecret"
	os.Exit(m.Run())
}
