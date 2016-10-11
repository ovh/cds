package internal

// VERSION of CDS Engine
// injected at build time with -ldflags "-X ${PROJECT_PATH}/${PROJECT_NAME}/api/internal.version=${architecture}
var VERSION = "snapshot"
