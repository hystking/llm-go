package version

// These variables can be overridden at build time using -ldflags.
// Example:
// go build -ldflags "-X llmx/pkg/version.Version=v0.1.0 -X llmx/pkg/version.Commit=$(git rev-parse --short HEAD) -X llmx/pkg/version.Date=$(date -u +%Y-%m-%d)"

var (
    Version = "dev"
    Commit  = ""
    Date    = ""
)

