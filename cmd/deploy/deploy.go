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

package deploy

import (
	"os"

	"github.com/crolly/dynQL/cmd/helpers"

	"github.com/crolly/dynQL/cmd/models"
	"github.com/spf13/cobra"
)

var (
	// DeployCmd represents the deploy command
	DeployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploys the stack to AWS using serverless framework",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := models.ReadDQLConfig()
			if err != nil {
				return err
			}

			if test {
				os.Setenv("GRAPH_DYNAMO_MODE", "test")
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
				// create tables for resources
				c.CreateResourceTables(force)
			}

			// build binaries
			c.MakeBuild(test)
			// deploy to AWS
			slsArgs := []string{"deploy", "--stage " + stage}
			if len(profile) > 0 {
				slsArgs = append(slsArgs, "--aws-profile "+profile)
			}
			return helpers.RunCmd("sls", slsArgs...)
		},
	}

	name, buildList, stage, profile string
	noUpdate, test, force           bool
)

func init() {
	DeployCmd.Flags().BoolVarP(&test, "test", "t", false, "Run Tests before deploying")
	DeployCmd.Flags().StringVarP(&stage, "stage", "s", "dev", "Define Deployment Stage")
	DeployCmd.Flags().StringVarP(&profile, "profile", "p", "", "Define Deployment Profile")
	DeployCmd.Flags().BoolVarP(&force, "force overwrite", "f", false, "Force overwrite existing Tables (might be necessary if you changed your Table Definition - e.g. new Index)")
}
