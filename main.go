package main

import (
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	goPathBin = filepath.Join(build.Default.GOPATH, "bin")
	goPathSrc = filepath.Join(build.Default.GOPATH, "src")
)

func main() {
	dry := flag.Bool("n", false, "don't build, just print the package names")
	verbose := flag.Bool("v", false, "run `go install` with the -v flag")
	flag.Parse()

	bins := flag.Args()
	if len(bins) == 0 {
		fi, err := ioutil.ReadDir(goPathBin)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range fi {
			if f.IsDir() {
				continue
			}
			bins = append(bins, filepath.Join(goPathBin, f.Name()))
		}
	}

	var (
		tmpBuildDir string
		err         error
	)
	if !*dry {
		tmpBuildDir, err = ioutil.TempDir("", "gorebuild")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(tmpBuildDir)
	}

	var wereInstallErrors bool

	for _, file := range bins {
		path, err := getMainPath(file)
		if err != nil {
			log.Printf("Skipping %s: %s", file, err)
			continue
		}
		importPath, err := filepath.Rel(goPathSrc, filepath.Dir(path))
		if err != nil {
			log.Fatal(err)
		}
		if *dry {
			fmt.Println(importPath)
			continue
		}

		goArgs := []string{"install"}
		if *verbose {
			goArgs = append(goArgs, "-v")
		}
		goArgs = append(goArgs, importPath)
		cmd := exec.Command("go", goArgs...)
		cmd.Env = append(os.Environ(), "GOBIN="+tmpBuildDir)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			wereInstallErrors = true
			continue
		}
		fi, err := ioutil.ReadDir(tmpBuildDir)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range fi {
			err := os.Rename(
				filepath.Join(tmpBuildDir, f.Name()),
				filepath.Join(goPathBin, f.Name()))
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if wereInstallErrors {
		os.Exit(1)
	}
}
