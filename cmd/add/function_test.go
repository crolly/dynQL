package add_test

import (
	"path/filepath"
	"testing"

	"github.com/crolly/dynQL/cmd"
	"github.com/crolly/dynQL/cmd/helpers"
)

func TestFunctionCmd(t *testing.T) {
	_, err := helpers.ExecuteCommand(cmd.RootCmd, "create", "test")
	wd, _ := helpers.GetWorkingDir()
	folder := filepath.Join(wd, "test")
	if err != nil {
		remove(folder)
	}

	// copy conf file
	confFilePath := filepath.Join(wd, "dql.conf.json")
	copy(filepath.Join(folder, "dql.conf.json"), confFilePath)
	_, err = helpers.ExecuteCommand(cmd.RootCmd, "add", "schema", "api")
	if err != nil {
		remove(folder, confFilePath)
	}

	// execute command to test
	_, err = helpers.ExecuteCommand(cmd.RootCmd, "add", "function", "register", "-p", "/register", "-m", "post")
	if err != nil {
		remove(folder, confFilePath)
	}
	// TODO: test Config

	remove(folder, confFilePath)
}
