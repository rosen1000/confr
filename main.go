package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	conf_file, err := os.OpenFile("./conf.json", os.O_RDWR|os.O_CREATE, 0755)
	var data []byte
	var result map[string]interface{}
	conf_file.Read(data)
	fmt.Println(data)
	err1 := json.Unmarshal(data, &result)
	if err1 != nil {
		log.Fatal(err)
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