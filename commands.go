package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	ansicodes "github.com/azer/go-ansi-codes"
	"github.com/spf13/cobra"
)

var (
	ignoreTime bool
	extra    bool
)

func InitCommands(root *cobra.Command) {
	saveCmd.Flags().BoolVar(&ignoreTime, "ignore-time", false, "Bypasses check if a config has changed")
	lsCmd.Flags().BoolVarP(&extra, "extra", "e", false, "Print extra information about configs")

	root.AddCommand(lsCmd)
	root.AddCommand(saveCmd)
	root.AddCommand(rmCmd)
	root.AddCommand(initCmd)
	root.AddCommand(restoreCmd)
}

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List available configs",
	Long: `List all available saved configs with their Name, Path and Tags
Name (Tags): Path
	
Example log:
  zshrc (linux zsh shell) /home/$USER/.zshrc
	
Note:
  Tags are single word tags seperated by spaces but can be split with dashes or underlines (arch-linux)`,
	Run: func(cmd *cobra.Command, args []string) {
		conf := ReadFile()
		text := []string{}
		for i := 0; i < len(conf.Files); i++ {
			file := conf.Files[i]
			if extra {
				text = append(text, fmt.Sprintf("%v%v%v:\n  Tags: %v\n  Size: %v bytes\n  Path: %v\n  Modified: %v", ansicodes.Blue, file.DisplayName, ansicodes.Reset, strings.Join(file.Tags, " "), len(file.Content), file.Path, file.Modified))
			} else {
				text = append(text, fmt.Sprintf("%v%v%v (%v%v%v) %v", ansicodes.Blue, file.DisplayName, ansicodes.Reset, ansicodes.Grey, strings.Join(file.Tags, " "), ansicodes.Reset, file.Path))
			}
		}
		fmt.Println(strings.Join(text, "\n"))
	},
}

var saveCmd = &cobra.Command{
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

var rmCmd = &cobra.Command{
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

var initCmd = &cobra.Command{
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

var restoreCmd = &cobra.Command{
	Use:   "restore [search]",
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

		option := StringRange(ReadLine())
		for _, i := range option {
			file := options[i-1]
			err := file.write()
			CatchErr(err)
			fmt.Println(options[i-1].Path)
		}
	},
	Args: cobra.ExactArgs(1),
}

func Format(text string) string {
	lines := strings.Split(text, "\n")
	var output []string
	for _, line := range lines {
		output = append(output, strings.Trim(line, "\t "))
	}
	return strings.Join(output, "\n")
}
