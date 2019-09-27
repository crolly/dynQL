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

package generate

import (
	"os"

	"github.com/crolly/dynQL/cmd/helpers"

	"github.com/crolly/dynQL/cmd/models"

	"github.com/spf13/cobra"
)

var (
	// GenerateTablesCmd represents the debug command
	GenerateTablesCmd = &cobra.Command{
		Use:   "generate tables",
		Short: "Generate Tables in Local DynamoDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			// get the config
			c, err := models.ReadDQLConfig()
			if err != nil {
				return err
			}

			// set debug environment
			os.Setenv("GRAPH_DYNAMO_MODE", "debug")

			// create lambda-local network if it doesn't exist already
			helpers.CreateLambdaNetwork()
			// start dynamodb-local
			helpers.StartLocalDynamoDB()
			// create tables for resources
			return c.CreateResourceTables(force)
		},
	}

	force bool
)

func init() {
	GenerateTablesCmd.Flags().BoolVarP(&force, "force overwrite", "f", false, "force overwrite existing tables (might be necessary if you changed you table definition - e.g. new index)")
}
