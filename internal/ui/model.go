package ui

import (
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/bearded-giant/redis-tui/internal/cmd"
	"github.com/bearded-giant/redis-tui/internal/types"
	"github.com/kujtimiihoxha/vimtea"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Cmds              *cmd.Commands
	ScanSize          int64
	Version           string
	Screen            types.Screen
	Connections       []types.Connection
	SelectedConnIdx   int
	ConnInputs        []textinput.Model
	ConnFocusIdx      int
	EditingConnection *types.Connection
	CurrentConn       *types.Connection
	Keys              []types.RedisKey
	SelectedKeyIdx    int
	KeyCursor         uint64
	KeyPattern        string
	SearchSeq         int
	CurrentKey        *types.RedisKey
	CurrentValue      types.RedisValue
	AddKeyInputs      []textinput.Model
	AddKeyFocusIdx    int
	AddKeyType        types.KeyType
	ServerInfo        types.ServerInfo
	TotalKeys         int64
	Width             int
	Height            int
	Err               error
	StatusMsg         string
	Loading           bool
	ConfirmType       string
	ConfirmData       any
	Logs              *types.LogWriter
	SendFunc          *func(tea.Msg)
	PendingSelectKey  string

	// New fields for additional features
	VimEditor          vimtea.Editor
	EditingIndex       int
	EditingField       string
	AddCollectionInput []textinput.Model
	AddCollFocusIdx    int
	LuaResult          string
	PubSubInput        []textinput.Model
	PubSubFocusIdx     int
	PubSubMessages     []types.PubSubMessage
	PubSubChannels     []types.PubSubChannel
	SelectedChannelIdx int
	SlowLogEntries     []types.SlowLogEntry
	MemoryUsage        int64
	SelectedItemIdx    int
	SortBy             string
	SortAsc            bool
	TestConnResult     string
	LogCursor          int
	ShowingLogDetail   bool

	// Favorites and recent keys
	Favorites         []types.Favorite
	RecentKeys        []types.RecentKey
	SelectedFavIdx    int
	SelectedRecentIdx int

	// Tree view
	TreeNodes       []types.TreeNode
	TreeExpanded    map[string]bool
	TreeSeparator   string
	SelectedTreeIdx int

	// Bulk operations
	BulkDeletePreview []string
	BulkDeleteCount   int
	SelectedBulkKeys  map[string]bool

	// Batch TTL
	BatchTTLPreview []string

	// Search
	SearchResults []types.RedisKey

	// Watch mode
	WatchActive     bool
	WatchKey        string
	WatchValue      string
	WatchLastUpdate time.Time
	WatchInterval   time.Duration

	// Client list and memory stats
	ClientList        []types.ClientInfo
	MemoryStats       *types.MemoryStats
	SelectedClientIdx int

	// Cluster mode
	ClusterNodes    []types.ClusterNode
	ClusterEnabled  bool
	SelectedNodeIdx int
	ConnClusterMode bool

	// SSH tunnel form (sub-screen of add/edit connection)
	SSHInputs       []textinput.Model
	SSHFocusIdx     int
	SSHEnabled      bool
	PendingSSH      *types.SSHConfig // staged config from sub-screen
	SSHTunnelStatus string           // result message from TestSSHConnection

	// Compare keys
	CompareResult   *types.KeyComparison
	CompareFocusIdx int

	// Key templates
	Templates           []types.KeyTemplate
	SelectedTemplateIdx int
	TemplateInputs      []textinput.Model
	TemplateFocusIdx    int

	// JSON path query
	JSONPathResult string

	// Keybindings
	KeyBindings types.KeyBindings

	// Value history
	ValueHistory       []types.ValueHistoryEntry
	SelectedHistoryIdx int

	// Keyspace events
	KeyspaceEvents    []types.KeyspaceEvent
	KeyspaceSubActive bool
	KeyspacePattern   string

	// Connection groups
	ConnectionGroups []types.ConnectionGroup
	SelectedGroupIdx int

	// Expiring keys alerts
	ExpiringKeys    []types.RedisKey
	ExpiryThreshold int64 // seconds

	// Last tick time for accurate TTL counting
	LastTickTime time.Time

	// Key preview (shown in keys list)
	PreviewKey    string
	PreviewValue  types.RedisValue
	PreviewScroll int
	DetailScroll  int

	// Live metrics dashboard
	LiveMetrics       *types.LiveMetrics
	LiveMetricsActive bool

	// Redis config
	RedisConfigParams  []types.RedisConfigParam
	SelectedConfigIdx  int
	EditingConfigParam string

	// Connection error (for prominent display)
	ConnectionError string

	// CLI auto-connect (set when --host flag is provided)
	CLIConnection *types.Connection

	// Update notification
	UpdateAvailable string
	UpdateCmd       string

	// Text inputs — held behind a pointer so Model stays small enough to avoid
	// Go's large-object GC span class (>32KB), which triggered a race-detector
	// false-positive bad-pointer crash in Go 1.26.
	Inputs *ModelInputs

	// Lazy initialization flag
	inputsInitialized bool
}

