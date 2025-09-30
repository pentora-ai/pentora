package scanexec

// Params defines the input required to initiate a scan run.
type Params struct {
	Targets       []string
	Profile       string
	Level         string
	IncludeTags   []string
	ExcludeTags   []string
	EnableVuln    bool
	Ports         string
	CustomTimeout string
	EnablePing    bool
	PingCount     int
	AllowLoopback bool
	Concurrency   int
	WorkspaceDir  string
	OutputFormat  string
	RawInputs     map[string]interface{}
	OnlyDiscover  bool
	SkipDiscover  bool
}

// Result is a placeholder for structured scan outputs.
type Result struct {
	RunID      string
	StartTime  string
	EndTime    string
	Status     string
	Findings   interface{}
	RawContext map[string]interface{}
}
