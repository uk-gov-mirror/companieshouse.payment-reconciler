package filetransfer

import (
	"encoding/csv"
	"fmt"
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
func New(cfg *config.Config) (*SFTP, error) {

	if cfg.SFTPPrivateKey == "" {
		return nil, fmt.Errorf("no SFTP private key provided")
	}

	signer, err := ssh.ParsePrivateKey([]byte(cfg.SFTPPrivateKey))
	if err != nil {
		return nil, fmt.Errorf("invalid SFTP private key: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.SFTPUserName,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
	}

	sshCfg.SetDefaults()

	return &SFTP{
		Config:          cfg,
		SSHClientConfig: sshCfg,
	}, nil
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
