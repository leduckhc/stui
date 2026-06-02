package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/natevick/stui/internal/aws"
	"github.com/natevick/stui/internal/bookmarks"
	"github.com/natevick/stui/internal/download"
	"github.com/natevick/stui/internal/providers"
	"github.com/natevick/stui/internal/views/bookmarksview"
	"github.com/natevick/stui/internal/views/browser"
	"github.com/natevick/stui/internal/views/buckets"
	downloadview "github.com/natevick/stui/internal/views/download"
	"github.com/natevick/stui/internal/views/profiles"
)

// Model is the root model for the TUI application
type Model struct {
	// AWS
	client        *aws.Client
	profile       string // profile/alias name (from --profile/--alias or picker)
	providerHint  string // explicit provider (from --provider or picker); "" = auto-resolve
	region        string
	endpoint      string // custom S3 endpoint (from --endpoint-url), overrides resolved
	initialBucket string // bucket to start in (from --bucket flag)
	demoMode      bool   // use mock data

	// Views
	activeView     ViewType
	profilesView   profiles.Model
	bucketsView    buckets.Model
	browserView    browser.Model
	downloadView   downloadview.Model
	bookmarksView  bookmarksview.Model
	showHelp       bool

	// State
	currentBucket string
	currentPrefix string
	bookmarkStore *bookmarks.Store
	downloadMgr   *download.Manager

	// UI
	styles       Styles
	keys         KeyMap
	width        int
	height       int
	statusMsg    string
	errorMsg     string
	errorTimeout time.Time

	// Prompt state
	showPrompt             bool
	promptType             string // "input" or "confirm"
	promptText             string
	promptInput            string
	promptDefault          string
	promptCursor           int
	pendingDownloadObjects []aws.S3Object // for multi-select downloads
	pendingBookmarkBucket  string         // for bucket bookmarks

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds configuration for the TUI
type Config struct {
	Profile  string // profile (aws/stui) or alias (minio) name
	Provider string // explicit provider: aws|minio|stui; "" = auto-resolve
	Region   string
	Bucket   string // Start directly in this bucket
	Endpoint string // Custom S3 endpoint URL (MinIO/SeaweedFS/etc.)
	DemoMode bool   // Use mock data instead of real AWS
}

// New creates a new TUI model
func New(cfg Config) Model {
	ctx, cancel := context.WithCancel(context.Background())

	// Determine initial view
	activeView := ViewBuckets
	if cfg.Bucket != "" {
		activeView = ViewBrowser
	} else if cfg.Profile == "" && !cfg.DemoMode {
		// No profile specified, show profile picker
		activeView = ViewProfiles
	}

	return Model{
		profile:       cfg.Profile,
		providerHint:  cfg.Provider,
		region:        cfg.Region,
		endpoint:      cfg.Endpoint,
		initialBucket: cfg.Bucket,
		demoMode:      cfg.DemoMode,
		activeView:    activeView,
		profilesView:  profiles.New(),
		bucketsView:   buckets.New(),
		browserView:   browser.New(),
		downloadView:  downloadview.New(),
		bookmarksView: bookmarksview.New(),
		styles:        DefaultStyles(),
		keys:          DefaultKeyMap(),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.demoMode {
		return tea.Batch(
			m.initDemo(),
			m.initBookmarks(),
			tea.SetWindowTitle("S3 TUI (Demo)"),
		)
	}

	// If no profile specified, load profile picker
	if m.profile == "" {
		return tea.Batch(
			m.initProfiles(),
			m.initBookmarks(),
			tea.SetWindowTitle("S3 TUI"),
		)
	}

	return tea.Batch(
		m.initAWS(),
		m.initBookmarks(),
		tea.SetWindowTitle("S3 TUI"),
	)
}

// initProfiles loads available profiles
func (m Model) initProfiles() tea.Cmd {
	return func() tea.Msg {
		return profilesReadyMsg{}
	}
}

// profilesReadyMsg is sent when profiles should be loaded
type profilesReadyMsg struct{}

// initDemo initializes with mock data
func (m Model) initDemo() tea.Cmd {
	return func() tea.Msg {
		return demoReadyMsg{}
	}
}

// demoReadyMsg is sent when demo mode is ready
type demoReadyMsg struct{}

// initAWS initializes the AWS client by resolving the requested profile/alias
// across the configured providers (aws, minio, stui). There is never any
// fallback between providers; an ambiguous name produces a clear error asking
// the user to pass --provider.
func (m Model) initAWS() tea.Cmd {
	return func() tea.Msg {
		entry, err := providers.Resolve(m.profile, m.providerHint)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		opts := entry.ClientOptions()
		// CLI overrides.
		if m.region != "" {
			opts.Region = m.region
		}
		if m.endpoint != "" {
			opts.Endpoint = m.endpoint
		}

		client, err := aws.NewClientWithOptions(m.ctx, opts)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return awsClientReadyMsg{client: client}
	}
}

// awsClientReadyMsg is sent when AWS client is ready
type awsClientReadyMsg struct {
	client *aws.Client
}

// initBookmarks initializes the bookmark store
func (m Model) initBookmarks() tea.Cmd {
	return func() tea.Msg {
		store, err := bookmarks.NewStore()
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return bookmarkStoreReadyMsg{store: store}
	}
}

// bookmarkStoreReadyMsg is sent when bookmark store is ready
type bookmarkStoreReadyMsg struct {
	store *bookmarks.Store
}

// SetSize sets the terminal size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Reserve space for header, tabs, and status bar
	contentHeight := height - 6

	m.profilesView.SetSize(width-2, contentHeight)
	m.bucketsView.SetSize(width-2, contentHeight)
	m.browserView.SetSize(width-2, contentHeight)
	m.downloadView.SetSize(width-2, contentHeight)
	m.bookmarksView.SetSize(width-2, contentHeight)
}

// loadBuckets returns a command to load buckets
func (m Model) loadBuckets() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return ErrorMsg{Err: nil}
		}
		bucketList, err := m.client.ListBuckets(m.ctx)
		if err != nil {
			return BucketsLoadedMsg{Err: err}
		}
		return BucketsLoadedMsg{Buckets: bucketList}
	}
}

