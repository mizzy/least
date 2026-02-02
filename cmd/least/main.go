package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mizzy/least/internal/checker"
	"github.com/mizzy/least/internal/policy"
	"github.com/mizzy/least/internal/provider"
	"github.com/mizzy/least/internal/provider/terraform"
)

var version = "dev"

// registry holds all available IaC providers
var registry *provider.Registry

func init() {
	// Initialize provider registry
	registry = provider.NewRegistry()
	registry.Register(terraform.New())
	// Future providers can be registered here:
	// registry.Register(cloudformation.New())
	// registry.Register(pulumi.New())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "least",
	Short:   "Generate least-privilege IAM policies from IaC code",
	Long:    `least analyzes Infrastructure-as-Code configurations and generates minimal IAM policies required to manage the defined resources.`,
	Version: version,
}

var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate IAM policy from IaC files",
	Long:  `Analyze IaC files and generate a minimal IAM policy JSON.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runGenerate,
}

var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Check IAM policy against IaC requirements",
	Long:  `Compare an existing IAM policy against the minimal requirements from IaC files and report over/under permissions.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCheck,
}

var (
	outputFile   string
	policyFile   string
	policyDir    string
	format       string
	providerName string
)

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(checkCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&providerName, "provider", "", "IaC provider (auto-detected if not specified)")

	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	generateCmd.Flags().StringVarP(&format, "format", "f", "terraform", "Output format: terraform (or tf), json")

	checkCmd.Flags().StringVarP(&policyFile, "policy", "p", "", "Existing IAM policy JSON file")
	checkCmd.Flags().StringVarP(&policyDir, "policy-dir", "d", "", "Directory with IaC IAM policy definitions")
}

// getProvider returns the appropriate provider for the given path
func getProvider(path string) (provider.Provider, error) {
	if providerName != "" {
		p := registry.Get(providerName)
		if p == nil {
			return nil, fmt.Errorf("unknown provider: %s", providerName)
		}
		return p, nil
	}

	// Auto-detect provider
	providers, err := registry.Detect(path)
	if err != nil {
		return nil, fmt.Errorf("detecting provider: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no supported IaC files found in %s", path)
	}

	if len(providers) > 1 {
		names := make([]string, len(providers))
		for i, p := range providers {
			names[i] = p.Name()
		}
		fmt.Fprintf(os.Stderr, "Multiple providers detected: %v, using %s\n", names, providers[0].Name())
	}

	return providers[0], nil
}

func runGenerate(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	p, err := getProvider(path)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Using provider: %s\n", p.Name())
	fmt.Fprintf(os.Stderr, "Analyzing files in: %s\n", path)

	ctx := context.Background()
	result, err := p.Parse(ctx, path)
	if err != nil {
		return fmt.Errorf("parsing files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d resources\n", len(result.Resources))

	gen := policy.New()
	iamPolicy, err := gen.Generate(result.Resources)
	if err != nil {
		return fmt.Errorf("generating policy: %w", err)
	}

	var output string
	switch format {
	case "json":
		output, err = iamPolicy.ToJSON()
		if err != nil {
			return fmt.Errorf("converting policy to JSON: %w", err)
		}
	case "terraform", "tf":
		output = iamPolicy.ToTerraform()
	default:
		return fmt.Errorf("unsupported format: %s (use 'json' or 'terraform')", format)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Policy written to: %s\n", outputFile)
	} else {
		fmt.Println(output)
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

	p, err := getProvider(path)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Using provider: %s\n", p.Name())

	// Parse IaC files for required permissions
	ctx := context.Background()
	result, err := p.Parse(ctx, path)
	if err != nil {
		return fmt.Errorf("parsing files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d resources in: %s\n", len(result.Resources), path)

	// Generate required policy
	gen := policy.New()
	requiredPolicy, err := gen.Generate(result.Resources)
	if err != nil {
		return fmt.Errorf("generating required policy: %w", err)
	}

	// Load existing policy from either JSON file or IaC directory
	var existingPolicy *policy.IAMPolicy

	if policyDir != "" {
		policyProvider, err := getProvider(policyDir)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Loading IAM policies from %s: %s\n", policyProvider.Name(), policyDir)
		policyResult, err := policyProvider.Parse(ctx, policyDir)
		if err != nil {
			return fmt.Errorf("parsing IAM policies: %w", err)
		}
		if len(policyResult.Policies) == 0 {
			return fmt.Errorf("no IAM policies found in %s", policyDir)
		}
		fmt.Fprintf(os.Stderr, "Found %d IAM policy documents\n", len(policyResult.Policies))
		existingPolicy = policy.FromProviderPolicies(policyResult.Policies)
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
	checkResult := checker.Check(existingPolicy, requiredPolicy)

	// Output results
	if checkResult.IsCompliant() {
		fmt.Println("✓ Policy is compliant with least-privilege requirements")
		return nil
	}

	exitCode := 0

	if checkResult.HasMissing() {
		fmt.Println("✗ Missing permissions (required but not granted):")
		for _, action := range checkResult.Missing {
			fmt.Printf("  - %s\n", action)
		}
		exitCode = 1
	}

	if checkResult.HasExcessive() {
		fmt.Println("⚠ Excessive permissions (granted but not required):")
		for _, action := range checkResult.Excessive {
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
