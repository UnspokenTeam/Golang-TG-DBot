package db

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"golang.org/x/crypto/ssh"
	"log/slog"
)

type SftpConfig struct {
}

func createDump(ctx context.Context, cfg *configs.PgDumpConfig, outputPath string) error {
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
		err := outFile.Close()
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}(outFile)

	var stderr bytes.Buffer
	cmd.Stdout = outFile
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func connectToExternalS3ViaSftp() {

}

func updateExternalDump() {

}

func RunAutoDumpJob(ctx context.Context) {
	viper.AutomaticEnv()

	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(l)

	cfg := configs.LoadConfig(viper.GetViper(), configs.PgDumpConfig{})

	timestamp := time.Now().UTC().Format("2006-01-02_150405")
	filename := fmt.Sprintf("pg_dump_%s.dump", timestamp)
	localPath := filepath.Join("/tmp", filename)

	log.Printf("Creating dump: %s", localPath)
	if err := createDump(ctx, &cfg, localPath); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("dump failed: %w", err))
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}(localPath)

	sftpCfg := configs.LoadConfig(viper.GetViper(), configs.SftpConfig{})
	client, err := connectSFTP(&sftpCfg)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("sftp connect: %v", err))
	}
	defer func(client *sftp.Client) {
		err := client.Close()
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}(client)
}

func run(ctx context.Context, cfg Config) error {
	// 3. Загружаем файл
	log.Printf("Uploading to SFTP: %s/%s", cfg.SFTPPath, filename)
	if err := uploadFile(client, cfg.SFTPPath, localPath, filename); err != nil {
		return fmt.Errorf("upload failed: %w", err)
		}

	// 4. Удаляем старые бэкапы (оставляем только последние N)
	log.Printf("Cleaning old backups (keep last %d)", cfg.KeepLastN)
	if err := cleanOldBackups(client, cfg.SFTPPath, cfg.KeepLastN); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
		}

	log.Println("Backup completed successfully")
	return nil
}


func connectSFTP(cfg *configs.SftpConfig) (*sftp.Client, error) {
	sshConfig := cfg.GetSshSftpConfig()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
		}

	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("sftp client: %w", err)
		}

	return client, nil
}

func uploadFile(client *sftp.Client, remotePath, localPath, filename string) error {
	// Создаём директорию (если нужно)
	if err := client.MkdirAll(remotePath); err != nil {
		return fmt.Errorf("mkdir: %w", err)
		}

	// Открываем локальный файл
	srcFile, err := os.Open(localPath)
	if err != nil {
		return err
		}
	defer srcFile.Close()

	// Создаём удалённый файл
	remoteFile := filepath.Join(remotePath, filename)
	dstFile, err := client.Create(remoteFile)
	if err != nil {
		return fmt.Errorf("create remote: %w", err)
		}
	defer dstFile.Close()