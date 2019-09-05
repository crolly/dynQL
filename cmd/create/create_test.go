package create_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crolly/dynQL/cmd/models"

	"github.com/crolly/dynQL/cmd/helpers"
	"github.com/stretchr/testify/assert"

	"github.com/crolly/dynQL/cmd"
)

func TestCreateCommand(t *testing.T) {
	_, err := helpers.ExecuteCommand(cmd.RootCmd, "create", "test")
	assert.NoError(t, err)

	wd, _ := helpers.GetWorkingDir()
	folder := filepath.Join(wd, "test")
	assert.DirExists(t, folder)
	assert.FileExists(t, filepath.Join(folder, "dql.conf.json"))
	assert.FileExists(t, filepath.Join(folder, "Gopkg.toml"))

	data, _ := helpers.ReadDataFromFile(filepath.Join(folder, "dql.conf.json"))

	var actual models.DQLConfig
	json.Unmarshal(data, &actual)

	si := strings.Index(folder, "github.com")
	expected := models.DQLConfig{
		ProjectName: "test",
		ProjectPath: folder[si:],
		Region:      "eu-central-1",
	}
	assert.Equal(t, expected, actual)

	os.RemoveAll(folder)
}
