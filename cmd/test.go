package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	testFile   string
	testMethod string
	testDebug  bool
)

var testCmd = &cobra.Command{
	Use:   "test [apps...]",
	Short: "Run tests in the backend container",
	Long: `Run tests using the configured test runner (default: pytest).

Examples:
  mf test                           # run all tests
  mf test application common        # run tests for specific apps
  mf test -f path/to/test.py        # run a specific test file
  mf test -m "TestClass.method"     # run a specific test method
  mf test -m "TestClass.method" -f path/to/test.py
  mf test --debug                   # run with debugpy (waits for VS Code attach)`,
	RunE: runTest,
}

func init() {
	testCmd.Flags().StringVarP(&testFile, "file", "f", "", "specific test file to run")
	testCmd.Flags().StringVarP(&testMethod, "method", "m", "", "specific test method (e.g. TestClass.test_method)")
	testCmd.Flags().BoolVar(&testDebug, "debug", false, "run with debugpy (waits for VS Code/Cursor attach)")

	// Register file completion for the --file flag
	testCmd.RegisterFlagCompletionFunc("file", completeTestFiles)

	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	service := cfg.Services.Backend
	if service == "" {
		return fmt.Errorf("no backend service configured — set services.backend in mf.yaml")
	}

	runner := cfg.Test.Runner
	if runner == "" {
		runner = "pytest"
	}

	// Build the test command
	var testArgs []string

	if testDebug {
		// Use debugpy to allow attaching a debugger
		debugPort := cfg.Test.DebugPort
		if debugPort == 0 {
			debugPort = 5679
		}
		testArgs = append(testArgs, "python", "-m", "debugpy",
			"--listen", fmt.Sprintf("0.0.0.0:%d", debugPort),
			"--wait-for-client", "-m", runner)
	} else {
		testArgs = append(testArgs, runner)
	}

	// Handle specific file + method combination
	if testMethod != "" && testFile != "" {
		// pytest format: file.py::TestClass::method
		methodPath := strings.ReplaceAll(testMethod, ".", "::")
		testArgs = append(testArgs, testFile+"::"+methodPath)
	} else if testFile != "" {
		testArgs = append(testArgs, testFile)
	} else if testMethod != "" {
		// Use -k filter when no file specified
		testArgs = append(testArgs, "-k", testMethod)
	} else if len(args) > 0 {
		// App names — append trailing slash for pytest discovery
		for _, app := range args {
			testArgs = append(testArgs, app+"/")
		}
	}

	// Pass configured environment variables
	env := cfg.Test.Env
	if len(env) == 0 {
		env = map[string]string{"ENV": "test"}
	}

	return comp.ExecWithEnv(service, env, testArgs...)
}
