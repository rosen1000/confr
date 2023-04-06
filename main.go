package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	conf_file, err := os.OpenFile("./conf.json", os.O_RDWR|os.O_CREATE, 0755)
	var result map[string]interface{}
	bytes, err2 := ioutil.ReadFile("./conf.json")
	data := string(bytes[:])
	if err2 != nil {
		fmt.Println("err2:", err2)
	}

	fmt.Println("data:", data)
	err1 := json.Unmarshal(bytes, &result)
	if err1 != nil {
		fmt.Println("err1:", err)
		os.Exit(1)
	}
	fmt.Println(result)
	// data := &ConfJSON{User: "hax", HomePath: "/home/hax"}
	// json, err := json.Marshal(data)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(json)
	// conf_file.Write(json)
	os.Exit(0)
	if err != nil {
		println(err)
		os.Exit(1)
	}
	defer conf_file.Close()

	switch os.Args[len(os.Args)-1] {
	case "ls", "list":
		println("hello")
	default:
		println("wrong command")
	}
}

type ConfJSON struct {
	User string
	HomePath string
}