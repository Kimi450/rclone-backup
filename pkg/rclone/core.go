package rclone

import (
	"errors"
	"os"
	"os/exec"

	"github.com/ansel1/merry/v2"
	"github.com/go-logr/logr"
)

// Rclone struct holds the information regarding the rclone instance that will
// be used
type Rclone struct {
	// Logger
	Log logr.Logger

	// File path to the rclone binary
	Binary string

	// File path to the rclone config file
	Config string
}

// RunVersion runs the rclone version command with opinionated default arguments
func (rclone *Rclone) RunVersion() error {
	cmd := exec.Command(rclone.Binary, "version",
		"--config", rclone.Config)
	rclone.Log.Info("running command", "cmd", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return merry.Errorf("failed to run command: %s: %w", string(output), err)
	}
	rclone.Log.Info("finished running command", "cmd", cmd.String(),
		"output", string(output))

	return nil
}

// RunSync runs the rclone sync command with opinionated default arguments
func (rclone *Rclone) RunSync(logFileSyncPath, logFileSyncCombinedReportPath string,
	sourceDir, destDir string, extraSyncArgs []string) error {

	cmdArgs := []string{
		"sync",
		sourceDir, destDir,
		"--config", rclone.Config,
		"--use-json-log",
		"--log-level", "DEBUG",
		"--log-file", logFileSyncPath,
		"--combined", logFileSyncCombinedReportPath,
		"--check-first",
		"--metadata",
	}
	cmdArgs = append(cmdArgs, extraSyncArgs...)
	cmd := exec.Command(rclone.Binary, cmdArgs...)

	rclone.Log.Info("running command", "cmd", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return merry.Errorf("failed to run command with output [%s]: %w", string(output), err)
	}
	rclone.Log.Info("finished running command", "cmd", cmd.String())

	return nil
}

// RunLsjson runs the rclone lsjon command with opinionated default arguments
func (rclone *Rclone) RunLsjson(commandOutputLogFile *os.File, dir string) error {
	cmd := exec.Command(rclone.Binary, "lsjson",
		"--config", rclone.Config,
		"-R", dir)
	rclone.Log.Info("running command", "cmd", cmd.String())

	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return merry.Errorf("directory does not exist: %w", err)
	}

	cmd.Stderr = commandOutputLogFile
	cmd.Stdout = commandOutputLogFile
	// Has to be Run so we can redirect output to only the file
	// Too bulky otherwise
	err := cmd.Run()
	if err != nil {
		return merry.Errorf("failed to run command: %w", err)
	}
	rclone.Log.Info("finished running command", "cmd", cmd.String())

	return nil
}
