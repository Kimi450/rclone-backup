package rclone

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ansel1/merry/v2"
	"github.com/go-logr/logr"
	"rclone-backup.kimi450.com/pkg/config"
)

// Rclone struct holds the information regarding the rclone instance that will
// be used
type RcloneInstance struct {
	// Logger
	Log logr.Logger

	// File path to the rclone binary
	Binary string

	// File path to the rclone config file
	Config string
}

type RcloneWorker interface {
	RunVersion() error
	RunLsjson(commandOutputLogFile *os.File, dir string) error
	RunSync(logFileSyncPath, logFileSyncCombinedReportPath, sourceDir, destDir string, extraSyncArgs []string) error

	LogReportSummary(reportFilePath string) error

	SyncSourceAndDestination(logBundleDir string, extraSyncArgs []string,
		backupConfig config.BackupConfigItem) error
}

// RunVersion runs the rclone version command with opinionated default arguments
func (rclone *RcloneInstance) RunVersion() error {
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
func (rclone *RcloneInstance) RunSync(logFileSyncPath, logFileSyncCombinedReportPath string,
	sourceDir, destDir string, extraSyncArgs []string,
) error {
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
		if strings.Contains(err.Error(), "exit status 3") {
			return merry.Errorf("directory does not exist: %w", err)
		}
		return merry.Errorf("failed to run command with output [%s]: %w", string(output), err)
	}
	rclone.Log.Info("finished running command", "cmd", cmd.String())

	return nil
}

// RunLsjson runs the rclone lsjon command with opinionated default arguments
func (rclone *RcloneInstance) RunLsjson(commandOutputLogFile *os.File, dir string) error {
	cmd := exec.Command(rclone.Binary, "lsjson",
		"--config", rclone.Config,
		"-R", dir)
	rclone.Log.Info("running command", "cmd", cmd.String())

	cmd.Stderr = commandOutputLogFile
	cmd.Stdout = commandOutputLogFile
	// Has to be Run so we can redirect output to only the file
	// Too bulky otherwise
	err := cmd.Run()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 3") {
			return merry.Errorf("directory does not exist: %w", err)
		}
		return merry.Errorf("failed to run command (please refer to https://rclone.org/docs/#list-of-exit-codes for more info): %w", err)
	}
	rclone.Log.Info("finished running command", "cmd", cmd.String())

	return nil
}
