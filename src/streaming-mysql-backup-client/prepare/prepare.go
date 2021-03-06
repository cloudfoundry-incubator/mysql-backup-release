package prepare

import (
	"os/exec"
)

type BackupPreparer struct {
}

func DefaultBackupPreparer() *BackupPreparer {
	return &BackupPreparer{}
}

func (*BackupPreparer) Command(backupDir string) *exec.Cmd {
	return exec.Command("xtrabackup", "--prepare", "--target-dir", backupDir)
}
