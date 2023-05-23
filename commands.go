package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/andybalholm/brotli"
	ansicodes "github.com/azer/go-ansi-codes"
	diff "github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var (
	ignoreTime bool
	extra      bool
	yes        bool
)

func InitCommands(root *cobra.Command) {
	saveCmd.Flags().BoolVar(&ignoreTime, "ignore-time", false, "Bypasses check if a config has changed")
	lsCmd.Flags().BoolVarP(&extra, "extra", "e", false, "Print extra information about configs")
	updateCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Auto accept config updates")
	importCmd.Flags().BoolVarP(&yes, "force", "f", false, "Ignore existing saved configs")

	root.AddCommand(lsCmd)
	root.AddCommand(saveCmd)
	root.AddCommand(rmCmd)
	root.AddCommand(initCmd)
	root.AddCommand(restoreCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(exportCmd)
	root.AddCommand(importCmd)
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
		conf := ReadConf()
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
	Use:   "save [name] [path] [tags...]",
	Short: "Save file to configs",
	Run: func(cmd *cobra.Command, args []string) {
		conf := ReadConf()

		replace := false
		replaceIndex := 0
		displayName, filePath := args[0], args[1]
		tags := args[2:]
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
		file.Tags = tags

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

		WriteConf(conf)
		fmt.Println("Saved!")
	},
	Args: cobra.MinimumNArgs(2),
}

var rmCmd = &cobra.Command{
	Use:   "rm [config-name]",
	Short: "Remove saved config",
	Run: func(cmd *cobra.Command, args []string) {
		conf := ReadConf()
		for i, file := range conf.Files {
			if strings.Contains(file.DisplayName, args[0]) {
				fmt.Println("Found:", file.DisplayName)
				conf.Files = append(conf.Files[:i], conf.Files[i+1:]...)
				WriteConf(conf)
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
		WriteConf(conf)
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore [search]",
	Short: "Restore stored configs",
	Run: func(cmd *cobra.Command, args []string) {
		var options []FileJSON
		search := args[0]
		conf := ReadConf()
		for _, file := range conf.Files {
			if strings.Contains(file.Path, search) ||
				strings.Contains(file.DisplayName, search) ||
				slices.Contains(file.Tags, search) {
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

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Checks for updates in existing config files",
	Run: func(cmd *cobra.Command, args []string) {
		conf := ReadConf()
		reader := bufio.NewScanner(os.Stdin)
	file_loop:
		for i, file := range conf.Files {
			stats, err := os.Stat(file.Path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			// Newer local file
			if stats.ModTime().Compare(file.Modified) == 1 {
				if !yes {
					for {
						fmt.Printf("%v (%v) is newer in fs. Update? [Ync] ", file.DisplayName, file.Path)
						var b string
						dif := diff.New()
						for reader.Scan() {
							b = reader.Text()
							break
						}
						if b == "c" {
							bytes, err := os.ReadFile(file.Path)
							CatchErr(err)
							dumps := dif.DiffMain(string(bytes), file.Content, true)
							fmt.Println(dif.DiffPrettyText(dumps))
							continue
						}
						if b != "\n" && b != "y" {
							continue file_loop
						}
						break
					}
				} else {
					fmt.Printf("%v (%v) is newer in fs. Updating\n", file.DisplayName, file.Path)
				}

				bytes, err := os.ReadFile(file.Path)
				if err != nil {
					fmt.Println("Couldn't read file: ", err)
					continue
				}
				file.Content = string(bytes)
				file.Modified = stats.ModTime()
				conf.Files[i] = file
			}
		}
		WriteConf(conf)
		fmt.Println("Done!")
	},
}

var exportCmd = &cobra.Command{
	Use: "export",
	Run: func(cmd *cobra.Command, args []string) {
		var b bytes.Buffer
		gz := brotli.NewWriter(&b)
		var fileBytes []byte
		var err error
		if false {
			EnsureConfPath()
			fileBytes, err = os.ReadFile(CONF_PATH)
		} else {
			fileBytes, err = json.Marshal(ReadConf())
		}
		CatchErr(err)
		if _, err := gz.Write(fileBytes); err != nil {
			CatchErr(err)
		}
		CatchErr(gz.Close())
		os.WriteFile("confr.save", b.Bytes(), 0755)
	},
}

var importCmd = &cobra.Command{
	Use: "import [save-path]",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(CONF_PATH); err == nil && yes {
			fmt.Print("You already have confr initialized!\nRunning this command will rewrite current db, continue? [yN] ")
			resp := strings.ToLower(ReadLine())
			if resp[0] != 'y' {
				return
			}
		}

		path := "confr.save"
		if args[0] != "" {
			if strings.HasPrefix(args[0], "http") {
				resp, err := http.Get(args[0])
				CatchErr(err)
				respBytes, err := io.ReadAll(resp.Body)
				CatchErr(err)
				b := bytes.NewReader(respBytes)
				reader := brotli.NewReader(b)
				content, err := io.ReadAll(reader)
				if err != nil {
					fmt.Println("Provided link does not contain valid confr save file")
					os.Exit(1)
				}
				var conf ConfJSON
				err = json.Unmarshal(content, &conf)
				if err != nil {
					fmt.Println("Provided link does not contain valid confr save file")
					os.Exit(1)
				}
				fmt.Printf("Imported %v files\n", len(conf.Files))
				return
			}
			path = args[0]
		}
		file, err := os.Open(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// Create a brotli reader for the file
		brotliReader := brotli.NewReader(file)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Read the uncompressed data from the brotli reader
		content, err := io.ReadAll(brotliReader)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Convert the content to a string and save it
		var conf ConfJSON
		err = json.Unmarshal(content, &conf)
		CatchErr(err)
		WriteConf(conf)
		fmt.Printf("Imported %v files\n", len(conf.Files))
	},
	Args: cobra.MaximumNArgs(1),
}
