package rclone

import (
	"fmt"
	"os"
	"path"

	"github.com/ansel1/merry/v2"
	"rclone-backup.kimi450.com/pkg/config"
	"rclone-backup.kimi450.com/pkg/logging"
)

// SyncSourceAndDestination syncs the source and destination directories
//
// It generates relevant ls logs of files before and after the sync for record
// keeping purposes and writes them to log files created in this function
//
// A summary report is outputed as well for ease of use by viewing script logs
func (rclone *RcloneInstance) SyncSourceAndDestination(logBundleDir string,
	extraSyncArgs []string, backupConfig config.BackupConfigItem,
) error {
	// Create all the log files required as part of this function
	fileDateTime := config.GetDateTimePrefixForFile()

	logFileSourceFiles := &logging.LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-source-files.json",
				fileDateTime, backupConfig.Name)),
	}
	logFileDestFilesBeforeSync := &logging.LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-dest-files-before-sync.json",
				fileDateTime, backupConfig.Name)),
	}
	logFileDestFilesAfterSync := &logging.LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-dest-files-after-sync.json",
				fileDateTime, backupConfig.Name)),
	}
	logFileSync := &logging.LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-sync-logs.json",
				fileDateTime, backupConfig.Name)),
	}
	logFileSyncCombinedReport := &logging.LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-sync-report.txt",
				fileDateTime, backupConfig.Name)),
	}

	logFiles := []*logging.LogFile{
		logFileSourceFiles,
		logFileDestFilesBeforeSync,
		logFileDestFilesAfterSync,
		logFileSync,
		logFileSyncCombinedReport,
	}

	// Create files on the OS for the paths
	for _, logFile := range logFiles {
		rclone.Log.Info("creating log file", "filePath", logFile.Path)
		file, err := os.Create(logFile.Path)
		if err != nil {
			return merry.Errorf("failed to create log file '%s': %s",
				logFile.Path, err)
		}
		logFile.File = file
		defer logFile.File.Close()
	}

	// Get all the files in source dir
	rclone.Log.Info("getting ls data for source directory", "path", backupConfig.SourceDir)
	err := rclone.RunLsjson(logFileSourceFiles.File,
		backupConfig.SourceDir)
	if err != nil {
		rclone.Log.Info("WARN: directory does not exist before sync, skipped getting ls data for dir",
			"path", backupConfig.SourceDir, "error", err)
	}

	// Get all the files in destination dir before syncing data
	rclone.Log.Info("getting ls data for destination directory before syncing data",
		"path", backupConfig.DestDir)
	err = rclone.RunLsjson(logFileDestFilesBeforeSync.File,
		backupConfig.DestDir)
	if err != nil {
		rclone.Log.Info("WARN: directory does not exist before sync, skipped getting ls data for dir",
			"path", backupConfig.DestDir, "error", err)
	}

	// Run the sync command
	rclone.Log.Info("syncing source and destination",
		"source", backupConfig.SourceDir,
		"destination", backupConfig.DestDir,
		"syncLogs", logFileSync.Path,
		"syncReport", logFileSyncCombinedReport.Path,
	)
	err = rclone.RunSync(logFileSync.Path, logFileSyncCombinedReport.Path,
		backupConfig.SourceDir, backupConfig.DestDir,
		extraSyncArgs)
	if err != nil {
		return merry.Errorf("failed to sync source and destination: %w", err)
	}

	// Generate a summary from the report file
	rclone.Log.Info("generating a summary from the report file",
		"reportFilePath", logFileSyncCombinedReport.Path,
		"documentation", "https://rclone.org/commands/rclone_sync/")
	err = rclone.LogReportSummary(logFileSyncCombinedReport.Path)
	if err != nil {
		return merry.Errorf("failed to generate output report: %w", err)
	}

	// Get all the files in destination dir after syncing data
	rclone.Log.Info("getting ls data for destination directory after syncing data",
		"path", backupConfig.DestDir)
	err = rclone.RunLsjson(logFileDestFilesAfterSync.File,
		backupConfig.DestDir)
	if err != nil {
		return merry.Errorf("failed to sync source and destination: %w", err)
	}

	return nil
}
