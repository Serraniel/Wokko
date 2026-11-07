// Command notices adds the licenses of the Go modules compiled into Wokko to the third-party
// notices. The build fails if one can't be found, so nothing ships without its license.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var licenseFile = regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice)(\.|-|$)`)

func main() {
	if len(os.Args) != 2 {
		fail("usage: notices <third-party-notices.txt>")
	}
	cmd := exec.Command("go", "list", "-deps", "-f", `{{with .Module}}{{if not .Main}}{{.Path}} {{.Version}} {{.Dir}}{{end}}{{end}}`, "./cmd/wokko")
	cmd.Stderr = os.Stderr
	listing, err := cmd.Output()
	if err != nil {
		fail(err.Error())
	}
	seen := map[string]bool{}
	var modules []string
	for _, line := range strings.Split(string(listing), "\n") {
		if line != "" && !seen[line] {
			seen[line] = true
			modules = append(modules, line)
		}
	}
	sort.Strings(modules)

	var out bytes.Buffer
	out.WriteString("\nBackend\n=======\n")
	for _, m := range modules {
		parts := strings.SplitN(m, " ", 3)
		if len(parts) != 3 {
			fail("unexpected module line: " + m)
		}
		entries, err := os.ReadDir(parts[2])
		if err != nil {
			fail(err.Error())
		}
		fmt.Fprintf(&out, "\n%s %s\n%s\n", parts[0], parts[1], strings.Repeat("-", 60))
		found := false
		for _, e := range entries {
			if e.IsDir() || !licenseFile.MatchString(e.Name()) {
				continue
			}
			text, err := os.ReadFile(filepath.Join(parts[2], e.Name()))
			if err != nil {
				fail(err.Error())
			}
			out.Write(bytes.TrimSpace(text))
			out.WriteString("\n")
			found = true
		}
		if !found {
			fail("no license file found for " + parts[0])
		}
	}
	file, err := os.OpenFile(os.Args[1], os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		fail(err.Error())
	}
	defer file.Close()
	if _, err := file.Write(out.Bytes()); err != nil {
		fail(err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "notices:", msg)
	os.Exit(1)
}
