package workspace

// overrideUserHomeDir temporarily overrides the userHomeDir function with the provided fn.
// It returns a cleanup function that restores the original userHomeDir implementation.
// This is useful for testing scenarios where the user's home directory needs to be mocked.
func overrideUserHomeDir(fn func() (string, error)) func() {
	old := userHomeDir
	userHomeDir = fn
	return func() { userHomeDir = old }
}

// overrideGOOS temporarily overrides the getGOOS function with the provided fn.
// It returns a restore function that, when called, restores getGOOS to its original value.
// This is useful for testing code that depends on the value returned by getGOOS.
func overrideGOOS(fn func() string) func() {
	old := getGOOS
	getGOOS = fn
	return func() { getGOOS = old }
}
