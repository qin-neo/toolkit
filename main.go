/*
the os/exec package intentionally does not invoke the system shell, then could not invoke seperated CMD shell.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/zserge/webview"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func startServer() string {
	ln, err := net.Listen("tcp", "127.0.0.1:55555")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer ln.Close()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if len(path) > 0 && path[0] == '/' {
				path = path[1:]
			}
			if path == "" {
				path = "index.html"
			}
			if bs, err := Asset(path); err != nil {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.Header().Add("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
				io.Copy(w, bytes.NewBuffer(bs))
			}
		})
		log.Fatal(http.Serve(ln, nil))
	}()
	return "http://" + ln.Addr().String()
}

type Shortcut struct {
	MainCmd string
	Dir     string
	OptList []string `json:"optList,omitempty"`
}

type InfoMap map[string]*Shortcut

var gInfoMap InfoMap

func save2Json() {
	file, err := os.Create("config.json")
	if err != nil {
		log.Fatal("create file failed", err)
	}
	defer file.Close()

	jsonBytes, err := json.Marshal(gInfoMap)
	if err != nil {
		fmt.Println(err)
	}

	_, err1 := file.WriteString(string(jsonBytes))

	if err1 != nil {
		log.Fatalf("failed writing to file: %s", err)
	}
}

func load2Json() {
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Printf("failed reading data from file: %s", err)
		gInfoMap = make(InfoMap)
		gInfoMap["explorer"] = &Shortcut{MainCmd: "explorer", OptList: []string{`https://fm939.wnyc.org/wnycfm-web`}}
		//gInfoMap["pypy"] = Shortcut {MainCmd: `pypy`, Dir:`D:\scripts\script_master\`, OptList: []string{`D:\scripts\script_master\script_master.py`}}
		//gInfoMap["pypyw"] = Shortcut {MainCmd: `pypyw`, Dir:`D:\scripts\script_master\`, OptList: []string{`D:\scripts\script_master\script_master.py`}}
		//gInfoMap["player"] = Shortcut {Alias: "player", MainCmd: `C:/ProgramData/PureCodec/PurePlayer.exe`, OptList: []string{`https://fm939.wnyc.org/wnycfm-web`}}
	} else {
		err = json.Unmarshal(data, &gInfoMap)
		if err != nil {
			log.Fatalf("Unmarshal: %s", err)
		}
	}
	//fmt.Println(gInfoMap)
}

func updateMainPage(w webview.WebView) {
	jsonBytes, err := json.Marshal(gInfoMap)
	if err != nil {
		fmt.Println(err)
	}
	varStr := fmt.Sprintf("toolkit=%s;showShortcuts();", string(jsonBytes))
	w.Eval(varStr)
	//fmt.Println(varStr)
	//w.Bind("toolkit", &gInfoMap)
	//w.Eval(`alert(JSON.stringify(toolkit));`)
}

func append2Last(list []string, node string) []string {
	for iii, iter := range list {
		if iter == node {
			list = append(list[:iii], list[iii+1:]...)
		}
	}
	return append(list, node)
}

func handleRPC(w webview.WebView, data string) {
	log.Println(data)
	switch data[:4] {
	case "INIT":
		updateMainPage(w)
	case "HOME":
		go func() {
			exec.Command("explorer", `C:\Users\ehuawqi\Desktop\go_webview`).Run()
		}()
	case "DEL:": //DEL:alias
		delete(gInfoMap, data[4:])
		updateMainPage(w)
		save2Json()
	case "CLEA": //CLEAN:alias
		gInfoMap[data[6:]].OptList = []string{""}
		updateMainPage(w)
		save2Json()
	case "SET:":
		switch data[4:8] {
		case "FILE": //SET:FILE
			mainCmd := w.Dialog(webview.DialogTypeOpen, 0, "Open file", "")
			mainCmd = strings.Replace(mainCmd, "\\", "/", -1)
			mainFolder := filepath.Dir(mainCmd)
			mainFolder = strings.Replace(mainFolder, "\\", "/", -1)
			tmpJs := fmt.Sprintf(`mainCmd.value="%s";if(!mainFolder.value){mainFolder.value="%s";}`, mainCmd, mainFolder)
			w.Eval(tmpJs)
			log.Println(tmpJs)
		case "_DIR": //SET:_DIR
			mainFolder := w.Dialog(webview.DialogTypeOpen, webview.DialogFlagDirectory, "Open directory", "")
			mainFolder = strings.Replace(mainFolder, "\\", "/", -1)
			tmpJs := fmt.Sprintf(`mainFolder.value="%s";`, mainFolder)
			w.Eval(tmpJs)
			log.Println(tmpJs)
		}
	case "ADD:":
		alias := data[9:]
		switch data[4:8] {
		case "FILE": //ADD:FILE:alias
			chosenFile := w.Dialog(webview.DialogTypeOpen, 0, "Open file", "")
			chosenFile = strings.Replace(chosenFile, "\\", "/", -1)
			tmpJs := fmt.Sprintf(`inputDict["%s"].value ="%s";`, alias, chosenFile)
			w.Eval(tmpJs)
			log.Println(tmpJs)
		case "_DIR": //ADD:_DIR:alias
			chosenFolder := w.Dialog(webview.DialogTypeOpen, webview.DialogFlagDirectory, "Open directory", "")
			chosenFolder = strings.Replace(chosenFolder, "\\", "/", -1)
			tmpJs := fmt.Sprintf(`inputDict["%s"].value ="%s";`, alias, chosenFolder)
			w.Eval(tmpJs)
			log.Println(tmpJs)
		}
	default:
		log.Println(data)
		var rawMap map[string]interface{}
		json.Unmarshal([]byte(data), &rawMap)
		alias, _ := rawMap["alias"].(string)
		if params, ok := rawMap["params"].(string); ok {
			mainCmd := gInfoMap[alias].MainCmd
			log.Println(mainCmd)
			log.Println(params)
			go func() {
				//var cmd *exec.Cmd
				cmd := exec.Command(mainCmd, params)
				cmd.Dir = gInfoMap[alias].Dir
				stdoutStderr, err := cmd.CombinedOutput()
				if err != nil {
					log.Println(err)
				}
				fmt.Printf("%s\n", stdoutStderr)
			}()
			gInfoMap[alias].OptList = append2Last(gInfoMap[alias].OptList, params)
		} else if mainCmd, ok := rawMap["mainCmd"].(string); ok {
			//add new shortcut
			mainFolder, _ := rawMap["mainFolder"].(string)
			gInfoMap[alias] = &Shortcut{MainCmd: mainCmd, Dir: mainFolder, OptList: []string{""}}
		}
		save2Json()
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("---------start----------")
	load2Json()
	winHeight := 26 + len(gInfoMap)*25
	url := startServer()

	w := webview.New(webview.Settings{
		Width:                  700,
		Height:                 winHeight,
		Title:                  "Toolkit",
		URL:                    url,
		Resizable:              true,
		ExternalInvokeCallback: handleRPC,
	})

	w.Run()
	defer w.Exit()
}
