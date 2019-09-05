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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crolly/dynQL/cmd/helpers"
	"github.com/crolly/dynQL/cmd/models"
	"github.com/spf13/cobra"
)

// resourceCmd represents the resource command
var (
	resourceCmd = &cobra.Command{
		Use:   "resource name [flags]",
		Short: "Add a CRUDL resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(keySchema) == 0 {
				return errors.New("KeySchema must be defined")
			}
			// instantiate new resource model and parse given attributes
			modelName := args[0]
			capacityUnits := map[string]int64{
				"read":  readUnits,
				"write": writeUnits,
			}
			options := map[string]interface{}{
				"keySchema": keySchema,
				"billing":   billingMode,
				"capacity":  capacityUnits,
			}
			m, err := models.New(modelName, false, attributes, options)
			if err != nil {
				return err
			}
			m.GetImports()

			// add fields initialization to schema.go
			c, err := m.GetConfig()
			if err != nil {
				return err
			}

			// render templates
			renderResourceTemplates(c, m, schema)

			// update serverless.yml
			s, err := c.ReadServerlessConfig()
			if err != nil {
				return err
			}
			s.SetResourceWithModel(c, m)
			err = s.Write()
			if err != nil {
				return err
			}

			// persist config
			return c.Write()
		},
	}

	schema                             string
	attributes, keySchema, billingMode string
	readUnits, writeUnits              int64
)

func init() {
	AddCmd.AddCommand(resourceCmd)
	resourceCmd.Flags().StringVarP(&schema, "schema", "s", "", "Name of the Schema the Resource will be added to")
	resourceCmd.Flags().StringVarP(&attributes, "attributes", "a", "", "Attribute Definition of the Resource")
	resourceCmd.Flags().StringVarP(&keySchema, "keySchema", "k", "id:HASH", "Key Schema Definition for the DynamoDB Table Resource (not compatible with generateID)")
	resourceCmd.Flags().StringVarP(&billingMode, "billingMode", "b", "provisioned", "Choose between 'provisioned' for ProvisionedThroughput or 'ondemand'")
	resourceCmd.Flags().Int64VarP(&readUnits, "readUnits", "r", 1, "Set the ReadCapacityUnits if billingMode is set to ProvisionedThroughput")
	resourceCmd.Flags().Int64VarP(&writeUnits, "writeUnits", "w", 1, "Set the WriteCapacityUnits if billingMode is set to ProvisionedThroughput")

	resourceCmd.MarkFlagRequired("schema")
}

func renderResourceTemplates(config *models.DQLConfig, model *models.Model, schema string) error {
	templates := map[string][]string{
		"models": {
			"model",
			"model_test",
			"resource",
			"resource_test",
		},
		"schema": {
			"schema",
			"modelSchema",
			"modelSchema_test",
		},
		"services": {
			"dynamo",
			"service",
			"service_test",
		},
		"main": {
			"main_test",
		},
	}

	data := map[string]interface{}{
		"Config": config,
		"Model":  model,
		"Schema": schema,
	}

	projPath := helpers.GetProjectPath(config.ProjectPath)

	// iterate over schema templates and execute
	for g, ts := range templates {
		var f string
		if g == "schema" {
			f = filepath.Join(projPath, "handler", schema, "schema")
		} else if g == "main" {
			f = filepath.Join(projPath, "handler", schema)
		} else {
			f = filepath.Join(projPath, g)
		}
		err := os.MkdirAll(f, 0755)
		if err != nil {
			return err
		}
		for _, t := range ts {
			tFile := t + ".tmpl"
			fFile := t + ".go"
			switch t {
			case "resource", "resource_test":
				fFile = strings.ReplaceAll(t, "resource", model.Ident.Camelize().String()) + ".go"
			case "modelSchema", "modelSchema_test":
				fFile = strings.ReplaceAll(t, "modelSchema", model.Ident.Camelize().String()) + ".go"
			case "service", "service_test":
				fFile = strings.ReplaceAll(t, "service", model.Ident.Camelize().String()) + ".go"
			}
			err = helpers.RenderFile(helpers.ResourceBox, fFile, tFile, f, data)
			if err != nil {
				fmt.Println(err)
				return err
			}
		}
	}

	return nil
}
