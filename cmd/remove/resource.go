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

package remove

import (
	"github.com/crolly/dynQL/cmd/models"
	"github.com/spf13/cobra"
)

// resourceCmd represents the rmfunction command
var (
	resourceCmd = &cobra.Command{
		Use:   "resource name",
		Short: "Removes resource from schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			c, err := models.ReadDQLConfig()
			if err != nil {
				return err
			}

			// delete from configuration
			c.RemoveResource(name, deleteTable)

			// delete files
			err = c.RemoveResourceFiles(schema, name)
			if err != nil {
				return err
			}
			return c.Write()
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			printRemoveMsg()
		},
	}

	schema      string
	deleteTable bool
)

func init() {
	RemoveCmd.AddCommand(resourceCmd)
	resourceCmd.Flags().StringVarP(&schema, "schema", "s", "", "Name of the Schema the Resource should be removed from")
	resourceCmd.Flags().BoolVarP(&deleteTable, "deleteTable", "d", false, "Delete all Tables from this Resource in the local DynamoDB")

	resourceCmd.MarkFlagRequired("schema")
}
