package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/ansel1/merry/v2"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

type ScriptArgs struct {
	LogBundleBaseDir string
	RcloneBinary     string
	RcloneConfig     string
	Config           string

	DryRun   bool
	Checksum bool
}

type LogFile struct {
	Path string
	File *os.File
}

type BackupConfig struct {
	Name      string `json:"name"`
	SourceDir string `json:"sourceDir"`
	DestDir   string `json:"destDir"`
}

type Config struct {
	Items []BackupConfig `json:"items"`
}

func RunRcloneVersion(log logr.Logger, rcloneBinary, rcloneConfig string) error {
	cmd := exec.Command(rcloneBinary, "version",
		"--config", rcloneConfig)
	log.Info("running command", "cmd", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return merry.Errorf("failed to run command: %w", err)
	}
	log.Info("finished running command", "cmd", cmd.String(),
		"output", string(output))

	return nil
}

func parseArgs() (*ScriptArgs, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

		flag.PrintDefaults()
	}
	flag.ErrHelp = errors.New("flag: help requested")

	cwd, err := os.Getwd()
	if err != nil {
		return nil, merry.Errorf("failed to get working directory: %w", err)
	}

	// linux
	var rcloneBinaryDefault string
	if runtime.GOOS == "windows" {
		rcloneBinaryDefault = path.Join(cwd, "rclone.exe")
	} else {
		rcloneBinaryDefault = path.Join(cwd, "rclone")
	}

	logBundleBaseDir := flag.String("log-bundle-base-dir", cwd,
		"Base directory for the log bundle generated")

	rcloneBinary := flag.String("rclone-binary", rcloneBinaryDefault,
		"Location of the rclone binary")

	rcloneConfig := flag.String("rclone-config",
		path.Join(cwd, "configs", "rclone.conf"),
		`Location of the clone config file
		
When using a remote source you are expected to have set it up already using '.\rclone.exe config'.
This remote's name is to be used in the the -BackupConfigJson file as the SourceDir.
Example: 'google-drive:'

Refer to these pages to setup drive and the recommended
client-id and client-secret required for this setup:
- https://rclone.org/drive/
- https://rclone.org/drive/#making-your-own-client-id`,
	)

	config := flag.String("config",
		path.Join(cwd, "configs", "config.json"),
		"Location of the script's config")

	dryRun := flag.Bool("dry-run", false,
		"Perform a dry-run of the script")

	checksum := flag.Bool("checksum", false,
		"Verify checksum of the source and destination files")

	flag.Parse()

	args := &ScriptArgs{
		LogBundleBaseDir: *logBundleBaseDir,
		RcloneBinary:     *rcloneBinary,
		RcloneConfig:     *rcloneConfig,
		Config:           *config,
		DryRun:           *dryRun,
		Checksum:         *checksum,
	}
	return args, nil
}

func GetDateTimeForFile() string {
	return "temp"
	// return time.Now().Format("20060102-150405")
}

func ValidateArgs(args *ScriptArgs) error {

	filePaths := []string{args.Config, args.RcloneBinary, args.RcloneConfig}

	for _, filePath := range filePaths {
		if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			return merry.Errorf("file does not exist '%s': %w", filePath, err)
		}
	}

	return nil
}

func SyncSourceAndDestination(log logr.Logger, logBundleDir string,
	rcloneBinary, rcloneConfig string,
	extraSyncArgs []string, backupConfig BackupConfig) error {

	fileDateTime := GetDateTimeForFile()

	logFileSourceFiles := &LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-%s-source-files.json",
				fileDateTime, fileDateTime, backupConfig.Name)),
	}
	logFileDestFilesBeforeSync := &LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-%s-dest-files-before-sync.json",
				fileDateTime, fileDateTime, backupConfig.Name)),
	}
	logFileDestFilesAfterSync := &LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-%s-dest-files-after-sync.json",
				fileDateTime, fileDateTime, backupConfig.Name)),
	}
	logFileSync := &LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-%s-sync-logs.json",
				fileDateTime, fileDateTime, backupConfig.Name)),
	}
	logFileSyncCombinedReport := &LogFile{
		Path: path.Join(logBundleDir,
			fmt.Sprintf("%s-%s-%s-sync-report.txt",
				fileDateTime, fileDateTime, backupConfig.Name)),
	}

	logFiles := []*LogFile{
		logFileSourceFiles,
		logFileDestFilesBeforeSync,
		logFileDestFilesAfterSync,
		logFileSync,
		logFileSyncCombinedReport,
	}

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
	err := RunRcloneLsJson(log, logFileSourceFiles.File, rcloneBinary,
		rcloneConfig, backupConfig.SourceDir)
	if err != nil {
		log.Info("WARN: directory does not exist before sync, skipped getting ls data for dir",
			"path", backupConfig.SourceDir)
	}

	// Get all the files in destination dir before syncing data
	log.Info("getting ls data for destination directory before syncing data",
		"path", backupConfig.DestDir)
	err = RunRcloneLsJson(log, logFileDestFilesBeforeSync.File, rcloneBinary,
		rcloneConfig, backupConfig.DestDir)
	if err != nil {
		log.Info("WARN: directory does not exist before sync, skipped getting ls data for dir",
			"path", backupConfig.DestDir)
	}

	// Run the sync command
	log.Info("syncing source and destination",
		"source", backupConfig.SourceDir,
		"destination", backupConfig.DestDir,
		"syncLogs", logFileSync.Path,
		"syncReport", logFileSyncCombinedReport.Path,
	)
	err = RunRcloneSync(log, logFileSync.Path, logFileSyncCombinedReport.Path,
		rcloneBinary, rcloneConfig, backupConfig.SourceDir, backupConfig.DestDir,
		extraSyncArgs)
	if err != nil {
		return merry.Errorf("failed to sync source and destination: %w", err)
	}

	log.Info("generating summary of from the report file",
		"reportFilePath", logFileSyncCombinedReport.Path,
		"documentation", "https://rclone.org/commands/rclone_sync/")
	err = OutputReportSummary(log, logFileSyncCombinedReport.Path)
	if err != nil {
		return merry.Errorf("failed to generate output report: %w", err)
	}

	// Get all the files in destination dir after syncing data
	log.Info("getting ls data for destination directory after syncing data",
		"path", backupConfig.DestDir)
	err = RunRcloneLsJson(log, logFileDestFilesAfterSync.File, rcloneBinary,
		rcloneConfig, backupConfig.DestDir)
	if err != nil {
		return merry.Errorf("failed to sync source and destination: %w", err)
	}

	return nil
}

