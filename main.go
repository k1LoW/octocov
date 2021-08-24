/*
Copyright © 2021 Ken'ichiro Oyama <k1lowxb@gmail.com>

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
package main

import (
	"os"
	"strings"

	"github.com/k1LoW/octocov/cmd"
)

const envPrefix = "OCTOCOV_"

func main() {
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		k := kv[0]
		v := kv[1]
		if !strings.HasPrefix(k, envPrefix) {
			continue
		}
		k2 := strings.TrimPrefix(k, envPrefix)
		if os.Getenv(k2) == "" {
			if err := os.Setenv(k2, v); err != nil {
				panic(err)
			}
		}
	}
	cmd.Execute()
}
