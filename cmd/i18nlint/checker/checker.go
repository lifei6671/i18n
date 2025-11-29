package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/lifei6671/i18n"
)

type LangFile struct {
	Language string            `yaml:"language"`
	Messages map[string]string `yaml:"messages"`
}

type Result struct {
	Languages     []string
	MissingKeys   map[string][]string
	RedundantKeys map[string][]string
	SyntaxErrors  map[string]map[string]error // lang -> key -> err
	AllKeys       []string
}

// CheckLocales performs:
//  1. key alignment check (missing / redundant)
//  2. template syntax check via i18n.ValidateTemplate()
func CheckLocales(dir string) (*Result, error) {
	files, err := scanYAML(dir)
	if err != nil {
		return nil, err
	}

	langKeys := make(map[string]map[string]struct{})
	allKeysSet := make(map[string]struct{})

	for _, file := range files {
		kset := make(map[string]struct{})
		for k := range file.Messages {
			kset[k] = struct{}{}
			allKeysSet[k] = struct{}{}
		}
		langKeys[file.Language] = kset
	}

	allKeys := make([]string, 0, len(allKeysSet))
	for k := range allKeysSet {
		allKeys = append(allKeys, k)
	}
	sort.Strings(allKeys)

	missing := make(map[string][]string)
	redundant := make(map[string][]string)

	for lang, kset := range langKeys {
		for _, k := range allKeys {
			if _, ok := kset[k]; !ok {
				missing[lang] = append(missing[lang], k)
			}
		}
		for k := range kset {
			if _, ok := allKeysSet[k]; !ok {
				redundant[lang] = append(redundant[lang], k)
			}
		}
	}

	// 新增：语法检查
	syntaxErrors := make(map[string]map[string]error)
	for _, file := range files {
		for key, msg := range file.Messages {
			if err := i18n.ValidateTemplate(msg); err != nil {
				if syntaxErrors[file.Language] == nil {
					syntaxErrors[file.Language] = make(map[string]error)
				}
				syntaxErrors[file.Language][key] = err
			}
		}
	}

	langs := make([]string, 0, len(langKeys))
	for l := range langKeys {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	return &Result{
		Languages:     langs,
		MissingKeys:   missing,
		RedundantKeys: redundant,
		SyntaxErrors:  syntaxErrors,
		AllKeys:       allKeys,
	}, nil
}

func scanYAML(dir string) ([]LangFile, error) {
	var res []LangFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		var lf LangFile
		if err := yaml.Unmarshal(data, &lf); err != nil {
			return fmt.Errorf("yaml error %s: %w", path, err)
		}
		if lf.Language == "" {
			return fmt.Errorf("file %s missing 'language' field", path)
		}

		res = append(res, lf)
		return nil
	})

	return res, err
}
