package main

import (
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/akamensky/argparse"
)

var osArch = map[string][]string{}

func generateOSArch(oa string) (map[string][]string, error) {
	oaMap := map[string][]string{}
	valid, err := regexp.Compile(`[a-z0-9]*/[a-z0-9]*`)
	if err != nil {
		return map[string][]string{}, err
	}
	for ln, l := range strings.Split(oa, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		} else if valid.Find([]byte(l)) == nil {
			return map[string][]string{}, fmt.Errorf("error on line %d: '%s' is an invalid [os]/[arch] pairing", ln+1, l)
		}
		s := strings.Split(l, "/")
		os, arch := s[0], s[1]
		if _, ok := oaMap[os]; !ok {
			oaMap[os] = []string{}
		}
		oaMap[os] = append(oaMap[os], arch)
	}
	return oaMap, nil
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
	if vr, err := GetDistraVersion(); err != nil && err.Error() != "" {
		fmt.Println(err.Error())
		os.Exit(1)
	} else if err != nil {
		if verRemote, err := NewVersionFromVersionString(vr); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		} else if verLocal, err := NewVersionFromVersionString(version); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		} else if verLocal.Compare(verRemote) == -1 {
			fmt.Printf("A new version of Distra is available!(%s -> %s)\nrun `go install github.com/voidwyrm-2/distra@latest` to install it\n", verLocal.Fmt(), verRemote.Fmt())
			return
		}
	}

	{
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

		osArch, _ = generateOSArch(strings.TrimSpace(oa))
	}

	outputDefault := func() string {
		p := strings.Split(rcom("pwd"), "/")
		return strings.TrimSpace(p[len(p)-1])
	}()

	fileNotGivenDef := fmt.Sprintf("[%d%d%d%d]", rand.Intn(10), rand.Intn(10), rand.Intn(10), rand.Intn(10))

	parser := argparse.NewParser("distra", "A distribution builder for Go")

	showVersion := parser.Flag("v", "version", &argparse.Options{Required: false, Help: "Shows the current Distra version"})
	listos := parser.Flag("", "listos", &argparse.Options{Required: false, Help: "Lists the available operating systems to build for"})
	listarch := parser.List("", "listarch", &argparse.Options{Required: false, Help: "Lists the available architectures for the given operating systems"})
	output := parser.String("o", "output", &argparse.Options{Required: false, Help: "The output name to append the OS and arch onto in the format [name]_[os]-[arch]", Default: outputDefault})
	buildDir := parser.String("b", "build", &argparse.Options{Required: false, Help: "The Go folder to build instead of the current one", Default: "."})
	buildAll := parser.Flag("", "build-all", &argparse.Options{Required: false, Help: "Builds all available operating systems and architectures"})
	zipFiles := parser.Flag("z", "zip", &argparse.Options{Required: false, Help: "Creates zip files named with the format [name]_[os]-[arch] with an executable inside"})
	emitSHF := parser.Flag("e", "emit-sh", &argparse.Options{Required: false, Help: "Stops the temporary compilation shellscript file from being run and deleted"})
	file := parser.String("f", "file", &argparse.Options{Required: false, Help: "The path to a folder containing a Distrafile", Default: fileNotGivenDef})
	emitToDistrafile := parser.Flag("", "emit-distrafile", &argparse.Options{Required: false, Help: "emits the given os/arch build flags to a Distrafile in the current directory"})

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

	*buildDir = path.Clean(strings.TrimSpace(*buildDir))
	if *file != fileNotGivenDef {
		*file = path.Clean(strings.TrimSpace(*file))
		if (*file)[len(*file)-1] != '/' {
			*file += "/"
		}
		*file += "Distrafile"
	}

	if *file != fileNotGivenDef && !*emitToDistrafile && !*buildAll {
		if content, err := readFile(*file); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		} else {
			if oam, err := generateOSArch(content); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			} else {
				for k, v := range oam {
					osFlags[k] = &v
				}
			}
		}
	}

	if len(osFlags) == 0 && !*buildAll {
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

	if *emitToDistrafile && *file == fileNotGivenDef {
		fmt.Println("generating Distrafile...")
		pairs := []string{}
		fmt.Println("formatting os/arch pairs...")
		osi := 1
		for k, v := range osFlags {
			for i, ar := range *v {
				pairs = append(pairs, k+"/"+ar)
				fmt.Printf("formatted %d out of %d architectures of os %d\n", i+1, len(*v), osi)
			}
			fmt.Printf("formatted %d of %d os\n", osi, len(osFlags))
			osi++
		}

		fmt.Println("writing Distrafile...")
		if err := writeFile("./Distrafile", strings.Join(pairs, "\n")); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("file written")
		return
	}

	shFile := `if [ -d build ] || [ -f build ]; then
rm -rf build
fi
mkdir build`

	*output = strings.TrimSpace(*output)

	appVersion, err := readFile("version.txt")
	if err == nil {
		appVersion = "_v" + appVersion
	}

	for os, archs := range osFlags {
		if slices.Contains(*archs, "all") {
			osa := osArch[os]
			archs = &osa
		}

		for _, arch := range *archs {
			name := strings.TrimSpace(*output + "_" + os + "-" + arch + appVersion)
			if os == "windows" && !strings.HasSuffix(*output, ".exe") {
				*output += ".exe"
			}

			fmt.Printf("generating build for %s/%s...\n", os, arch)
			if *zipFiles {
				shFile += fmt.Sprintf(`
echo 'building %s/%s...'
GOOS=%s GOARCH=%s go build -o '%s/build/%s' '%s'
echo 'built %s/%s'
if [ "$?" = "0" ]; then
recall="$(pwd)"
cd '%s/build'
echo 'zipping %s/%s...'
zip -r '%s.zip' '%s'
echo 'zipped %s/%s'
rm '%s'
cd "$recall"
fi`, os, arch, os, arch, *buildDir, *output, *buildDir, os, arch, *buildDir, os, arch, name, *output, os, arch, *output)
			} else {
				shFile += fmt.Sprintf("\necho 'building %s/%s...'\nGOOS=%s GOARCH=%s go build -o '%s/build/%s' '%s'\necho 'built %s/%s'", os, arch, os, arch, *buildDir, name, *buildDir, os, arch)
			}
			fmt.Printf("build for %s/%s generated\n", os, arch)
		}
	}

	shfname := fmt.Sprintf("__%d%d%d%d__.sh", rand.Intn(9), rand.Intn(9), rand.Intn(9), rand.Intn(9))
	fmt.Println("generated shellfile name")

	fmt.Println("writing shellfile...")
	if err := writeFile(shfname, shFile); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println("shellfile written")

	if !*emitSHF {
		fmt.Println("running shellfile...")
		rcom("sh", shfname)
		fmt.Println("shellfile completed, deleting...")
		rcom("rm", shfname)
		fmt.Println("shellfile deleted")
	}
}
