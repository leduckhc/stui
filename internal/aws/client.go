package aws

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps the AWS S3 client with configuration
type Client struct {
	S3       *s3.Client
	Config   aws.Config
	Profile  string
	Region   string
	Endpoint string

	// opts is retained so WithRegion can rebuild an equivalent client
	// (preserving endpoint, path-style and static credentials).
	opts ClientOptions
}

// ClientOptions configures how the AWS/S3 client is built.
type ClientOptions struct {
	Profile  string
	Region   string
	Endpoint string // custom S3-compatible endpoint URL; "" for real AWS
	// PathStyle forces path-style addressing. If nil, it defaults to true
	// whenever a custom Endpoint is in effect (required by most S3-compatible
	// servers).
	PathStyle *bool
	// Static credentials. When AccessKeyID and SecretAccessKey are both set,
	// they are used directly instead of the AWS profile / default credential
	// chain. SessionToken is optional (for temporary credentials).
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// NewClient creates a new AWS client with the specified profile.
// Supports SSO profiles - user must run `aws sso login --profile <profile>` first.
func NewClient(ctx context.Context, profile, region string) (*Client, error) {
	return NewClientWithOptions(ctx, ClientOptions{Profile: profile, Region: region})
}

// NewClientWithOptions creates a new AWS client, optionally pointed at a custom
// S3-compatible endpoint (SeaweedFS, MinIO, Ceph, etc.).
//
// Region and SSO are loaded from the standard AWS config (~/.aws/config,
// ~/.aws/credentials) via the SDK. Credentials come from opts.AccessKeyID /
// opts.SecretAccessKey when both are set (used directly, bypassing any AWS
// profile); otherwise from the named profile / SDK default chain. The custom
// endpoint is supplied by stui's own config (see internal/config), an mc alias
// (see internal/providers), or the --endpoint-url flag; stui never extends the
// AWS config schema itself.
//
// The endpoint actually used is, in order of precedence: opts.Endpoint, then
// anything the SDK itself resolved from standard config or AWS_ENDPOINT_URL[_S3]
// env vars. When any custom endpoint is in effect, path-style addressing is
// enabled unless opts.PathStyle explicitly disables it.
func NewClientWithOptions(ctx context.Context, opts ClientOptions) (*Client, error) {
	var loadOpts []func(*config.LoadOptions) error

	useStaticCreds := opts.AccessKeyID != "" && opts.SecretAccessKey != ""

	// With static credentials the endpoint is self-contained (the "profile" may
	// exist only in stui's config, not in ~/.aws), so don't try to load a shared
	// AWS profile that might not exist.
	if opts.Profile != "" && !useStaticCreds {
		loadOpts = append(loadOpts, config.WithSharedConfigProfile(opts.Profile))
	}
	if opts.Region != "" {
		loadOpts = append(loadOpts, config.WithRegion(opts.Region))
	}

	// Use static credentials from stui config when provided, bypassing the AWS
	// profile / default credential chain.
	if useStaticCreds {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				opts.AccessKeyID, opts.SecretAccessKey, opts.SessionToken,
			),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
		}
	})

	// Determine the endpoint actually in effect (explicit, or SDK-resolved from
	// AWS_ENDPOINT_URL[_S3] env vars).
	resolvedEndpoint := opts.Endpoint
	if resolvedEndpoint == "" && s3Client.Options().BaseEndpoint != nil {
		resolvedEndpoint = *s3Client.Options().BaseEndpoint
	}

	// Decide on path-style addressing: explicit override, else default to true
	// whenever a custom endpoint is in effect.
	pathStyle := s3Client.Options().UsePathStyle
	if opts.PathStyle != nil {
		pathStyle = *opts.PathStyle
	} else if resolvedEndpoint != "" {
		pathStyle = true
	}

	if resolvedEndpoint != "" && pathStyle != s3Client.Options().UsePathStyle {
		s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(resolvedEndpoint)
			o.UsePathStyle = pathStyle
		})
	}

	return &Client{
		S3:       s3Client,
		Config:   cfg,
		Profile:  opts.Profile,
		Region:   cfg.Region,
		Endpoint: resolvedEndpoint,
		opts:     opts,
	}, nil
}

// WithRegion creates a new client with a different region, preserving the
// original endpoint, path-style and credential settings.
func (c *Client) WithRegion(ctx context.Context, region string) (*Client, error) {
	opts := c.opts
	opts.Region = region
	return NewClientWithOptions(ctx, opts)
}

// ProfileInfo contains information about an AWS profile
type ProfileInfo struct {
	Name       string
	Region     string
	SSOSession string
	AccountID  string
}

// ListProfiles returns a list of available AWS profiles from ~/.aws/config
func ListProfiles() ([]ProfileInfo, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".aws", "config")
	file, err := os.Open(configPath)
	if err != nil {
		// No AWS config is fine: the user may rely solely on stui's own config
		// (e.g. a self-contained MinIO/SeaweedFS endpoint).
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open AWS config: %w", err)
	}
	defer file.Close()

	var profiles []ProfileInfo
	var currentProfile *ProfileInfo

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Save previous profile if it exists
			if currentProfile != nil {
				profiles = append(profiles, *currentProfile)
			}

			section := strings.TrimPrefix(strings.TrimSuffix(line, "]"), "[")

			// Skip sso-session sections, only get profiles
			if strings.HasPrefix(section, "sso-session ") {
				currentProfile = nil
				continue
			}

			// Extract profile name
			name := section
			if strings.HasPrefix(section, "profile ") {
				name = strings.TrimPrefix(section, "profile ")
			}

			currentProfile = &ProfileInfo{Name: name}
			continue
		}

		// Parse key-value pairs for current profile
		if currentProfile != nil && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "region":
					currentProfile.Region = value
				case "sso_session":
					currentProfile.SSOSession = value
				case "sso_account_id":
					currentProfile.AccountID = value
				}
			}
		}
	}

	// Don't forget the last profile
	if currentProfile != nil {
		profiles = append(profiles, *currentProfile)
	}

	return profiles, scanner.Err()
}
