package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/natevick/stui/internal/providers"
	"github.com/natevick/stui/internal/security"
	"github.com/natevick/stui/internal/tui"
)

var (
	version = "dev"
)

func main() {
	// Parse flags
	profile := flag.String("profile", os.Getenv("AWS_PROFILE"), "Profile (aws/stui) or alias (minio) name to use (also AWS_PROFILE env var)")
	alias := flag.String("alias", "", "Alias name (synonym of --profile, matching mc terminology)")
	provider := flag.String("provider", "", "Force a provider: aws|minio|stui (skips cross-provider matching)")
	region := flag.String("region", os.Getenv("AWS_REGION"), "AWS region (can also use AWS_REGION env var)")
	bucket := flag.String("bucket", "", "Start directly in this S3 bucket")
	endpoint := flag.String("endpoint-url", "", "Custom S3 endpoint URL (for MinIO, SeaweedFS, Ceph, etc.; overrides resolved)")
	demo := flag.Bool("demo", false, "Run with mock data (no AWS credentials needed)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("stui version %s\n", version)
		os.Exit(0)
	}

	// --profile and --alias are synonyms (one names the same target). Reject a
	// conflicting pair rather than silently picking one.
	name := *profile
	if *alias != "" {
		if name != "" && name != *alias {
			fmt.Fprintf(os.Stderr, "--profile and --alias are synonyms; specify only one\n")
			os.Exit(1)
		}
		name = *alias
	}

	if *provider != "" && !providers.Valid(*provider) {
		fmt.Fprintf(os.Stderr, "Invalid provider %q (valid: aws, minio, stui)\n", *provider)
		os.Exit(1)
	}

	// --provider only takes effect together with a name; on its own the picker
	// runs and would ignore it. Fail loudly instead of silently dropping it.
	if *provider != "" && name == "" {
		fmt.Fprintf(os.Stderr, "--provider requires --profile/--alias\n")
		os.Exit(1)
	}

	// Validate inputs
	if err := security.ValidProfileName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid profile/alias: %v\n", err)
		os.Exit(1)
	}
	if err := security.ValidBucketName(*bucket); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid bucket: %v\n", err)
		os.Exit(1)
	}

	// Create TUI model
	cfg := tui.Config{
		Profile:  name,
		Provider: *provider,
		Region:   *region,
		Bucket:   *bucket,
		Endpoint: *endpoint,
		DemoMode: *demo,
	}

	model := tui.New(cfg)

	// Create and run program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
