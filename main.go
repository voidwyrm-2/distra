package main

import (
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strings"

	"github.com/akamensky/argparse"
)

var osArch = map[string][]string{}

func generateOSArch(oa string) {
	for _, l := range strings.Split(oa, "\n") {
		s := strings.Split(l, "/")
		os, arch := s[0], s[1]
		if _, ok := osArch[os]; !ok {
			osArch[os] = []string{}
		}
		osArch[os] = append(osArch[os], arch)
	}
}

func osList() []string {
	o := []string{}
	for k := range osArch {
		o = append(o, k)
	}
	return o
}

//go:embed version.txt
var version string

func capitalize(s string) string {
	if s == "ios" {
		return "iOS"
	} else if s == "js" {
		return "JS"
	} else if strings.HasSuffix(s, "bsd") {
		pre := s[:len(s)-3]
		return string(pre[0]-32) + pre[1:] + "BSD"
	}

	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

func main() {
	if vr, err := GetDistraVersion(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	} else {
		if verRemote, err := NewVersionFromVersionString(vr); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		} else if verLocal, err := NewVersionFromVersionString(version); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		} else if verLocal.Compare(verRemote) == -1 {
			fmt.Printf("A new version of Distra is available!(%s -> %s)\nrun `go get github.com/voidwyrm-2/distra@latest` to install it\n", verLocal.Fmt(), verRemote.Fmt())
			return
		}
	}

	oa, se, err := runCommand("go", "tool", "dist", "list")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	} else if se != "" {
		if strings.Contains(se, "not found") {
			fmt.Println("the Go executable is either not installed or not on your path; in order to use this tool, please install Go or add it to your path")
		} else {
			fmt.Println(se)
		}
		os.Exit(1)
	}

	generateOSArch(strings.TrimSpace(oa))

	parser := argparse.NewParser("distra", "A distribution builder for Go")

	version := parser.Flag("v", "version", &argparse.Options{Required: false, Help: "Shows the current Distra version"})
	listos := parser.Flag("", "listos", &argparse.Options{Required: false, Help: "Lists the available operating systems to build for"})
	listarch := parser.List("", "listarch", &argparse.Options{Required: false, Help: "Lists the available architectures for the given operating systems"})
	output := parser.String("o", "output", &argparse.Options{Required: false, Help: "The output name to append the OS and arch onto in the format [name]_[os]-[arch]"})
	buildAll := parser.Flag("", "build-all", &argparse.Options{Required: false, Help: "Builds all available operating systems and architectures"})

	osFlags := map[string]*[]string{}

	for _, o := range osList() {
		osFlags[o] = parser.List("", o, &argparse.Options{Required: false, Help: "Compiles the given architectures for " + capitalize(o)})
	}

	if err := parser.Parse(os.Args); err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	if *version {
		fmt.Println(version)
		return
	}

	if *listos {
		fmt.Println("available operating systems:", strings.Join(osList(), ", "))
		return
	}

	if len(*listarch) > 0 {
		amountMsg := "given"
		if slices.Contains(*listarch, "all") {
			*listarch = osList()
			amountMsg = "all"
		} else {
			for _, o := range *listarch {
				if _, ok := osArch[o]; !ok {
					fmt.Println("unknown operating system '" + o + "'")
					os.Exit(1)
				}
			}
		}

		fmt.Println("available architectures for " + amountMsg + " operating systems:")
		for _, o := range *listarch {
			fmt.Println("'"+o+"':", strings.Join(osArch[o], ", "))
		}
		return
	}

	if *buildAll {
		d := map[string]*[]string{}
		for o, al := range osArch {
			d[o] = &al
		}
		osFlags = d
	} else {
		for o, ar := range osFlags {
			for _, a := range *ar {
				if !slices.Contains(osArch[o], a) {
					fmt.Println("invalid architecture for operating system '" + o + "'")
					os.Exit(1)
				}
			}
		}
	}

	shFile := `if [ -d build ] || [ -f build ]; then
rm -rf build
fi
mkdir build`

	*output = strings.TrimSpace(*output)
	if *output != "" {
		*output += "_"
	} else {
		p := strings.Split(rcom("pwd"), "/")
		*output += strings.TrimSpace(p[len(p)-1]) + "_"
	}

	for os, archs := range osFlags {
		if slices.Contains(*archs, "all") {
			osa := osArch[os]
			archs = &osa
		}

		for _, arch := range *archs {
			name := *output + os + "-" + arch
			if os == "windows" {
				name += ".exe"
			}

			shFile += fmt.Sprintf("\nGOOS=%s GOARCH=%s go build -o build/%s .", os, arch, name)
		}
	}

	shfname := fmt.Sprintf("__%d%d%d%d__.sh", rand.Intn(9), rand.Intn(9), rand.Intn(9), rand.Intn(9))

	if err := writeFile(shfname, shFile); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	rcom("sh", shfname)
	rcom("rm", shfname)
}
