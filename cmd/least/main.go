package main

import (
	"fmt"
	"os"

	"github.com/mizzy/least/internal/checker"
	"github.com/mizzy/least/internal/parser"
	"github.com/mizzy/least/internal/policy"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "least",
	Short:   "Generate least-privilege IAM policies from Terraform code",
	Long:    `least analyzes Terraform configurations and generates minimal IAM policies required to manage the defined resources.`,
	Version: version,
}

var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate IAM policy from Terraform files",
	Long:  `Analyze Terraform files and generate a minimal IAM policy JSON.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runGenerate,
}

var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Check IAM policy against Terraform requirements",
	Long:  `Compare an existing IAM policy against the minimal requirements from Terraform files and report over/under permissions.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCheck,
}

var (
	outputFile string
	policyFile string
	policyDir  string
	format     string
)

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(checkCmd)

	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	generateCmd.Flags().StringVarP(&format, "format", "f", "json", "Output format (json, terraform)")

	checkCmd.Flags().StringVarP(&policyFile, "policy", "p", "", "Existing IAM policy JSON file")
	checkCmd.Flags().StringVarP(&policyDir, "policy-dir", "d", "", "Directory with Terraform IAM policy definitions")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	fmt.Fprintf(os.Stderr, "Analyzing Terraform files in: %s\n", path)

	p := parser.New()
	resources, err := p.ParseDirectory(path)
	if err != nil {
		return fmt.Errorf("parsing terraform files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d resources\n", len(resources))

	gen := policy.New()
	iamPolicy, err := gen.Generate(resources)
	if err != nil {
		return fmt.Errorf("generating policy: %w", err)
	}

	jsonStr, err := iamPolicy.ToJSON()
	if err != nil {
		return fmt.Errorf("converting policy to JSON: %w", err)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(jsonStr), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Policy written to: %s\n", outputFile)
	} else {
		fmt.Println(jsonStr)
	}

	return nil
}

func runCheck(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	if policyFile == "" && policyDir == "" {
		return fmt.Errorf("either --policy or --policy-dir must be specified")
	}

	// Parse Terraform files for required permissions
	p := parser.New()
	resources, err := p.ParseDirectory(path)
	if err != nil {
		return fmt.Errorf("parsing terraform files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d resources in: %s\n", len(resources), path)

	// Generate required policy
	gen := policy.New()
	requiredPolicy, err := gen.Generate(resources)
	if err != nil {
		return fmt.Errorf("generating required policy: %w", err)
	}

	// Load existing policy from either JSON file or Terraform directory
	var existingPolicy *policy.IAMPolicy

	if policyDir != "" {
		fmt.Fprintf(os.Stderr, "Loading IAM policies from Terraform: %s\n", policyDir)
		iamDocs, err := p.ParseIAMPolicies(policyDir)
		if err != nil {
			return fmt.Errorf("parsing IAM policies: %w", err)
		}
		if len(iamDocs) == 0 {
			return fmt.Errorf("no IAM policies found in %s", policyDir)
		}
		fmt.Fprintf(os.Stderr, "Found %d IAM policy documents\n", len(iamDocs))
		existingPolicy = policy.FromParsedPolicies(iamDocs)
	} else {
		fmt.Fprintf(os.Stderr, "Loading IAM policy from JSON: %s\n", policyFile)
		existingData, err := os.ReadFile(policyFile)
		if err != nil {
			return fmt.Errorf("reading policy file: %w", err)
		}
		existingPolicy, err = policy.ParsePolicy(existingData)
		if err != nil {
			return fmt.Errorf("parsing existing policy: %w", err)
		}
	}

	// Check policies
	result := checker.Check(existingPolicy, requiredPolicy)

	// Output results
	if result.IsCompliant() {
		fmt.Println("✓ Policy is compliant with least-privilege requirements")
		return nil
	}

	exitCode := 0

	if result.HasMissing() {
		fmt.Println("✗ Missing permissions (required but not granted):")
		for _, action := range result.Missing {
			fmt.Printf("  - %s\n", action)
		}
		exitCode = 1
	}

	if result.HasExcessive() {
		fmt.Println("⚠ Excessive permissions (granted but not required):")
		for _, action := range result.Excessive {
			fmt.Printf("  + %s\n", action)
		}
		if exitCode == 0 {
			exitCode = 2
		}
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
