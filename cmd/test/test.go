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

package test

import (
	"os"

	"github.com/crolly/dynQL/cmd/helpers"
	"github.com/crolly/dynQL/cmd/models"

	"github.com/spf13/cobra"
)

var (
	// TestCmd represents the test command
	TestCmd = &cobra.Command{
		Use:   "test",
		Short: "Run go tests for your project",
		RunE: func(cmd *cobra.Command, args []string) error {
			// get the config
			c, err := models.ReadDQLConfig()
			if err != nil {
				return err
			}

			// create lambda-local network if it doesn't exist already
			err = helpers.CreateLambdaNetwork()
			if err != nil {
				return err
			}
			// start dynamodb-local
			err = helpers.StartLocalDynamoDB()
			if err != nil {
				return err
			}

			os.Setenv("GRAPH_DYNAMO_MODE", "test")
			err = c.CreateResourceTables(force)
			if err != nil {
				return err
			}

			t := "go test -cover -p 1"
			if profile {
				err = helpers.RunCmd("/bin/sh", "-c", t+" -coverprofile=cover.out ./...")
				if err != nil {
					return err
				}
				return helpers.RunCmd("/bin/sh", "-c", "go tool cover -html=cover.out")
			}

			return helpers.RunCmd("/bin/sh", "-c", t+" ./...")
		},
	}

	force, profile bool
)

func init() {
	TestCmd.Flags().BoolVarP(&force, "force overwrite", "f", false, "force overwrite existing tables (might be necessary if you changed you table definition - e.g. new index)")
	TestCmd.Flags().BoolVarP(&profile, "profile coverage", "p", false, "show the code coverage profile (not compatibale with list flag)")
}
