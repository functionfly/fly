package flypy

// Version is the current FlyPy compiler version.
// It is set at build time via ldflags:
//
//	go build -ldflags "-X github.com/functionfly/fly/internal/flypy.Version=1.2.3"
//
// Falls back to "dev" if not set.
var Version = "dev"
