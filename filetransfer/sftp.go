package filetransfer

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payment-reconciler/config"
	"github.com/companieshouse/payment-reconciler/models"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTP provides a concrete implementation of the FileTransfer interface, transferring files to an SFTP server
type SFTP struct {
	Config          *config.Config
	SSHClientConfig *ssh.ClientConfig
}

// New returns a new SFTP struct using the provided config
func New(cfg *config.Config) *SFTP {

	var authMethods []ssh.AuthMethod

	if cfg.SFTPPrivateKeyPath != "" {
		key, err := os.ReadFile(cfg.SFTPPrivateKeyPath)
		if err != nil {
			log.Error(fmt.Errorf("unable to read private key file: %s", err), nil)
		} else {
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				log.Error(fmt.Errorf("unable to parse private key: %s", err), nil)
			} else {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				log.Info("Using private key authentication from path: " + cfg.SFTPPrivateKeyPath)
			}
		}
	}

	if len(authMethods) == 0 && cfg.SFTPPrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.SFTPPrivateKey))
		if err != nil {
			log.Error(fmt.Errorf("unable to parse private key from env: %s", err), nil)
		} else {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
			log.Info("Using private key authentication from environment variable")
		}
	}

	if len(authMethods) == 0 && cfg.SFTPPassword != "" {
		authMethods = append(authMethods, ssh.Password(cfg.SFTPPassword))
		log.Info("Using password authentication (fallback)")
	}

	if len(authMethods) == 0 {
		log.Error(fmt.Errorf("no authentication methods available"), nil)
		return nil
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.SFTPUserName,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            authMethods,
	}

	sshCfg.SetDefaults()

	return &SFTP{
		Config:          cfg,
		SSHClientConfig: sshCfg,
	}
}

// UploadCSVFiles uploads an array of CSV's to an STFP server
func (t *SFTP) UploadCSVFiles(csvs []models.CSV) error {

	log.Info("Starting upload of CSV's. Initiating SSH connection to " + t.Config.SFTPServer)

	client, err := ssh.Dial("tcp", t.Config.SFTPServer+":"+t.Config.SFTPPort, t.SSHClientConfig)
	if err != nil {
		return fmt.Errorf("failed to establish connection: %s", err)
	}
	defer client.Close()

	sftpSession, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("error creating SFTP session: %s", err)
	}
	defer sftpSession.Close()

	log.Info("Connection established. Writing CSV's")

	for i := 0; i < len(csvs); i++ {

		file, err := sftpSession.Create(filepath.Join(t.Config.SFTPFilePath, filepath.Base(csvs[i].FileName)))
		if err != nil {
			return fmt.Errorf("failed to create CSV: %s", err)
		}

		w := csv.NewWriter(file)

		if err := w.WriteAll(csvs[i].Data.ToCSV()); err != nil {
			return fmt.Errorf("error writing CSV data: %s", err)
		}

		file.Close()
	}

	return nil
}
