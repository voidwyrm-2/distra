package main

import (
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"path"
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
			fmt.Printf("A new version of Distra is available!(%s -> %s)\nrun `go install github.com/voidwyrm-2/distra` to install it\n", verLocal.Fmt(), verRemote.Fmt())
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

	outputDefault := func() string {
		p := strings.Split(rcom("pwd"), "/")
		return strings.TrimSpace(p[len(p)-1])
	}()

	parser := argparse.NewParser("distra", "A distribution builder for Go")

	showVersion := parser.Flag("v", "version", &argparse.Options{Required: false, Help: "Shows the current Distra version"})
	listos := parser.Flag("", "listos", &argparse.Options{Required: false, Help: "Lists the available operating systems to build for"})
	listarch := parser.List("", "listarch", &argparse.Options{Required: false, Help: "Lists the available architectures for the given operating systems"})
	output := parser.String("o", "output", &argparse.Options{Required: false, Help: "The output name to append the OS and arch onto in the format [name]_[os]-[arch]", Default: outputDefault})
	buildDir := parser.String("b", "build", &argparse.Options{Required: false, Help: "The Go folder to build instead of the current", Default: "."})
	buildAll := parser.Flag("", "build-all", &argparse.Options{Required: false, Help: "Builds all available operating systems and architectures"})
	zipFiles := parser.Flag("z", "zip", &argparse.Options{Required: false, Help: "Creates zip files named with the format [name]_[os]-[arch] with an executable inside"})
	emitSHF := parser.Flag("e", "emit-sh", &argparse.Options{Required: false, Help: "Stops the temporary compilation shellscript file from being run and deleted"})

	*buildDir = path.Clean(strings.TrimSpace(*buildDir))

	osFlags := map[string]*[]string{}

	for _, o := range osList() {
		osFlags[o] = parser.List("", o, &argparse.Options{Required: false, Help: "Compiles the given architectures for " + capitalize(o)})
	}

	if err := parser.Parse(os.Args); err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	if *showVersion {
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

	for os, archs := range osFlags {
		if slices.Contains(*archs, "all") {
			osa := osArch[os]
			archs = &osa
		}

		for _, arch := range *archs {
			name := strings.TrimSpace(*output + "_" + os + "-" + arch + "_v" + version)

			if os == "windows" {
				*output += ".exe"
			}

			if *zipFiles {
				shFile += fmt.Sprintf(`
GOOS=%s GOARCH=%s go build -o '%s/build/%s' '%s'
if [ "$?" = "0" ]; then
recall="$(pwd)"
cd '%s/build'
zip -r '%s.zip' '%s'
rm '%s'
cd "$recall"
fi`, os, arch, *buildDir, *output, *buildDir, *buildDir, name, *output, *output)
			} else {
				shFile += fmt.Sprintf("\nGOOS=%s GOARCH=%s go build -o '%s/build/%s' '%s'", os, arch, *buildDir, name, *buildDir)
			}
		}
	}

	shfname := fmt.Sprintf("__%d%d%d%d__.sh", rand.Intn(9), rand.Intn(9), rand.Intn(9), rand.Intn(9))

	if err := writeFile(shfname, shFile); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if !*emitSHF {
		rcom("sh", shfname)
		rcom("rm", shfname)
	}
}
