// Copyright © 2019 Christian Rolly <mail@chromium-solutions.de>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package create

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/crolly/dynQL/cmd/helpers"
	"github.com/crolly/dynQL/cmd/models"
	"github.com/spf13/cobra"
)

var (
	// CreateCmd represents the create command
	CreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Creates the boilerplate for dynQL project.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := createProjectStructure(args[0], force)
			err = os.Chdir(p)
			if err != nil {
				return err
			}
			_, err = helpers.ExecuteCommand(cmd.Root(), "add", "schema", schema)
			return err
		},
	}

	region, schema string
	force          bool

	gopkg = `[[constraint]]
	name = "github.com/aws/aws-lambda-go"
	version = "^1.0.1"`

	gitignore = `# dynQL
	bin/
	vendor/`
)

func init() {
	CreateCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	CreateCmd.Flags().StringVarP(&region, "region", "r", "eu-central-1", "Region the Project will be deployed to (e.g. us-east-1 or eu-central-1)")
	CreateCmd.Flags().StringVarP(&schema, "schema", "s", "graphql", "Schema generated together with the create command to save one step")
	CreateCmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite of the Directory in case it exists already")
}

// createsProjectStructure creates the project structure with serverless.yml and mug.config.json
func createProjectStructure(projectName string, force bool) (string, error) {
	// create new config from project name
	config, err := newConfig(projectName)
	if err != nil {
		return "", err
	}

	projPath := helpers.GetProjectPath(config.ProjectPath)
	if force {
		os.RemoveAll(projPath)
	} else if _, err := os.Stat(projPath); !os.IsNotExist(err) {
		// projectPath exists already
		return "", errors.New("folder already exists - raising error: " + err.Error())
	}
	os.MkdirAll(projPath, 0755)

	// write Gopkg.toml
	if err := ioutil.WriteFile(filepath.Join(projPath, "Gopkg.toml"), []byte(gopkg), 0644); err != nil {
		return "", err
	}

	// write .gitignore
	if err := ioutil.WriteFile(filepath.Join(projPath, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return "", err
	}

	// persist config
	return projPath, config.Write()
}

func newConfig(projectName string) (*models.DQLConfig, error) {
	pName, path, err := getPath(projectName)
	if err != nil {
		return nil, err
	}

	config := &models.DQLConfig{
		ProjectName: pName,
		ProjectPath: path,
		Region:      region,
	}

	return config, nil
}

func getPath(projectName string) (string, string, error) {
	path := ""

	// environments GOPATH
	goPath := os.Getenv("GOPATH")
	if len(goPath) == 0 {
		return "", "", errors.New("$GOPATH is not set")
	}
	srcPath := filepath.Join(goPath, "src")

	if strings.Contains(projectName, "/") {
		// project is created with full path to GOPATH src e.g. github.com/crolly/dynQL-example
		path = projectName

		i := strings.LastIndex(projectName, "/")
		projectName = projectName[i+1 : len(projectName)]
	} else {
		// project is created with project name only
		wd, err := helpers.GetWorkingDir()
		if err != nil {
			return "", "", err
		}
		if filepathHasPrefix(wd, srcPath) {
			path = filepath.Join(wd, projectName)
			path = strings.TrimPrefix(strings.Replace(path, srcPath, "", 1), "/")
		} else {
			return "", "", errors.New("You must either create the project inside of $GOPATH or provide the full path (e.g. github.com/crolly/dynQL-example")
		}
	}

	return projectName, path, nil
}

func filepathHasPrefix(path string, prefix string) bool {
	if len(path) <= len(prefix) {
		return false
	}
	if runtime.GOOS == "windows" {
		// Paths in windows are case-insensitive.
		return strings.EqualFold(path[0:len(prefix)], prefix)
	}
	return path[0:len(prefix)] == prefix

}
