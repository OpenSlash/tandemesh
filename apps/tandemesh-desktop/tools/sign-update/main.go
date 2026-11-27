// Command sign-update writes a signed desktop update manifest for a release
// package, or generates a new signing key pair with -genkey.
//
//	TANDEMESH_UPDATE_KEY=<base64 seed> sign-update -file Tandemesh.zip -version 0.2.0 -url https://... > update.json
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"changeme/updatesig"
)

func main() {
	genkey := flag.Bool("genkey", false, "print a new public key and private key seed, then exit")
	file := flag.String("file", "", "release package to describe")
	version := flag.String("version", "", "release version")
	url := flag.String("url", "", "download URL of the package")
	notes := flag.String("notes", "", "release notes shown in the update prompt")
	flag.Parse()

	if *genkey {
		pub, priv, err := updatesig.GenerateKey()
		if err != nil {
			fail(err)
		}
		fmt.Printf("public:  %s\nprivate: %s\n", pub, priv)
		return
	}
	key := os.Getenv("TANDEMESH_UPDATE_KEY")
	if key == "" || *file == "" || *version == "" || *url == "" {
		fail(fmt.Errorf("need TANDEMESH_UPDATE_KEY, -file, -version and -url"))
	}
	sum, err := fileSHA256(*file)
	if err != nil {
		fail(err)
	}
	m := updatesig.Manifest{Version: *version, URL: *url, SHA256: sum, Notes: *notes}
	if err := updatesig.Sign(&m, key); err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		fail(err)
	}
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "sign-update:", err)
	os.Exit(1)
}
