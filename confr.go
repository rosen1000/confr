package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const CONF_PATH = "./conf.json"

func main() {
	rootCmd := &cobra.Command{
		Use:   "confr",
		Short: "Configuration backup tool",
	}

	InitCommands(rootCmd)

	rootCmd.Execute()
}

func ReadConf() ConfJSON {
	var result ConfJSON
	bytes, err := os.ReadFile(CONF_PATH)
	if err != nil {
		fmt.Println("Conf file not found. Creating new...")
		conf := ConfJSON{}
		WriteConf(conf)
		return conf
	}
	if err := json.Unmarshal(bytes, &result); err != nil {
		fmt.Println("Error parsing from config file:\n", err)
		os.Exit(1)
	}
	return result
}

func WriteConf(conf ConfJSON) {
	bytes, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		fmt.Println("Error parsing config:\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(CONF_PATH, bytes, 0644); err != nil {
		fmt.Println("Error writing to config file:\n", err)
		os.Exit(1)
	}
}

func ReadLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

func (file FileJSON) ParsePermissions() Permission {
	var output Permission
	segments := strings.Split(file.Permisions, " ")
	userGroup := strings.Split(segments[0], ":")
	output.User = userGroup[0]
	output.Group = userGroup[1]
	arr := segments[1][1:] // remove first character which represents type of file
	perms := 0
	pos := 0
	for i, ch := range arr {
		if i/3 == 0 {
			pos = 100
		} else if i/3 == 1 {
			pos = 10
		} else {
			pos = 1
		}

		if ch != '-' {
			perms += (1 << (i % 3)) * pos
		}
	}
	output.Perms = perms
	return output
}

func StringRange(ranges string) []int {
	regex := regexp.MustCompile(`^!?\d+([, -]!?\d+)*$`)
	dash := regexp.MustCompile(`^(\d+)-(\d+)$`)
	if regex.Match([]byte(ranges)) {
		// compute ranges
		compiledRange := []int{}
		segments := strings.Split(ranges, "[, ]")
		var negativeSegments []int
		for _, seg := range segments {
			if seg[0] == '!' {
				neg, err := strconv.Atoi(seg[1:])
				CatchErr(err)
				negativeSegments = append(negativeSegments, neg)
			} else {
				if dash.Match([]byte(seg)) {
					finds := dash.FindStringSubmatch(seg)
					min, _ := strconv.Atoi(finds[1])
					max, _ := strconv.Atoi(finds[2])
					for i := min; i <= max; i++ {
						compiledRange = append(compiledRange, i)
					}
				} else {
					num, err := strconv.Atoi(seg)
					CatchErr(err)
					compiledRange = append(compiledRange, num)
				}
			}
		}
		for i, seg := range compiledRange {
			for _, neg := range negativeSegments {
				if seg == neg {
					compiledRange = append(compiledRange[:i], compiledRange[i+1:]...)
				}
			}
		}
		return RemoveDuplicates(compiledRange)
	} else {
		fmt.Println("Not valid range")
		return []int{}
	}
}

func RemoveDuplicates(nums []int) []int {
	unique := make(map[int]bool)
	result := []int{}

	for _, num := range nums {
		if !unique[num] {
			unique[num] = true
			result = append(result, num)
		}
	}

	return result
}

// If err is not nil, the program will exit with exit code 1
func CatchErr(err error, msg ...string) {
	if err != nil {
		if len(msg) > 0 {
			fmt.Println(msg[0], err)
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

type ConfJSON struct {
	User     string `json:"User"`
	HomePath string
	Files    []FileJSON
}

type FileJSON struct {
	Content     string
	DisplayName string
	Tags        []string
	Path        string
	Permisions  string
	Modified    time.Time
}

func (file FileJSON) write() error {
	return os.WriteFile(file.Path, []byte(file.Content), fs.FileMode(file.ParsePermissions().Perms))
}

type Permission struct {
	User  string
	Group string
	Perms int
}
