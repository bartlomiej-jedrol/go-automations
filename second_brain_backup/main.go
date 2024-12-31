// package main handles logis for making second_brain backup and uploading it to google drive.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	iCfg "github.com/bartlomiej-jedrol/go-toolkit/cfg"
	iLog "github.com/bartlomiej-jedrol/go-toolkit/log"
	iZip "github.com/bartlomiej-jedrol/go-toolkit/zip"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

var (
	service     = "second_brain_backup"
	sbPath      string
	uploadPath  string
	gdCredsPath string
	cfg         iCfg.Config
)

// init initializes config values from the config.yaml.
func init() {
	function := "init"

	cfgPath := flag.String("config", "/mnt/c/PRIV/config/config.yaml", "path to config file")
	flag.Parse()

	data, err := os.ReadFile(*cfgPath)
	if err != nil {
		iLog.Error("failed to read config.yaml", err, service, function, nil, nil)
		return
	}

	cfg = iCfg.Config{}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		iLog.Error("failed to unmarshal config.yaml", err, service, function, nil, nil)
		return
	}
	iLog.Info(fmt.Sprintf("successfully parsed config: %+v", cfg), nil, service, function, nil, nil)

	sbPath = cfg.Services[0].LocalPaths.SecondBrainPath
	uploadPath = cfg.Services[0].LocalPaths.UploadPath
	gdCredsPath = cfg.Services[0].LocalPaths.GoogleDriveCreds
}

func uploadToGDrive(filePath string) {
	function := "uploadToGDrive"
	iLog.Info("starting uploading backup to google drive...", nil, service, function, nil, nil)
	svc, err := drive.NewService(context.Background(), option.WithCredentialsFile(gdCredsPath))
	if err != nil {
		iLog.Error("failed to initialize Google Drive service", err, service, function, nil, nil)
		return
	}

	fullPath := filepath.Join(uploadPath, filePath)
	file, err := os.Open(fullPath)
	if err != nil {
		iLog.Error("failed to open zip file", err, service, function, nil, nil)
		return
	}
	defer file.Close()

	gdFolderID := cfg.Services[0].GoogleDriveFolders.SecondBrainBackups
	fileMetadata := &drive.File{
		Name:     filepath.Base(filePath),
		MimeType: "application/zip",
		Parents:  []string{gdFolderID},
	}
	gdFolder, err := svc.Files.Get(gdFolderID).Fields("name").Do()
	if err != nil {
		iLog.Error("failed to get folder name", err, service, function, nil, nil)
	}

	driveFile, err := svc.Files.Create(fileMetadata).Media(file).Do()
	if err != nil {
		iLog.Error("failed to upload file to Google Drive", err, service, function, nil, nil)
		return
	}
	iLog.Info(fmt.Sprintf("file: %s uploaded successfully to folder: %s", driveFile.Name, gdFolder.Name), nil, service, function, nil, nil)

	permission := drive.Permission{
		Type:         "user",
		Role:         "writer",
		EmailAddress: cfg.Email,
	}

	// By default, the file is accessible only to the service that uploads the file
	_, err = svc.Permissions.Create(driveFile.Id, &permission).Do()
	if err != nil {
		iLog.Error("failed to share file", err, service, function, nil, nil)
		return
	}
	iLog.Info("file shared successfully", nil, service, function, nil, nil)
	iLog.Info("finished uploading file to google drive", nil, service, function, nil, nil)
}

func main() {
	filePath := iZip.Folder(sbPath, uploadPath, "second_brain_backup")
	uploadToGDrive(filePath)
}
