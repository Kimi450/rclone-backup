package rclone

import (
	"bufio"
	"os"
	"strings"

	"github.com/ansel1/merry/v2"
	"github.com/go-logr/logr"
)

// LogReportSummary logs a summary of the report file passed into the function
// It logs any non-equal files from the report
func LogReportSummary(log logr.Logger, reportFilePath string) error {
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