type ModelInputs struct {
	PatternInput     textinput.Model
	TTLInput         textinput.Model
	RenameInput      textinput.Model
	CopyInput        textinput.Model
	SearchValueInput textinput.Model
	ExportInput      textinput.Model
	ImportInput      textinput.Model
	LuaScriptInput   textinput.Model
	DBSwitchInput    textinput.Model
	BulkDeleteInput  textinput.Model
	BatchTTLInput    textinput.Model
	BatchTTLPattern  textinput.Model
	RegexSearchInput textinput.Model
	FuzzySearchInput textinput.Model
	CompareKey1Input textinput.Model
	CompareKey2Input textinput.Model
	JSONPathInput    textinput.Model
	ConfigEditInput  textinput.Model
}

type ActionType string

const (
	ActionAdd    ActionType = "add"
	ActionEdit   ActionType = "edit"
	ActionDelete ActionType = "delete"
	ActionTest   ActionType = "test"
)

func NewModel() Model {
	return Model{
		Screen:             types.ScreenConnections,
		Connections:        []types.Connection{},
		ConnInputs:         createConnectionInputs(),
		SSHInputs:          createSSHInputs(),
		Keys:               []types.RedisKey{},
		AddKeyInputs:       createAddKeyInputs(),
		AddCollectionInput: createAddCollectionInputs(),
		PubSubInput:        createPubSubInputs(),
		AddKeyType:         types.KeyTypeString,
		SortBy:             "name",
		SortAsc:            true,
		TreeExpanded:       make(map[string]bool),
		TreeSeparator:      ":",
		SelectedBulkKeys:   make(map[string]bool),
		WatchInterval:      time.Second * 2,
		KeyBindings:        types.DefaultKeyBindings(),
		ExpiryThreshold:    300,
		Inputs: &ModelInputs{
			PatternInput:     createTextInput("Filter pattern...", 40),
			TTLInput:         createTextInput("TTL in seconds (-1 to remove)", 30),
			RenameInput:      createTextInput("New key name", 40),
			CopyInput:        createTextInput("New key name for copy", 40),
			SearchValueInput: createTextInput("Search in values...", 40),
			ExportInput:      createTextInput("Export filename", 40),
			ImportInput:      createTextInput("Import filename", 40),
			LuaScriptInput:   createTextInput("Lua script", 60),
			DBSwitchInput:    createTextInput("Database number (0-15)", 30),
			BulkDeleteInput:  createTextInput("Pattern to delete (e.g., user:*)", 40),
			BatchTTLInput:    createTextInput("TTL in seconds", 30),
			BatchTTLPattern:  createTextInput("Key pattern", 40),
			RegexSearchInput: createTextInput("Regex pattern", 40),
			FuzzySearchInput: createTextInput("Fuzzy search...", 40),
			CompareKey1Input: createTextInput("First key", 40),
			CompareKey2Input: createTextInput("Second key", 40),
			JSONPathInput:    createTextInput("JSONPath expression (e.g., $.name)", 40),
			ConfigEditInput:  createTextInput("New value", 50),
		},
		inputsInitialized: true,
	}
}

func createTextInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = width
	return ti
}

