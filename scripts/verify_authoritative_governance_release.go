//go:build ignore

package main

import (
	"flag"
	"fmt"
	"github.com/Clyra-AI/axym/core/governance"
	"os"
)

func main() {
	root := flag.String("root", "dist", "release output directory")
	tag := flag.String("tag", "", "expected release tag")
	commit := flag.String("commit", "", "expected peeled commit")
	trustedKey := flag.String("trusted-key-sha256", "", "caller-trusted public key digest")
	flag.Parse()
	if err := governance.VerifyAuthoritativeReleaseWithKeyDigest(*root, *tag, *commit, *trustedKey); err != nil {
		fmt.Fprintln(os.Stderr, "authoritative release verification failed:", err)
		os.Exit(1)
	}
	fmt.Println("authoritative governance release verified")
}
