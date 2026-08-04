package main

import (
	"flag"
	"fmt"
	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"os"
	"os/user"
	"sort"
)

type result struct {
	Schema     string   `json:"schema"`
	Identity   string   `json:"identity"`
	Outcome    string   `json:"outcome"`
	UID        string   `json:"uid"`
	GID        string   `json:"gid"`
	Groups     []string `json:"groups"`
	Conclusion string   `json:"conclusion"`
}

func main() { os.Exit(run()) }
func run() int {
	fs := flag.NewFlagSet("authwhy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	name := fs.String("user", "", "local user (default current user)")
	format := fs.String("format", "json", "json")
	version := fs.Bool("version", false, "print version")
	noColor := fs.Bool("no-color", false, "disable color")
	_ = noColor
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(os.Stdout, cli.Version)
		return 0
	}
	if *format != "json" || len(fs.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "--format must be json")
		return 2
	}
	var account *user.User
	var err error
	if *name == "" {
		account, err = user.Current()
	} else {
		account, err = user.Lookup(*name)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	groups, _ := account.GroupIds()
	sort.Strings(groups)
	if err := cli.WriteJSON(os.Stdout, result{"missing-utils/authwhy/v1", account.Username, "partial", account.Uid, account.Gid, groups, "local identity and group evidence collected; SSH, PAM, NSS, and policy evidence is not yet collected"}); err != nil {
		return 4
	}
	return 3
}
