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
	"os"

	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/spf13/cobra"
)

var (
	label        string
	message      string
	labelColor   string
	messageColor string
	icon         string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "badgen",
	Short: "Generate SVG badge",
	Long:  `Generate SVG badge.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := badge.New(label, message)
		if labelColor == "" {
			if err := b.SetLabelColor(labelColor); err != nil {
				return err
			}
		}
		if messageColor == "" {
			if err := b.SetMessageColor(messageColor); err != nil {
				return err
			}
		}
		if icon != "" {
			if err := b.AddIconFile(icon); err != nil {
				return err
			}
		}
		if err := b.Render(os.Stdout); err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&label, "label", "l", "label is here", "label of badge")
	rootCmd.Flags().StringVarP(&message, "message", "m", "message is here", "message of badge")
	rootCmd.Flags().StringVarP(&labelColor, "label-color", "lc", "", "color of label background")
	rootCmd.Flags().StringVarP(&messageColor, "label-message", "mc", "", "color of message background")
	rootCmd.Flags().StringVarP(&icon, "icon", "i", "", "icon of badge")
}
