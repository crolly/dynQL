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

package add

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/crolly/dynQL/cmd/helpers"

	"github.com/crolly/dynQL/cmd/models"

	"github.com/spf13/cobra"
)

// schemaCmd represents the schema command
var (
	schemaCmd = &cobra.Command{
		Use:   "schema name [flags]",
		Short: "Add a schema to the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// get schema name
			schemaName := args[0]
			// make sure path is set
			if len(path) == 0 {
				path = schemaName
			}
			// render templates with project config and schema name
			c, err := models.ReadDQLConfig()
			if err != nil {
				return err
			}
			c.AddSchema(schemaName, strings.TrimPrefix(path, "/"))
			err = c.Write()
			if err != nil {
				return err
			}

			return renderSchemaTemplates(c, schemaName)
		},
	}
)

func init() {
	AddCmd.AddCommand(schemaCmd)
	schemaCmd.Flags().StringVarP(&path, "path", "p", "", "Path under which the Schema will be available")
}

func renderSchemaTemplates(config *models.DQLConfig, schema string) error {
	templates := []string{
		"main",
		"main_test",
		"schema",
		"schema_test",
	}

	data := map[string]interface{}{
		"Config": config,
		"Schema": schema,
	}

	// iterate over schema templates and execute
	folder := filepath.Join(config.ProjectPath, "handler", schema)
	for _, t := range templates {
		err := os.MkdirAll(folder, 0755)
		if err != nil {
			return err
		}
		f := folder
		if strings.HasPrefix(t, "schema") {
			f = filepath.Join(folder, "schema")
			err = os.MkdirAll(f, 0755)
			if err != nil {
				return err
			}
		}
		err = helpers.RenderFile(helpers.SchemaBox, t+".go", t+".tmpl", f, data)
		if err != nil {
			return err
		}
	}

	return nil
}