// loadObjects returns a command to load objects at the current prefix
func (m Model) loadObjects() tea.Cmd {
	if m.demoMode {
		return m.loadDemoObjects()
	}
	return func() tea.Msg {
		if m.client == nil || m.currentBucket == "" {
			return nil
		}
		objects, err := m.client.ListObjects(m.ctx, m.currentBucket, m.currentPrefix)
		if err != nil {
			return ObjectsLoadedMsg{Err: err}
		}
		return ObjectsLoadedMsg{Objects: objects, Prefix: m.currentPrefix}
	}
}

// startDownload starts a download operation
func (m Model) startDownload(key, localPath string, isPrefix bool) tea.Cmd {
	return func() tea.Msg {
		if m.downloadMgr == nil || m.client == nil {
			return ErrorMsg{Err: nil}
		}

		// Set up progress callback
		progressChan := make(chan download.Progress, 10)
		m.downloadMgr.SetProgressCallback(func(p download.Progress) {
			select {
			case progressChan <- p:
			default:
			}
		})

		go func() {
			var err error
			if isPrefix {
				err = m.downloadMgr.DownloadPrefix(m.ctx, m.currentBucket, key, localPath)
			} else {
				err = m.downloadMgr.DownloadFile(m.ctx, m.currentBucket, key, localPath)
			}
			if err != nil {
				progressChan <- download.Progress{Status: download.StatusFailed}
			}
			close(progressChan)
		}()

		return downloadStartedMsg{progressChan: progressChan}
	}
}

// downloadStartedMsg is sent when a download starts
type downloadStartedMsg struct {
	progressChan <-chan download.Progress
}

// startMultiDownload starts downloading multiple objects
func (m Model) startMultiDownload(objects []aws.S3Object, localDir string) tea.Cmd {
	return func() tea.Msg {
		if m.downloadMgr == nil || m.client == nil {
			return ErrorMsg{Err: nil}
		}

		// Set up progress callback
		progressChan := make(chan download.Progress, 10)
		m.downloadMgr.SetProgressCallback(func(p download.Progress) {
			select {
			case progressChan <- p:
			default:
			}
		})

		go func() {
			// Convert to aws.S3Object slice for the download manager
			err := m.downloadMgr.DownloadMultiple(m.ctx, m.currentBucket, objects, m.currentPrefix, localDir)
			if err != nil {
				progressChan <- download.Progress{Status: download.StatusFailed}
			}
			close(progressChan)
		}()

		return downloadStartedMsg{progressChan: progressChan}
	}
}

// tickCmd returns a command that ticks periodically
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

// Demo mode mock data

func (m Model) loadDemoBuckets() tea.Cmd {
	return func() tea.Msg {
		buckets := []aws.Bucket{
			{Name: "demo-bucket-1", CreationDate: time.Now().AddDate(0, -6, 0)},
			{Name: "demo-bucket-2", CreationDate: time.Now().AddDate(0, -3, 0)},
			{Name: "demo-data-exports", CreationDate: time.Now().AddDate(-1, 0, 0)},
			{Name: "demo-logs", CreationDate: time.Now().AddDate(0, -1, 0)},
			{Name: "demo-backups", CreationDate: time.Now().AddDate(-2, 0, 0)},
		}
		return BucketsLoadedMsg{Buckets: buckets}
	}
}

func (m Model) loadDemoObjects() tea.Cmd {
	return func() tea.Msg {
		var objects []aws.S3Object

		if m.currentPrefix == "" {
			// Root level - show folders
			objects = []aws.S3Object{
				{Key: "2024-01-01/", IsPrefix: true},
				{Key: "2024-01-02/", IsPrefix: true},
				{Key: "2024-01-03/", IsPrefix: true},
				{Key: "config.json", Size: 1024, LastModified: time.Now().AddDate(0, 0, -1), ETag: "abc123"},
				{Key: "readme.txt", Size: 256, LastModified: time.Now().AddDate(0, 0, -7), ETag: "def456"},
			}
		} else {
			// Inside a folder - show files
			objects = []aws.S3Object{
				{Key: m.currentPrefix + "data-001.parquet", Size: 1024 * 1024 * 50, LastModified: time.Now().AddDate(0, 0, -1), ETag: "file1"},
				{Key: m.currentPrefix + "data-002.parquet", Size: 1024 * 1024 * 75, LastModified: time.Now().AddDate(0, 0, -1), ETag: "file2"},
				{Key: m.currentPrefix + "data-003.parquet", Size: 1024 * 1024 * 25, LastModified: time.Now().AddDate(0, 0, -1), ETag: "file3"},
				{Key: m.currentPrefix + "metadata.json", Size: 2048, LastModified: time.Now().AddDate(0, 0, -1), ETag: "meta1"},
			}
		}

		return ObjectsLoadedMsg{Objects: objects, Prefix: m.currentPrefix}
	}
}

