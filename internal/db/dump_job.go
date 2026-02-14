package db

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"log/slog"

	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"golang.org/x/crypto/ssh"
)

func createPgDump(ctx context.Context, cfg *configs.PgDumpConfig, outputPath string) error {
	cmd := exec.CommandContext(ctx,
		"docker", "exec", cfg.Host,
		"pg_dump",
		"--username", cfg.Username,
		"--dbname", cfg.DbName,
		"-Fc",
		"-Z", "6",
		"-w",
	)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func(outFile *os.File) {
		fileCloseErr := outFile.Close()
		if fileCloseErr != nil {
			slog.ErrorContext(ctx, fileCloseErr.Error())
		}
	}(outFile)

	var stderr bytes.Buffer
	cmd.Stdout = outFile
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func closeSSH(ctx context.Context, connection *ssh.Client) {
	err := connection.Close()
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Failed to close SSH connection: %v", err))
	}
}

func connectSFTP(ctx context.Context, cfg *configs.SftpConfig) (*sftp.Client, error) {
	sshConfig := cfg.GetSshConfig()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}

	client, clientErr := sftp.NewClient(conn)
	if clientErr != nil {
		closeSSH(ctx, conn)
		return nil, fmt.Errorf("sftp client: %w", clientErr)
	}

	return client, nil
}

func uploadFile(ctx context.Context, client *sftp.Client, remotePath, localPath, filename string) error {
	// Prepare file to upload
	srcFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer func(srcFile *os.File) {
		err = srcFile.Close()
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Failed to close file: %v", err))
		}
	}(srcFile)

	// Create remote dump
	remoteFile := filepath.Join(remotePath, filename)
	dstFile, dstErr := client.Create(remoteFile)

	if dstErr != nil {
		return fmt.Errorf("create remote: %w", dstErr)
	}
	defer func(dstFile *sftp.File) {
		err = dstFile.Close()
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Failed to close remote file: %v", err))
		}
	}(dstFile)

	if _, err = srcFile.WriteTo(dstFile); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func cleanOldBackups(ctx context.Context, client *sftp.Client, remotePath string, keepLastN int) error {
	files, err := client.ReadDir(remotePath)
	if err != nil {
		return err
	}

	var backups []os.FileInfo
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "pg_dump_") {
			backups = append(backups, f)
		}
	}

	if len(backups) <= keepLastN {
		slog.InfoContext(ctx, fmt.Sprintf("Found %d backups, nothing to delete", len(backups)))
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime().Before(backups[j].ModTime())
	})

	toDelete := backups[:len(backups)-keepLastN]
	for _, f := range toDelete {
		remoteFilePath := filepath.Join(remotePath, f.Name())
		slog.InfoContext(ctx, fmt.Sprintf("Deleting old backup: %s", f.Name()))
		if err = client.Remove(remoteFilePath); err != nil {
			slog.InfoContext(ctx, fmt.Sprintf("Failed to delete %s: %v", f.Name(), err))
		}
	}

	return nil
}

func RunAutoDumpJob(ctx context.Context) {
	v := viper.GetViper()
	v.AutomaticEnv()

	v.SetDefault("POSTGRES_INTERNAL_HOST", "postgres")
	v.SetDefault("POSTGRES_PGUSER", "postgres")
	v.SetDefault("POSTGRES_PGDB", "postgres")

	v.SetDefault("SFTP_HOST", "localhost")
	v.SetDefault("SFTP_PORT", 22)
	v.SetDefault("SFTP_USER", "backup")
	v.SetDefault("SFTP_PASSWORD", "")
	v.SetDefault("SFTP_PATH", "/backups")

	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(l)

	// Prepare postgres dump for uploading
	pgCfg := configs.LoadConfig(v, configs.PgDumpConfig{})

	timestamp := time.Now().UTC().Format("2006-01-02_150405")
	filename := fmt.Sprintf("pg_dump_%s.dump", timestamp)
	localPath := filepath.Join("/tmp", filename)

	slog.InfoContext(ctx, "Creating dump: "+localPath)
	if err := createPgDump(ctx, &pgCfg, localPath); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("dump failed: %v", err))
		return
	}
	// Remove local dump copy after upload
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}(localPath)

	// Connect to S3 with SFTP
	sftpCfg := configs.LoadConfig(viper.GetViper(), configs.SftpConfig{})

	client, err := connectSFTP(ctx, &sftpCfg)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("sftp connect: %v", err))
	}
	defer func(client *sftp.Client) {
		closeSftpClientErr := client.Close()
		if closeSftpClientErr != nil {
			slog.ErrorContext(ctx, closeSftpClientErr.Error())
		}
	}(client)

	// Upload to S3
	slog.InfoContext(ctx, fmt.Sprintf("Uploading to SFTP: %s/%s", sftpCfg.Path, filename))

	if err = uploadFile(ctx, client, sftpCfg.Path, localPath, filename); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("upload failed: %v", err))
	}

	slog.InfoContext(ctx, "Finished upload to SFTP\nCleaning old backups (keep last 5)...")

	if err = cleanOldBackups(ctx, client, sftpCfg.Path, 5); err != nil {
		slog.WarnContext(ctx, fmt.Sprintf("Warning: cleanup failed: %v", err))
		slog.InfoContext(ctx, "Job completed with warnings")
	} else {
		slog.InfoContext(ctx, "Job completed successfully")
	}
}
