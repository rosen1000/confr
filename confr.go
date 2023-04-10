package main

import (
	"bufio"
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
			conf := ReadFile()
			text := []string{}
			for i := 0; i < len(conf.Files); i++ {
				file := conf.Files[i]
				text = append(text, fmt.Sprintf("%v%v%v:\n  Size: %v bytes\n  Path: %v\n  Modified: %v", ansicodes.Blue, file.DisplayName, ansicodes.Reset, len(file.Content), file.Path, file.Modified))
			}
			fmt.Println(strings.Join(text, "\n"))
		},
	}

	var ignoreTime bool
	saveCmd := &cobra.Command{
		Use:   "save [name] [path]",
		Short: "Save file to configs",
		Run: func(cmd *cobra.Command, args []string) {
			conf := ReadFile()

			replace := false
			replaceIndex := 0
			displayName, filePath := args[0], args[1]
			stats, err := os.Stat(filePath)
			CatchErr(err, "Error while checking file:")
			for i, file := range conf.Files {
				if !ignoreTime && file.Modified.Equal(stats.ModTime()) {
					fmt.Println("File not changed. Ignoring")
					return
				}

				absPath, err := filepath.Abs(filePath)
				CatchErr(err, "File not found:")
				if file.DisplayName == displayName || file.Path == absPath {
					fmt.Printf("Found the following:\n  Name: %v\n  Size: %v bytes\n  Path: %v\n  Modified: %v\n", file.DisplayName, len(file.Content), file.Path, file.Modified)
					fmt.Print("Overwrite? (y/n) ")
					option := ReadLine()
					if option == "y" || option == "yes" {
						replace = true
						replaceIndex = i
						break
					}
					return
				}
			}

			var file FileJSON
			file.DisplayName = args[0]
			path, err := filepath.Abs(args[1])
			CatchErr(err)
			file.Path = path

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

			if replace {
				fmt.Println()
				conf.Files = append(conf.Files[:replaceIndex], conf.Files[:replaceIndex+1]...)
			} else {
				conf.Files = append(conf.Files, file)
			}

			WriteFile(conf)
			fmt.Println("Saved!")
		},
		Args: cobra.ExactArgs(2),
	}

	saveCmd.Flags().BoolVar(&ignoreTime, "ignore-time", false, "Bypasses check if a config has changed")

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
		Short: "Explicidly init config file storage",
		Long:  "Explicidly initialize config file storage and provide all information about it like location and such",
		Run: func(cmd *cobra.Command, args []string) {
			conf := ConfJSON{}
			User, err := user.Current()
			CatchErr(err)
			conf.User = User.Username
			conf.HomePath = os.Getenv("HOME")
			WriteFile(conf)
		},
	}

	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore stored configs",
		Run: func(cmd *cobra.Command, args []string) {
			var options []FileJSON
			search := args[0]
			conf := ReadFile()
			for _, file := range conf.Files {
				if strings.Contains(file.Path, search) || strings.Contains(file.DisplayName, search) {
					options = append(options, file)
				}
			}

			for i, option := range options {
				fmt.Printf("%d: %s %s\n", i+1, option.DisplayName, option.Path)
			}

			option, err := strconv.Atoi(ReadLine())
			CatchErr(err, "Not a number")

			if option < 1 || option > len(options) {
				fmt.Println("Not in range")
				return
			}

			fmt.Printf("%#v\n", options[option])
		},
		Args: cobra.ExactArgs(1),
	}

	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(restoreCmd)

	rootCmd.Execute()
}

func ReadFile() ConfJSON {
	var result ConfJSON
	bytes, err := os.ReadFile(CONF_PATH)
	if err != nil {
		fmt.Println("Conf file not found. Creating new...")
		conf := ConfJSON{}
		WriteFile(conf)
		return conf
	}
	if err := json.Unmarshal(bytes, &result); err != nil {
		fmt.Println("Error parsing from config file:\n", err)
		os.Exit(1)
	}
	return result
}

func WriteFile(conf ConfJSON) {
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
