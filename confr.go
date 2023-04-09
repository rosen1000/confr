package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/azer/go-ansi-codes"
	"github.com/spf13/cobra"
)

const CONF_PATH = "./conf.json"

func main() {

	rootCmd := &cobra.Command{
		Use:   "confr",
		Short: "Configuration backup tool",
	}

	lsCmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List available configs",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("List command")
			conf := ReadFile()
			text := ""
			for i := 0; i < len(conf.Files); i++ {
				file := conf.Files[i]
				text += fmt.Sprintf("%v%v (%v):\n%v%v\n", ansicodes.Blue, file.Path, file.DisplayName, ansicodes.Reset, file.Content)
			}
			fmt.Println(text)
		},
	}

	saveCmd := &cobra.Command{
		Use:   "save [name] [path]",
		Short: "Save file to configs",
		Run: func(cmd *cobra.Command, args []string) {
			conf := ReadFile()
			var file FileJSON
			file.DisplayName = args[0]
			path, err := filepath.Abs(args[1])
			CatchErr(err)
			file.Path = path

			stats, err := os.Stat(file.Path)
			CatchErr(err, "Error while checking file:")
			if stats.IsDir() {
				panic("Directories are not implemented")
			}

			User, err := (user.LookupId(strconv.Itoa(int(stats.Sys().(*syscall.Stat_t).Uid))))
			CatchErr(err, "Couldn't get user owenership")
			Group, err := (user.LookupGroupId(strconv.Itoa(int(stats.Sys().(*syscall.Stat_t).Gid))))
			CatchErr(err, "Couldn't get group ownership")

			file.Permisions = fmt.Sprintf("%v:%v %v", User.Username, Group.Name, stats.Mode())
			file.Modified = stats.ModTime()

			bytes, err := os.ReadFile(file.Path)
			CatchErr(err)
			file.Content = string(bytes)

			conf.Files = append(conf.Files, file)

			WriteFile(conf)
		},
		Args: cobra.ExactArgs(2),
	}

	rmCmd := &cobra.Command{
		Use:   "rm [config-name]",
		Short: "Remove config",
		Run: func(cmd *cobra.Command, args []string) {
			conf := ReadFile()
			for i, file := range conf.Files {
				if strings.Contains(file.DisplayName, args[0]) {
					fmt.Println("Found:", file.DisplayName)
					conf.Files = append(conf.Files[:i], conf.Files[i+1:]...)
					WriteFile(conf)
					return
				}
			}
			fmt.Println("Nothing found")
		},
		Args: cobra.ExactArgs(1),
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Init config file storage",
		Run: func(cmd *cobra.Command, args []string) {
			conf := ConfJSON{}
			User, err := user.Current()
			CatchErr(err)
			conf.User = User.Username
			conf.HomePath = os.Getenv("HOME")
			WriteFile(conf)
		},
	}

	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(initCmd)

	rootCmd.Execute()
}

func ReadFile() ConfJSON {
	var result ConfJSON
	bytes, err := os.ReadFile(CONF_PATH)
	if err != nil {
		fmt.Println("Error reading from config file:\n", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(bytes, &result); err != nil {
		fmt.Println("Error parsing from config file:\n", err)
		os.Exit(1)
	}
	return result
}

func WriteFile(conf ConfJSON) {
	bytes, err := json.Marshal(conf)
	if err != nil {
		fmt.Println("Error parsing config:\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(CONF_PATH, bytes, 0644); err != nil {
		fmt.Println("Error writing to config file:\n", err)
		os.Exit(1)
	}
}

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
	User     string
	HomePath string
	Files    []FileJSON
}

type FileJSON struct {
	Content     string
	DisplayName string
	Path        string
	Permisions  string
	Modified    time.Time
}
