package i18n

import (
	"slices"
	"testing"
)

func TestBundle_LoadYAMLDir(t *testing.T) {
	t.Run("Bundle_LoadYAMLDir_Success", func(t *testing.T) {
		bundle := New(Config{})
		err := bundle.LoadYAMLDir("./locales/")
		if err != nil {
			t.Fatalf("LoadYAMLDir: %v", err)
		}
	})
	t.Run("Bundle_LoadYAMLDir_Fail", func(t *testing.T) {
		bundle := New(Config{})
		err := bundle.LoadYAMLDir("./testdata/error/")
		if err != nil {
			t.Logf("LoadYAMLDir: %v", err)
		}
	})
}

func TestBundle_Locale(t *testing.T) {
	t.Run("Bundle_Locale_Success", func(t *testing.T) {
		bundle := New(Config{})
		err := bundle.LoadYAMLDir("./locales/")
		if err != nil {
			t.Fatalf("LoadYAMLDir: %v", err)
		}

		locale := bundle.Locale("en")
		if locale == nil {
			t.Fatalf("Locale: nil")
		}
		if !slices.Contains(locale.langs, "en") {
			t.Fatalf("Locale.langs: %v", locale.langs)
		}
	})
}