func createConnectionInputs() []textinput.Model {
	inputs := make([]textinput.Model, 6)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Connection Name"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Host"
	inputs[1].Width = 30
	inputs[1].SetValue("localhost")

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Port"
	inputs[2].Width = 30
	inputs[2].SetValue("6379")

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Username (optional)"
	inputs[3].Width = 30
	inputs[3].SetValue("default")

	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Password (optional)"
	inputs[4].Width = 30
	inputs[4].EchoMode = textinput.EchoPassword

	inputs[5] = textinput.New()
	inputs[5].Placeholder = "Database (0-15)"
	inputs[5].Width = 30
	inputs[5].SetValue("0")

	return inputs
}

// SSH form inputs:
// 0: SSH host
// 1: SSH port (22)
// 2: SSH user
// 3: Private key path
// 4: Passphrase (echo masked)
// 5: Password (echo masked)
// 6: Local port (0 = ephemeral)
func createSSHInputs() []textinput.Model {
	inputs := make([]textinput.Model, 7)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Bastion Host"
	inputs[0].Width = 30
	inputs[0].Focus()

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Bastion Port"
	inputs[1].Width = 30
	inputs[1].SetValue("22")

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "SSH User"
	inputs[2].Width = 30

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Private Key Path (optional)"
	inputs[3].Width = 30

	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Passphrase (optional)"
	inputs[4].Width = 30
	inputs[4].EchoMode = textinput.EchoPassword

	inputs[5] = textinput.New()
	inputs[5].Placeholder = "SSH Password (optional)"
	inputs[5].Width = 30
	inputs[5].EchoMode = textinput.EchoPassword

	inputs[6] = textinput.New()
	inputs[6].Placeholder = "Local Port (0 = ephemeral)"
	inputs[6].Width = 30
	inputs[6].SetValue("0")

	return inputs
}

func createAddKeyInputs() []textinput.Model {
	inputs := make([]textinput.Model, 3)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Key Name"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Value"
	inputs[1].Width = 30

	// Third input: Field name (hash/stream) or Score (zset)
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Field"
	inputs[2].Width = 30

	return inputs
}

func createAddCollectionInputs() []textinput.Model {
	inputs := make([]textinput.Model, 2)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Field/Member"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Value/Score"
	inputs[1].Width = 30

	return inputs
}

