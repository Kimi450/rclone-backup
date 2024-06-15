package rclone

import (
	"errors"
	"os"
	"os/exec"

	"github.com/ansel1/merry/v2"
	"github.com/go-logr/logr"
)

// RunVersion runs the rclone version command with opinionated default arguments
func RunVersion(log logr.Logger, rcloneBinary, rcloneConfig string) error {
	cmd := exec.Command(rcloneBinary, "version",
		"--config", rcloneConfig)
	log.Info("running command", "cmd", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return merry.Errorf("failed to run command: %s: %w", string(output), err)
	}
	log.Info("finished running command", "cmd", cmd.String(),
		"output", string(output))

	return nil
}

// RunSync runs the rclone sync command with opinionated default arguments
func RunSync(log logr.Logger,
	logFileSyncPath, logFileSyncCombinedReportPath string,
	rcloneBinary, rcloneConfig, sourceDir, destDir string,
	extraSyncArgs []string,
) error {
	cmdArgs := []string{
		"sync",
		sourceDir, destDir,
		"--config", rcloneConfig,
		"--use-json-log",
		"--log-level", "DEBUG",
		"--log-file", logFileSyncPath,
		"--combined", logFileSyncCombinedReportPath,
		"--check-first",
		"--metadata",
	}
	cmdArgs = append(cmdArgs, extraSyncArgs...)
	cmd := exec.Command(rcloneBinary, cmdArgs...)

	log.Info("running command", "cmd", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return merry.Errorf("failed to run command: %s: %w", string(output), err)
	}
	log.Info("finished running command", "cmd", cmd.String())

	return nil
}

// RunLsjson runs the rclone lsjon command with opinionated default arguments
func RunLsjson(log logr.Logger, commandOutputLogFile *os.File,
	rcloneBinary, rcloneConfig, dir string,
) error {
	cmd := exec.Command(rcloneBinary, "lsjson",
		"--config", rcloneConfig,
		"-R", dir)
	log.Info("running command", "cmd", cmd.String())

	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return merry.Errorf("directory does not exist: %w", err)
	}

	cmd.Stdout = commandOutputLogFile
	output, err := cmd.CombinedOutput()
	if err != nil {
		return merry.Errorf("failed to run command: %s: %w", string(output), err)
	}
	log.Info("finished running command", "cmd", cmd.String())

	return nil
}
