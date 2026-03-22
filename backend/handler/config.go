package handler

// AppMode controls authentication requirements
type AppMode string

const (
	ModeTE AppMode = "te" // Test mode: no wallet required
	ModePE AppMode = "pe" // Production mode: wallet required
)

// Mode is the current application mode, set by main.go from APP_MODE env var
var Mode AppMode = ModeTE
