package core

// CommandName holds the active pipeline subcommand for AI failure context.
// Commands set this at the start of Run; ExecCommand reads it on failure.
var CommandName string
