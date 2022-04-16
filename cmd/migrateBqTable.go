/*
Copyright © 2022 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/datastore/bq"
	"github.com/spf13/cobra"
)

// migrateBqTableCmd represents the migrateBqTable command
var migrateBqTableCmd = &cobra.Command{
	Use:   "migrate-bq-table",
	Short: "migrate table of BigQuery dataset for code metrics datastore",
	Long:  `migrate table of BigQuery dataset for code metrics datastore.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		if !c.Loaded() {
			cmd.PrintErrf("%s are not found\n", strings.Join(config.DefaultConfigFilePaths, " and "))
		}

		c.Build()

		if err := c.ReportConfigTargetReady(); err != nil {
			return err
		}
		datastores := map[string]datastore.Datastore{}
		for _, u := range c.Report.Datastores {
			if !strings.HasPrefix(u, "bq://") {
				continue
			}
			d, err := datastore.New(ctx, u, datastore.Root(c.Root()))
			if err != nil {
				return err
			}
			datastores[u] = d
		}

		if len(datastores) == 0 {
			return errors.New("bq:// are not exists")
		}

		var merr *multierror.Error
		for u, d := range datastores {
			b := d.(*bq.BQ)
			if err := b.CreateTable(ctx); err != nil {
				merr = multierror.Append(merr, err)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "%s has been created\n", u)
			}
		}
		return merr
	},
}

func init() {
	migrateBqTableCmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	rootCmd.AddCommand(migrateBqTableCmd)
}
