package add_test

import (
	"path/filepath"
	"testing"

	"github.com/crolly/dynQL/cmd"
	"github.com/crolly/dynQL/cmd/helpers"
	"github.com/stretchr/testify/assert"
)

func TestSchemaCmd(t *testing.T) {
	_, err := helpers.ExecuteCommand(cmd.RootCmd, "create", "test")
	wd, _ := helpers.GetWorkingDir()
	folder := filepath.Join(wd, "test")
	if err != nil {
		remove(folder)
	}

	// copy conf file
	confFilePath := filepath.Join(wd, "dql.conf.json")
	copy(filepath.Join(folder, "dql.conf.json"), confFilePath)

	// execute command to test
	_, err = helpers.ExecuteCommand(cmd.RootCmd, "add", "schema", "api")
	if err != nil {
		remove(folder, confFilePath)
	}
	assert.FileExists(t, filepath.Join(folder, "serverless.yml"))

	sFolder := filepath.Join(folder, "schemas", "api")
	assert.DirExists(t, sFolder)
	assert.DirExists(t, filepath.Join(sFolder, "schema"))
	assert.FileExists(t, filepath.Join(sFolder, "main.go"))
	assert.FileExists(t, filepath.Join(sFolder, "main_test.go"))

	// TODO: test ServerlessConfig

	remove(folder, confFilePath)
}
