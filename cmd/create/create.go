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
			// TODO: change to directory after creation
			_, err := createProjectStructure(args[0], force)
			if err != nil {
				return err
			}
			return nil
		},
	}

	region, schema string
	force          bool

	gopkg = `[[constraint]]
	name = "github.com/aws/aws-lambda-go"
	version = "^1.0.1"`
)

func init() {
	CreateCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	CreateCmd.Flags().StringVarP(&region, "region", "r", "eu-central-1", "Region the Project will be deployed to (e.g. us-east-1 or eu-central-1)")
	CreateCmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite of the Directory in case it exists already")
}

// createsProjectStructure creates the project structure with serverless.yml and mug.config.json
func createProjectStructure(projectName string, force bool) (string, error) {
	// create new config from project name
	config, err := newConfig(projectName)
	if err != nil {
		return "", err
	}

	if force {
		os.RemoveAll(config.ProjectPath)
	} else if _, err := os.Stat(config.ProjectPath); !os.IsNotExist(err) {
		// projectPath exists already
		return "", errors.New("folder already exists - raising error: " + err.Error())
	}
	os.MkdirAll(config.ProjectPath, 0755)

	// write Gopkg.toml
	if err := ioutil.WriteFile(filepath.Join(config.ProjectPath, "Gopkg.toml"), []byte(gopkg), 0644); err != nil {
		return "", err
	}

	// persist config
	return config.ProjectPath, config.Write()
}

func newConfig(projectName string) (*models.DQLConfig, error) {
	pName, pPath, iPath, err := getPaths(projectName)
	if err != nil {
		return nil, err
	}

	config := &models.DQLConfig{
		ProjectName: pName,
		ProjectPath: pPath,
		ImportPath:  iPath,
		Region:      region,
	}

	return config, nil
}

func getPaths(projectName string) (string, string, string, error) {
	projectPath, importPath := "", ""

	// environments GOPATH
	goPath := os.Getenv("GOPATH")
	if len(goPath) == 0 {
		return "", "", "", errors.New("$GOPATH is not set")
	}
	srcPath := filepath.Join(goPath, "src")

	if strings.Contains(projectName, "/") {
		// project is created with full path to GOPATH src e.g. github.com/crolly/dynQL-example
		projectPath = filepath.Join(srcPath, projectName)
		importPath = projectName

		i := strings.LastIndex(projectName, "/")
		projectName = projectName[i+1 : len(projectName)]
	} else {
		// project is created with project name only
		wd, err := helpers.GetWorkingDir()
		if err != nil {
			return "", "", "", err
		}
		if filepathHasPrefix(wd, srcPath) {
			projectPath = filepath.Join(wd, projectName)
			importPath = strings.TrimPrefix(strings.Replace(projectPath, srcPath, "", 1), "/")
		} else {
			return "", "", "", errors.New("You must either create the project inside of $GOPATH or provide the full path (e.g. github.com/crolly/dynQL-example")
		}
	}

	return projectName, projectPath, importPath, nil
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
