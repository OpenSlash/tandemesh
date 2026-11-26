package main

// inheritLoginShellPath is a no-op on Windows: GUI apps already get the
// user's PATH from the registry.
func inheritLoginShellPath() {}
