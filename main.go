package main

/*
 go run main.go /path/to/repo /path/to/file -hd -ui -sleep

 ui = builds an index; provide dark, column-based GUI
 hd = warm up hard drive on all requests (only useful when `-ui` is being used)
 sleep = sleep for 60 seconds before serving
 */

import (
  "fmt"
  "io/ioutil"
  "encoding/json"
  "log"
  "net/http"
  "os"
  "math/rand"
  "strings"
  "strconv"
  "time"
)

var PORT = "8080"
var ROOT_PATH string
var REPO_PATH string
var IS_UI bool
var IS_HD bool
var SHOULD_SLEEP bool

// FILE_INDEX["/foo/bar"] = ["baz", "qux"]
var FILE_INDEX map[string][]string
var INDEX_STRING = "{}"

func reconstructIndex(rootPath string) {
  newIndex := make(map[string][]string)
  q := make(map[string]bool)
  nextQ := make(map[string]bool)
  nextQ[""] = true
  for {
    if len(nextQ) == 0 { break }
    q = nextQ
    nextQ = make(map[string]bool)
    for path, ok := range q {
      if !ok { break }
      newIndex[path] = []string{}
      children, err := ioutil.ReadDir(rootPath + "/" + path)
      if err != nil { continue }
      for _, child := range children {
        childName := child.Name()
        if strings.HasPrefix(childName, ".") { continue }
        newIndex[path] = append(newIndex[path], childName)
        file, err := os.Open(rootPath + "/" + path + "/" + childName)
        if err != nil { continue }
        fileInfo, err := file.Stat();
        if err != nil { continue }
        if fileInfo.Mode().IsDir() {
          if path == "" {
            nextQ[childName] = true
          } else {
            nextQ[path + "/" + childName] = true
          }
        }
        file.Close()
      }
    }
  }
  FILE_INDEX = newIndex
  slurp, _ := json.Marshal(FILE_INDEX)
  INDEX_STRING = string(slurp)
}

func main() {
  IS_HD = false
  IS_UI = false
  SHOULD_SLEEP = false
  ROOT_PATH = ""
  for i, arg := range(os.Args) {
    if i == 0 { continue }
    if arg[0] != '-' {
    	if REPO_PATH == "" {
    		REPO_PATH = arg
    	} else if ROOT_PATH == "" {
        ROOT_PATH = arg
      } else {
      	log.Printf("Too many paths given (2 expected).\n")
        os.Exit(1)
      }
      continue
    } else {
      flag := arg[1:]
      if flag == "hd" {
        IS_HD = true
      } else if flag == "ui" {
        IS_UI = true
      } else if flag == "sleep" {
        SHOULD_SLEEP = true
      } else {
        log.Printf("Unrecognized flag.\n")
        os.Exit(1)
      }
    }
  }
  if REPO_PATH == "" {
  	log.Printf("No repo path provided.\n")
  	os.Exit(1)
  }
  if REPO_PATH[0] != '/' {
  	log.Printf("Repo path must be absolute.\n")
    os.Exit(1)
  }
  if ROOT_PATH == "" {
    log.Printf("No file path provided.\n")
    os.Exit(1)
  }
  if ROOT_PATH[0] != '/' {
    log.Printf("File path must be absolute.\n")
    os.Exit(1)
  }

  if IS_UI {
    if SHOULD_SLEEP {
      log.Printf("Sleeping for 60 seconds...")
      time.Sleep(60 * time.Second)
    }
    log.Printf("Building an index...")
    reconstructIndex(ROOT_PATH)
  }
  log.Printf("Serving at http://localhost:" + PORT + "/...")
  http.HandleFunc("/", handle)
  log.Printf("FATAL ERROR: %v", http.ListenAndServe(":" + PORT, nil))
}

func warmUpDisk() {
  go func() {
    fakeFilePath := randomString(8)
    os.Open(fakeFilePath)
  }()
}

func childrenOfDir(path string) []string {
  if IS_HD {
    warmUpDisk()
  }
  if IS_UI {
    if strings.HasSuffix(path, "/") { path = path[:len(path)-1] }
    return FILE_INDEX[path]
  } else {
    files, err := ioutil.ReadDir(path)
    if err != nil { return nil }
    rtn := []string{}
    for _, file := range files {
      rtn = append(rtn, file.Name())
    }
    return rtn
  }
}

func handle(writer http.ResponseWriter, request *http.Request) {
  requestPath := request.URL.Path
  log.Printf("handle(\"%s\")", requestPath)
  if IS_UI {
    if strings.HasPrefix(requestPath, "/@/") {
      requestPath = requestPath[2:]
    } else {
      warmUpDisk()
      writer.Header().Set("Content-type", "text/html")
      slurp, _ := ioutil.ReadFile(REPO_PATH + "/foo.html")
      indexHTML := string(slurp)
      indexHTML = strings.Replace(indexHTML, "<JSON_INDEX_DATA>", INDEX_STRING, -1)
      writer.WriteHeader(200)
      writer.Write([]byte(indexHTML))
      return
    }
  }

  path := ROOT_PATH + requestPath

  file, err := os.Open(path)
  if err != nil {
    SendError(writer, 404, "File Not Found [d]")
    return
  }
  defer file.Close()

  fileInfo, err := file.Stat();
  if err != nil {
    SendError(writer, 500, "Internal Server Error [e]")
    return
  }

  isFile := !fileInfo.Mode().IsDir()
  if isFile {
    http.ServeFile(writer, request, path)
    return
  }

  // The path is a directory.
  if !strings.HasSuffix(path, "/") {
    http.Redirect(writer, request, request.URL.Path + "/", http.StatusSeeOther)
    return
  }
  children := childrenOfDir(path)
  if children == nil {
    SendError(writer, 500, "Internal Server Error [f]")
    return
  }

  writer.Header().Set("Content-type", "text/plain")
  writer.WriteHeader(200)
  for _, childName := range children {
    if strings.HasPrefix(childName, ".") { continue }
    writer.Write([]byte(childName + "\n"))
  }
}

func SendError(writer http.ResponseWriter, errorCode int, format string, args ...interface{}) {
  errorMessage := "Error " + strconv.Itoa(errorCode) + ": " + fmt.Sprintf(format, args...)
  writer.Header().Set("Content-type", "text/html")
  writer.WriteHeader(errorCode)
  writer.Write([]byte(errorMessage))
  log.Printf(errorMessage)
}

func randomString(length int) string {
  if length <= 0 { return "" }
  rand.Seed(time.Now().UnixNano())
  chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
  var b strings.Builder
  for i := 0; i < length; i++ {
    b.WriteRune(chars[rand.Intn(len(chars))])
  }
  return b.String()
}
