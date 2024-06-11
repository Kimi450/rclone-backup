package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

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

	DryRun    bool
	Checksums bool
}

type BackupConfig struct {
	Name      string `json:"name"`
	SourceDir string `json:"sourceDir"`
	DestDir   string `json:"destDir"`
}

type Config struct {
	Items []BackupConfig `json:"items"`
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
		rcloneBinaryDefault = path.Join(cwd, "configs", "rclone.conf")
	} else {
		rcloneBinaryDefault = path.Join(cwd, "configs", "rclone.conf")
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

	checksums := flag.Bool("checksums", false,
		"Verify checksums of the source and destination files")

	flag.Parse()

	args := &ScriptArgs{
		LogBundleBaseDir: *logBundleBaseDir,
		RcloneBinary:     *rcloneBinary,
		RcloneConfig:     *rcloneConfig,
		Config:           *config,
		DryRun:           *dryRun,
		Checksums:        *checksums,
	}

	return args, nil
}

func GetDateTimeForFile() string {
	return time.Now().Format("20060102-150405")
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

func SyncSourceAndDestination(log logr.Logger, logBundleDir string, backupConfig BackupConfig) error {
	fileDateTime := GetDateTimeForFile()

	suffix := "test"

	filePath := path.Join(logBundleDir,
		fmt.Sprintf("%s-%s-%s.log", fileDateTime, backupConfig.Name, suffix))
	log.Info("creating log file", "filePath", filePath)
	_, err := os.Create(filePath)
	if err != nil {
		return merry.Errorf("failed to create log file: %s", err)
	}

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

	log := zapr.NewLoggerWithOptions(zapLogger).WithName("rclone-backup")

	scriptArgs, err := parseArgs()
	if err != nil {
		log.Error(err, "failed to parse args")
		os.Exit(1)
	}

	log.Info("script args passed", "scriptArgs", scriptArgs)

	if scriptArgs.Checksums {
		log.Info("checksums will be verified")
	}
	if scriptArgs.DryRun {
		log.Info("dry-run mode is set")
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

	for _, backupConfig := range config.Items {

		err := SyncSourceAndDestination(log, logBundleDir, backupConfig)
		if err != nil {
			log.Error(err, "failed to sync source and destination",
				"backupConfig", backupConfig)
			os.Exit(1)
		}
	}
}

func OutputReportSummary(log logr.Logger, reportFilePath string) error {
	log = log.WithValues("reportFilePath", reportFilePath)
	log.Info("hi output")

	return nil
}

func parseConfigFile(filePath string) (*Config, error) {
	configFile, err := os.Open(filePath)
	if err != nil {
		return nil, merry.Errorf("failed to open config file")
	}

	config := &Config{}
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(config); err != nil {
		return nil, merry.Errorf("failed to parse config file")
	}

	return config, nil
}
