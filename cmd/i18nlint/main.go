package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lifei6671/i18n/cmd/i18nlint/checker"
)

func main() {
	dir := flag.String("d", "./i18n/locales", "directory of YAML locale files")
	failOnError := flag.Bool("fail", false, "exit with code 1 if any issue found")
	flag.Parse()

	res, err := checker.CheckLocales(*dir)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	printResult(res)

	if *failOnError && (hasIssues(res)) {
		os.Exit(1)
	}
}

func printResult(res *checker.Result) {
	fmt.Println("=== I18N CHECK RESULT ===")
	fmt.Println("Languages:", res.Languages)
	fmt.Println("Total keys:", len(res.AllKeys))

	for _, lang := range res.Languages {
		fmt.Printf("\n--- [%s] ---\n", lang)

		// missing keys
		if arr := res.MissingKeys[lang]; len(arr) > 0 {
			fmt.Println("Missing keys:")
			for _, k := range arr {
				fmt.Println("  -", k)
			}
		} else {
			fmt.Println("Missing keys: None")
		}

		// redundant
		if arr := res.RedundantKeys[lang]; len(arr) > 0 {
			fmt.Println("Redundant keys:")
			for _, k := range arr {
				fmt.Println("  -", k)
			}
		} else {
			fmt.Println("Redundant keys: None")
		}

		// syntax errors
		if errs := res.SyntaxErrors[lang]; len(errs) > 0 {
			fmt.Println("Syntax errors:")
			for key, err := range errs {
				fmt.Printf("  - %s: %v\n", key, err)
			}
		} else {
			fmt.Println("Syntax errors: None")
		}
	}
}

func hasIssues(res *checker.Result) bool {
	for _, arr := range res.MissingKeys {
		if len(arr) > 0 {
			return true
		}
	}
	for _, arr := range res.RedundantKeys {
		if len(arr) > 0 {
			return true
		}
	}
	for _, errs := range res.SyntaxErrors {
		if len(errs) > 0 {
			return true
		}
	}
	return false
}
