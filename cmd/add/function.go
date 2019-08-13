// Copyright © 2000 Christian Rolly <mail@chromium-solutions.de>
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

package add

import (
	"os"
	"path/filepath"

	"github.com/crolly/dynQL/cmd/helpers"
	"github.com/crolly/dynQL/cmd/models"
	"github.com/gobuffalo/flect"

	"github.com/spf13/cobra"
)

// functionCmd represents the function command
var (
	functionCmd = &cobra.Command{
		Use:   "function functionName",
		Short: "Add a Function to a Schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fName := args[0]

			// get config and add function to it
			c, err := models.ReadDQLConfig()
			if err != nil {
				return err
			}
			sc, err := models.ReadServerlessConfig(c.ProjectPath)
			if err != nil {
				return err
			}

			sc.AddFunction(fName, path, method)

			// generate files
			err = renderFunctionTemplates(c, fName)
			if err != nil {
				return err
			}
			return sc.Write()

		},
	}

	method string
)

func init() {
	AddCmd.AddCommand(functionCmd)

	functionCmd.Flags().StringVarP(&path, "path", "p", "", "Path the function will respond to e.g. /users")
	functionCmd.Flags().StringVarP(&method, "method", "m", "", "Method the function will respond to e.g. get")

	functionCmd.MarkFlagRequired("path")
	functionCmd.MarkFlagRequired("method")
}

func renderFunctionTemplates(config *models.DQLConfig, fName string) error {
	templates := []string{
		"main",
		"main_test",
	}

	data := map[string]interface{}{
		"Function": flect.New(fName),
	}

	// iterate over schema templates and execute
	folder := filepath.Join(config.ProjectPath, "handler", fName)
	for _, t := range templates {
		err := os.MkdirAll(folder, 0755)
		if err != nil {
			return err
		}
		err = helpers.RenderFile(helpers.FunctionBox, t+".go", t+".tmpl", folder, data)
		if err != nil {
			return err
		}
	}

	return nil
}
