package main

import (
	"fmt"
	"regexp"
)

// maskCredentials masks passwords and sensitive information in URLs and commands
func maskCredentials(text string) string {
	// Regex to match URLs with credentials: protocol://user:password@host/path
	credentialURLRegex := regexp.MustCompile(`(https?://)([^:]+):([^@]+)(@[^/\s]+)`)
	masked := credentialURLRegex.ReplaceAllString(text, "${1}${2}:***${4}")

	// Also mask any standalone passwords that might appear
	// This is a more generic approach for any password-like strings
	passwordRegex := regexp.MustCompile(`(\bpassword[=:\s]+)([^\s&]+)`)
	masked = passwordRegex.ReplaceAllString(masked, "${1}***")

	return masked
}

func main() {
	// Test cases
	testCases := []string{
		"https://bilgehan.nal:Blg%28245434268%29@bitbucket.int.sahibinden.com/scm/misc/jenkins-devops-libs.git",
		"git remote set-url origin https://user:password123@github.com/user/repo.git",
		"https://user:complex%40password@example.com/path",
		"password: mySecretPassword",
		"normal text without credentials",
	}

	fmt.Println("Testing credential masking functionality:")
	fmt.Println("=========================================")

	for i, testCase := range testCases {
		masked := maskCredentials(testCase)
		fmt.Printf("Test %d:\n", i+1)
		fmt.Printf("Original: %s\n", testCase)
		fmt.Printf("Masked:   %s\n", masked)
		fmt.Println()
	}
}
