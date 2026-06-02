// Package providers resolves a connection target across the three independent
// configuration sources stui understands:
//
//   - "aws"   : AWS profiles in ~/.aws/config (+ ~/.aws/credentials)
//   - "minio" : mc aliases in ~/.mc/config.json (or $MC_CONFIG_DIR)
//   - "stui"  : endpoints in ~/.config/stui/config.json
//
// Each provider is independent: there is never any fallback from one provider
// to another. A name (an AWS "profile" or an mc "alias") is looked up within a
// single provider only. If the same name exists in more than one provider, the
// user must disambiguate with an explicit provider.
package providers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/natevick/stui/internal/aws"
)

// Provider identifiers.
const (
	AWS   = "aws"
	MinIO = "minio"
	Stui  = "stui"
)

// All lists every supported provider, in resolution order.
var All = []string{AWS, MinIO, Stui}

// Valid reports whether p is a known provider id.
func Valid(p string) bool {
	for _, x := range All {
		if x == p {
			return true
		}
	}
	return false
}

// Entry is a resolved connection target from one provider.
type Entry struct {
	Provider string // AWS | MinIO | Stui
	Name     string // profile (aws/stui) or alias (minio)

	Region   string
	Endpoint string // custom S3 endpoint; "" for real AWS

	// Static credentials (minio/stui). Empty for aws, which uses the SDK
	// credential chain for the named profile.
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string

	PathStyle *bool // nil = auto (path-style when Endpoint set)

	// AWS-only display metadata.
	SSOSession string
	AccountID  string
}

// ClientOptions maps an entry to AWS client options. The aws provider loads the
// named shared profile (with its own credential chain); the others are
// self-contained endpoints with their own static credentials.
func (e Entry) ClientOptions() aws.ClientOptions {
	if e.Provider == AWS {
		return aws.ClientOptions{Profile: e.Name, Region: e.Region}
	}
	return aws.ClientOptions{
		Region:          e.Region,
		Endpoint:        e.Endpoint,
		PathStyle:       e.PathStyle,
		AccessKeyID:     e.AccessKeyID,
		SecretAccessKey: e.SecretAccessKey,
		SessionToken:    e.SessionToken,
	}
}

// AmbiguousError is returned when a name matches more than one provider and no
// explicit provider was given.
type AmbiguousError struct {
	Name      string
	Providers []string
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("%q matches multiple providers (%s); specify one with --provider <%s>",
		e.Name, strings.Join(e.Providers, ", "), strings.Join(e.Providers, "|"))
}

// entriesFor returns the entries for a single provider. A missing config file
// yields no entries and no error.
func entriesFor(provider string) ([]Entry, error) {
	switch provider {
	case AWS:
		return awsEntries()
	case MinIO:
		return minioEntries()
	case Stui:
		return stuiEntries()
	default:
		return nil, fmt.Errorf("unknown provider %q (valid: %s)", provider, strings.Join(All, ", "))
	}
}

// List returns every entry across all providers. It is best effort: entries
// from healthy providers are always returned, and any per-provider load errors
// (e.g. a malformed ~/.mc/config.json) are joined into the returned error so
// the caller can surface them without hiding the working providers.
func List() ([]Entry, error) {
	var out []Entry
	var errs []error
	for _, p := range All {
		entries, err := entriesFor(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
			continue
		}
		out = append(out, entries...)
	}
	return out, errors.Join(errs...)
}

// Resolve finds the entry for name. When providerHint is non-empty, only that
// provider is consulted (no cross-provider fallback ever). Otherwise every
// provider is searched; a name present in multiple providers is an
// *AmbiguousError.
func Resolve(name, providerHint string) (Entry, error) {
	if name == "" {
		return Entry{}, fmt.Errorf("no profile/alias specified")
	}

	if providerHint != "" {
		if !Valid(providerHint) {
			return Entry{}, fmt.Errorf("unknown provider %q (valid: %s)", providerHint, strings.Join(All, ", "))
		}
		entries, err := entriesFor(providerHint)
		if err != nil {
			return Entry{}, err
		}
		for _, e := range entries {
			if e.Name == name {
				return e, nil
			}
		}
		return Entry{}, fmt.Errorf("profile/alias %q not found in provider %q", name, providerHint)
	}

	var matches []Entry
	var errs []error
	for _, p := range All {
		entries, err := entriesFor(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
			continue
		}
		for _, e := range entries {
			if e.Name == name {
				matches = append(matches, e)
			}
		}
	}

	switch len(matches) {
	case 0:
		if len(errs) > 0 {
			// Don't claim "not found" when a provider failed to load — the name
			// might live in the broken config.
			return Entry{}, fmt.Errorf("profile/alias %q not found in any readable provider (%s); some providers failed to load: %w",
				name, strings.Join(All, ", "), errors.Join(errs...))
		}
		return Entry{}, fmt.Errorf("profile/alias %q not found in any provider (%s)", name, strings.Join(All, ", "))
	case 1:
		return matches[0], nil
	default:
		provs := make([]string, len(matches))
		for i, m := range matches {
			provs[i] = m.Provider
		}
		return Entry{}, &AmbiguousError{Name: name, Providers: provs}
	}
}

func awsEntries() ([]Entry, error) {
	profiles, err := aws.ListProfiles()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, len(profiles))
	for i, p := range profiles {
		entries[i] = Entry{
			Provider:   AWS,
			Name:       p.Name,
			Region:     p.Region,
			SSOSession: p.SSOSession,
			AccountID:  p.AccountID,
		}
	}
	return entries, nil
}
