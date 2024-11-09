package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

/*
type zipFile struct {
	name, content string
}

func nzf(name, content string) zipFile {
	return zipFile{name: name, content: content}
}

func zipContent(files ...zipFile) error {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for _, file := range files {
		f, err := w.Create(file.name)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(file.content))
		if err != nil {
			return err
		}
	}

	return w.Close()
}
*/

func writeFile(filename string, data string) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}

func runCommand(command string, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

func rcom(command string, args ...string) string {
	if so, se, err := runCommand(command, args...); err != nil {
		fmt.Println("rcomErr:", err.Error())
		os.Exit(1)
	} else if se != "" {
		fmt.Println(se)
		os.Exit(1)
	} else {
		return so
	}
	return ""
}

type Version struct {
	major, minor, subminor int
}

func NewVersion(major, minor, subminor int) Version {
	return Version{major: major, minor: minor, subminor: subminor}
}

func NewVersionFromVersionString(v string) (Version, error) {
	if strings.TrimSpace(v) == "" {
		return NewVersion(0, 0, 0), nil
	}

	p := strings.Split(v, ".")
	if len(p) != 3 {
		return Version{}, errors.New("invalid version number, expected [major].[minor].[subminor]")
	}

	for i := range p {
		p[i] = strings.TrimSpace(p[i])
		if p[i] == "" {
			return Version{}, errors.New([]string{"major", "minor", "subminor"}[i] + " version is empty")
		}
	}

	major := 0
	minor := 0
	subminor := 0
	var err error

	if major, err = strconv.Atoi(strings.TrimSpace(p[0])); err != nil {
		return Version{}, fmt.Errorf("major is '%s' instead of a number", p[0])
	} else if minor, err = strconv.Atoi(strings.TrimSpace(p[1])); err != nil {
		return Version{}, fmt.Errorf("minor is '%s' instead of a number", p[1])
	} else if subminor, err = strconv.Atoi(strings.TrimSpace(p[2])); err != nil {
		return Version{}, fmt.Errorf("subminor is '%s' instead of a number", p[2])
	}

	return NewVersion(major, minor, subminor), nil
}

func (v Version) Fmt() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.subminor)
}

func (v Version) Compare(ver Version) int {
	if v.major > ver.major {
		return 1
	} else if v.major < ver.major {
		return -1
	}

	if v.minor > ver.minor {
		return 1
	} else if v.minor < ver.minor {
		return -1
	}

	if v.subminor > ver.subminor {
		return 1
	} else if v.subminor < ver.subminor {
		return -1
	}

	return 0
}

func GetDistraVersion() (string, error) {
	res, err := http.Get("https://raw.githubusercontent.com/voidwyrm-2/distra/refs/heads/main/version.txt")
	if err != nil {
		return "", err
	}

	version, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	} else if string(version) == "404: Not Found" {
		return "", err
	}

	return string(version), nil
}
