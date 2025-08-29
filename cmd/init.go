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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/pplang"
	"github.com/spf13/cobra"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "generate .octocov.yml",
	Long:  `generate .octocov.yml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		cf := config.Paths[0]
		p := filepath.Join(wd, cf)
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("%s already exist", p)
		}
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
		if err != nil {
			return err
		}
		lang, err := pplang.Detect(wd)
		if err != nil {
			log.Println(err)
		}
		if err := config.Generate(ctx, lang, f); err != nil {
			return err
		}
		cmd.PrintErrf("%s is generated\n", cf)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