func RunRcloneLsJson(log logr.Logger, commandOutputLogFile *os.File,
	rcloneBinary, rcloneConfig, dir string) error {

	cmd := exec.Command(rcloneBinary, "lsjson",
		"--config", rcloneConfig,
		"-R", dir)
	log.Info("running command", "cmd", cmd.String())

	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return merry.Errorf("directory does not exist: %w", err)
	}

	cmd.Stdout = commandOutputLogFile
	err := cmd.Run()
	if err != nil {
		return merry.Errorf("failed to run command: %w", err)
	}
	log.Info("finished running command", "cmd", cmd.String())

	return nil
}

func RunRcloneSync(log logr.Logger,
	logFileSyncPath, logFileSyncCombinedReportPath string,
	rcloneBinary, rcloneConfig, sourceDir, destDir string,
	extraSyncArgs []string) error {

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
	// extraSyncArgs
	log.Info("running command", "cmd", cmd.String())

	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return merry.Errorf("failed to run command: %w", err)
	}
	log.Info("finished running command", "cmd", cmd.String())

	return nil
}

func main() {
	// TODO implement better logging?
	// TODO context based stuff
	zapLogger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer zapLogger.Sync()

	log := zapr.NewLoggerWithOptions(zapLogger)

	scriptArgs, err := parseArgs()
	if err != nil {
		log.Error(err, "failed to parse args")
		os.Exit(1)
	}

	log.Info("script args passed", "scriptArgs", scriptArgs)
	extraSyncArgs := []string{}
	if scriptArgs.Checksum {
		log.Info("checksum will be verified")
		extraSyncArgs = append(extraSyncArgs, "--checksum")
	}
	if scriptArgs.DryRun {
		log.Info("dry-run mode is set")
		extraSyncArgs = append(extraSyncArgs, "--dry-run")
	}

	err = ValidateArgs(scriptArgs)
	if err != nil {
		log.Error(err, "failed to validate config")
		os.Exit(1)
	}

	config, err := parseConfigFile(scriptArgs.Config)
	if err != nil {
		log.Error(err, "failed to parse config")
		os.Exit(1)
	}

	logBundleDir := path.Join(scriptArgs.LogBundleBaseDir,
		GetDateTimeForFile()+"-log-bundle")
	if _, err := os.Stat(logBundleDir); errors.Is(err, os.ErrNotExist) {
		log.Info("creating log bundle directory", "filePath", logBundleDir)
		err := os.Mkdir(logBundleDir, os.ModeDir)
		if err != nil {
			log.Error(err, "failed to create log bundle directory")
			os.Exit(1)
		}
	}

	log.Info("getting rclone version")
	err = RunRcloneVersion(log, scriptArgs.RcloneBinary, scriptArgs.RcloneConfig)
	if err != nil {
		log.Error(err, "failed to get rclone version")
		os.Exit(1)
	}

	for _, backupConfig := range config.Items {
		log.Info("processing backup item", "config", backupConfig)
		err := SyncSourceAndDestination(log, logBundleDir,
			scriptArgs.RcloneBinary, scriptArgs.RcloneConfig, extraSyncArgs,
			backupConfig)
		if err != nil {
			log.Error(err, "failed to sync source and destination",
				"backupConfig", backupConfig)
			os.Exit(1)
		}
		log.Info("finished processing backup item", "config", backupConfig)
	}
}

func OutputReportSummary(log logr.Logger, reportFilePath string) error {
	file, err := os.Open(reportFilePath)
	if err != nil {
		return merry.Errorf("failed to open report file")
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)

	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if !strings.HasPrefix(line, "=") {
			log.Info("non-equal file from report", "line", line)
		}
	}
	return nil
}

func parseConfigFile(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, merry.Errorf("failed to open config file")
	}
	defer file.Close()

	config := &Config{}
	jsonParser := json.NewDecoder(file)
	if err = jsonParser.Decode(config); err != nil {
		return nil, merry.Errorf("failed to parse config file")
	}

	return config, nil
}
