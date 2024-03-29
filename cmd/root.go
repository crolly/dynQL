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

package cmd

import (
	"fmt"
	"os"

	"github.com/crolly/dynQL/cmd/generate"

	"github.com/crolly/dynQL/cmd/remove"

	"github.com/crolly/dynQL/cmd/test"

	"github.com/crolly/dynQL/cmd/deploy"

	"github.com/crolly/dynQL/cmd/debug"

	"github.com/crolly/dynQL/cmd/add"
	"github.com/crolly/dynQL/cmd/create"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "dynql",
	Short: "dynql",
	Long: `
dynql lets you ....`,
}

func init() {
	RootCmd.AddCommand(create.CreateCmd)
	RootCmd.AddCommand(add.AddCmd)
	RootCmd.AddCommand(debug.DebugCmd)
	RootCmd.AddCommand(deploy.DeployCmd)
	RootCmd.AddCommand(remove.RemoveCmd)
	RootCmd.AddCommand(test.TestCmd)
	RootCmd.AddCommand(generate.GenerateTablesCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