func createPubSubInputs() []textinput.Model {
	inputs := make([]textinput.Model, 2)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Channel"
	inputs[0].Focus()
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Message"
	inputs[1].Width = 30

	return inputs
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.Cmds.LoadConnections(),
		m.Cmds.CheckVersion(m.Version),
		func() tea.Msg { return tea.EnableBracketedPaste() },
		tickCmd(),
	}
	if m.CLIConnection != nil {
		conn := *m.CLIConnection
		cmds = append(cmds, func() tea.Msg {
			return types.AutoConnectMsg{Connection: conn}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) getPort() int {
	port, err := strconv.Atoi(m.ConnInputs[2].Value())
	if err != nil {
		return 6379
	}
	return port
}

func (m Model) getDB() int {
	db, err := strconv.Atoi(m.ConnInputs[5].Value())
	if err != nil {
		return 0
	}
	return db
}

func (m *Model) resetConnInputs() {
	for i := range m.ConnInputs {
		m.ConnInputs[i].SetValue("")
		m.ConnInputs[i].Blur()
	}
	m.ConnInputs[1].SetValue("localhost")
	m.ConnInputs[2].SetValue("6379")
	m.ConnInputs[5].SetValue("0")
	m.ConnInputs[0].Focus()
	m.ConnFocusIdx = 0
	m.ConnClusterMode = false
	m.SSHEnabled = false
	m.PendingSSH = nil
	m.resetSSHInputs()
}

func (m *Model) resetAddKeyInputs() {
	for i := range m.AddKeyInputs {
		m.AddKeyInputs[i].SetValue("")
		m.AddKeyInputs[i].Blur()
	}
	if len(m.AddKeyInputs) > 0 {
		m.AddKeyInputs[0].Focus()
	}
	m.AddKeyFocusIdx = 0
	m.AddKeyType = types.KeyTypeString
}

func (m *Model) populateConnInputs(conn types.Connection) {
	m.ConnInputs[0].SetValue(conn.Name)
	m.ConnInputs[1].SetValue(conn.Host)
	m.ConnInputs[2].SetValue(strconv.Itoa(conn.Port))
	m.ConnInputs[3].SetValue(conn.Username)
	m.ConnInputs[4].SetValue(conn.Password)
	m.ConnInputs[5].SetValue(strconv.Itoa(conn.DB))
	m.ConnClusterMode = conn.UseCluster
	m.SSHEnabled = conn.UseSSH
	if conn.SSHConfig != nil {
		cfg := *conn.SSHConfig
		m.PendingSSH = &cfg
	} else {
		m.PendingSSH = nil
	}
}

func (m *Model) resetSSHInputs() {
	for i := range m.SSHInputs {
		m.SSHInputs[i].SetValue("")
		m.SSHInputs[i].Blur()
	}
	m.SSHInputs[1].SetValue("22")
	m.SSHInputs[6].SetValue("0")
	m.SSHInputs[0].Focus()
	m.SSHFocusIdx = 0
	m.SSHTunnelStatus = ""
}

func (m *Model) populateSSHInputs(cfg *types.SSHConfig) {
	m.resetSSHInputs()
	if cfg == nil {
		return
	}
	m.SSHInputs[0].SetValue(cfg.Host)
	if cfg.Port != 0 {
		m.SSHInputs[1].SetValue(strconv.Itoa(cfg.Port))
	}
	m.SSHInputs[2].SetValue(cfg.User)
	m.SSHInputs[3].SetValue(cfg.PrivateKeyPath)
	m.SSHInputs[4].SetValue(cfg.Passphrase)
	m.SSHInputs[5].SetValue(cfg.Password)
	m.SSHInputs[6].SetValue(strconv.Itoa(cfg.LocalPort))
}

// convertSSHInputs builds an SSHConfig from current SSH form inputs.
// Returns nil if Host is empty (treated as "not configured").
func (m *Model) convertSSHInputs() *types.SSHConfig {
	host := m.SSHInputs[0].Value()
	if host == "" {
		return nil
	}
	port, _ := strconv.Atoi(m.SSHInputs[1].Value())
	if port == 0 {
		port = 22
	}
	localPort, _ := strconv.Atoi(m.SSHInputs[6].Value())
	return &types.SSHConfig{
		Host:           host,
		Port:           port,
		User:           m.SSHInputs[2].Value(),
		PrivateKeyPath: m.SSHInputs[3].Value(),
		Passphrase:     m.SSHInputs[4].Value(),
		Password:       m.SSHInputs[5].Value(),
		LocalPort:      localPort,
	}
}

// convertCurrentInputsToConnection converts the current inputs to a connection
func (m *Model) convertCurrentInputsToConnection(inputs []textinput.Model, action ActionType) types.Connection {
	var id int64
	if action == "edit" && m.EditingConnection != nil {
		id = m.EditingConnection.ID
	}

	port, _ := strconv.Atoi(inputs[2].Value())
	db, _ := strconv.Atoi(inputs[5].Value())
	return types.Connection{
		ID:         id,
		Name:       inputs[0].Value(),
		Port:       port,
		Host:       inputs[1].Value(),
		Username:   inputs[3].Value(),
		Password:   inputs[4].Value(),
		DB:         db,
		UseCluster: m.ConnClusterMode,
		UseSSH:     m.SSHEnabled,
		SSHConfig:  m.PendingSSH,
	}
}

// connFieldCount returns the number of focusable fields in the connection form.
// When cluster mode is on, the DB field is skipped (6 fields: name, host, port, username, password, cluster toggle).
// Otherwise there are 7 fields: name, host, port, username, password, cluster toggle, database.
func (m Model) connFieldCount() int {
	if m.ConnClusterMode {
		return 6
	}
	return 7
}

func (m *Model) resetAddCollectionInputs() {
	for i := range m.AddCollectionInput {
		m.AddCollectionInput[i].SetValue("")
		m.AddCollectionInput[i].Blur()
	}
	m.AddCollectionInput[0].Focus()
	m.AddCollFocusIdx = 0
}

func (m *Model) resetPubSubInputs() {
	for i := range m.PubSubInput {
		m.PubSubInput[i].SetValue("")
		m.PubSubInput[i].Blur()
	}
	m.PubSubInput[0].Focus()
	m.PubSubFocusIdx = 0
}
