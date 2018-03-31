package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/kr/pretty"
)

var data sync.Map

func handler(w http.ResponseWriter, r *http.Request) {
	tmpMap := make(map[string]string)
	data.Range(func(k, v interface{}) bool {
		tmpMap[k.(string)] = v.(string)
		return true
	})
	data, err := json.Marshal(tmpMap)
	if err != nil {
		pretty.Println(err)
		w.Write([]byte(err.Error()))
	} else {
		w.Write(data)
	}
}

func main() {
	data.Store("https://ngerakines.me/", "success")
	go func() {
		http.HandleFunc("/", handler)
		if err := http.ListenAndServe(":8080", nil); err != nil {
			panic(err)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	var text string
	for text != "q" { // break the loop if text == "q"
		fmt.Print("command (set and unset):")
		scanner.Scan()
		text = scanner.Text()
		if text != "q" {
			parts := strings.Split(text, " ")
			if parts[0] == "set" {
				data.Store(parts[1], parts[2])
			} else if parts[0] == "unset" {
				data.Delete(parts[1])
			}
		}
	}
}
