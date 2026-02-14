package configs

import (
	"time"

	"golang.org/x/crypto/ssh"
)

type SftpConfig struct {
	Host     string `mapstructure:"SFTP_HOST"`
	Port     int    `mapstructure:"SFTP_PORT"`
	User     string `mapstructure:"SFTP_USER"`
	Password string `mapstructure:"SFTP_PASSWORD"`
	Path     string `mapstructure:"SFTP_PATH"`
}

func (cfg *SftpConfig) GetSshConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
}
