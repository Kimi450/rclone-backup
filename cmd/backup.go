package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/ansel1/merry/v2"
	"rclone-backup.kimi450.com/pkg/config"
	"rclone-backup.kimi450.com/pkg/io"
	"rclone-backup.kimi450.com/pkg/logging"
	"rclone-backup.kimi450.com/pkg/rclone"
)

type ScriptArgs struct {
	LogBundleBaseDir string
	RcloneBinary     string
	RcloneConfig     string
	Config           string

	DryRun   bool
	Checksum bool
}

func (scriptArgs *ScriptArgs) parseArgs() error {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

		flag.PrintDefaults()
	}
	flag.ErrHelp = errors.New("flag: help requested")

	cwd, err := os.Getwd()
	if err != nil {
		return merry.Errorf("failed to get working directory: %w", err)
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
This remote's name is to be used in the the config file as the SourceDir.
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

	scriptArgs.LogBundleBaseDir = *logBundleBaseDir
	scriptArgs.RcloneBinary = *rcloneBinary
	scriptArgs.RcloneConfig = *rcloneConfig
	scriptArgs.Config = *config
	scriptArgs.DryRun = *dryRun
	scriptArgs.Checksum = *checksum

	return nil
}

func (scriptArgs *ScriptArgs) verifyExpectedFilesExist() error {
	filePaths := []string{
		scriptArgs.Config,
		scriptArgs.RcloneBinary,
		scriptArgs.RcloneConfig,
	}

	for _, filePath := range filePaths {
		if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			return merry.Errorf("file does not exist '%s': %w", filePath, err)
		}
	}

	return nil
}

func (scriptArgs *ScriptArgs) run() {
	logBundleDir := path.Join(scriptArgs.LogBundleBaseDir,
		io.GetDateTimePrefixForFile()+"-log-bundle")
	if _, err := os.Stat(logBundleDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(logBundleDir, os.ModePerm)
		if err != nil {
			panic(fmt.Errorf("failed to create log bundle directory: %w", err))
		}
	}

	logFilePath := path.Join(logBundleDir,
		fmt.Sprintf("%s-script-logs.txt", io.GetDateTimePrefixForFile()))
	_, log, err := logging.GetDefaultFileAndConsoleLogger(logFilePath, false)
	if err != nil {
		panic(fmt.Errorf("failed to setup logger: %w", err))
	}

	log.Info("log bundle directory", "filePath", logBundleDir)

	config, err := config.ParseConfigFile(scriptArgs.Config)
	if err != nil {
		log.Error(err, "failed to parse config")
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

	log.Info("getting rclone version")
	err = rclone.RunVersion(log, scriptArgs.RcloneBinary, scriptArgs.RcloneConfig)
	if err != nil {
		log.Error(err, "failed to get rclone version")
		os.Exit(1)
	}

	for _, backupConfig := range config.BackupConfigItem {
		log.Info("processing backup item", "config", backupConfig)
		err := rclone.SyncSourceAndDestination(log, logBundleDir,
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

func main() {
	// TODO
	// - rclone Create a struct with logger and basic configs?
	// - update docs
	// - comtext based stuff so we can cancel early?

	scriptArgs := &ScriptArgs{}
	err := scriptArgs.parseArgs()
	if err != nil {
		panic(fmt.Errorf("failed to parse args: %w", err))
	}

	err = scriptArgs.verifyExpectedFilesExist()
	if err != nil {
		panic(fmt.Errorf("failed to validate args: %w", err))
	}

	scriptArgs.run()
}
