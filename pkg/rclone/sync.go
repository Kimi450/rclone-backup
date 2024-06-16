package rclone

import (
	"fmt"
	"os"
	"path"

	"github.com/ansel1/merry/v2"
	"github.com/go-logr/logr"
	"rclone-backup.kimi450.com/pkg/config"
	"rclone-backup.kimi450.com/pkg/io"
	"rclone-backup.kimi450.com/pkg/logging"
)

// SyncSourceAndDestination syncs the source and destination directories
//
// It generates relevant ls logs of files before and after the sync for record
// keeping purposes and writes them to log files created in this function
//
// A summary report is outputed as well for ease of use by viewing script logs
func SyncSourceAndDestination(log logr.Logger, logBundleDir string,
	rcloneBinary, rcloneConfig string,
	extraSyncArgs []string, backupConfig config.BackupConfigItem,
) error {
	// Create all the log files required as part of this function
	fileDateTime := io.GetDateTimePrefixForFile()

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
		log.Info("creating log file", "filePath", logFile.Path)
		file, err := os.Create(logFile.Path)
		if err != nil {
			return merry.Errorf("failed to create log file '%s': %s",
				logFile.Path, err)
		}
		logFile.File = file
		defer logFile.File.Close()
	}

	// Get all the files in source dir
	log.Info("getting ls data for source directory", "path", backupConfig.SourceDir)
	err := RunLsjson(log, logFileSourceFiles.File, rcloneBinary,
		rcloneConfig, backupConfig.SourceDir)
	if err != nil {
		log.Info("WARN: directory does not exist before sync, skipped getting ls data for dir",
			"path", backupConfig.SourceDir, "error", err)
	}

	// Get all the files in destination dir before syncing data
	log.Info("getting ls data for destination directory before syncing data",
		"path", backupConfig.DestDir)
	err = RunLsjson(log, logFileDestFilesBeforeSync.File, rcloneBinary,
		rcloneConfig, backupConfig.DestDir)
	if err != nil {
		log.Info("WARN: directory does not exist before sync, skipped getting ls data for dir",
			"path", backupConfig.DestDir, "error", err)
	}

	// Run the sync command
	log.Info("syncing source and destination",
		"source", backupConfig.SourceDir,
		"destination", backupConfig.DestDir,
		"syncLogs", logFileSync.Path,
		"syncReport", logFileSyncCombinedReport.Path,
	)
	err = RunSync(log, logFileSync.Path, logFileSyncCombinedReport.Path,
		rcloneBinary, rcloneConfig, backupConfig.SourceDir, backupConfig.DestDir,
		extraSyncArgs)
	if err != nil {
		return merry.Errorf("failed to sync source and destination: %w", err)
	}

	// Generate a summary from the report file
	log.Info("generating a summary from the report file",
		"reportFilePath", logFileSyncCombinedReport.Path,
		"documentation", "https://rclone.org/commands/rclone_sync/")
	err = LogReportSummary(log, logFileSyncCombinedReport.Path)
	if err != nil {
		return merry.Errorf("failed to generate output report: %w", err)
	}

	// Get all the files in destination dir after syncing data
	log.Info("getting ls data for destination directory after syncing data",
		"path", backupConfig.DestDir)
	err = RunLsjson(log, logFileDestFilesAfterSync.File, rcloneBinary,
		rcloneConfig, backupConfig.DestDir)
	if err != nil {
		return merry.Errorf("failed to sync source and destination: %w", err)
	}

	return nil
}
