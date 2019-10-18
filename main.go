package main

import (
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "math/rand"
  "strings"
  "strconv"
  "time"
)

var ROOT_PATH string
var IS_UI bool
var IS_HD bool

// FILE_INDEX["/foo/bar"] = ["baz", "qux"]
var FILE_INDEX map[string][]string

var WEB_STRING = `
<style>
body {
  background-color: #222;
  color: white;
}

a {
  color: rgb(255, 221, 27);
}

</style>
`

/*
 go run main.go /path/to/file -hd -ui
 */
func main() {
  IS_UI = false
  IS_HD = false
  ROOT_PATH = ""
  for i, arg := range(os.Args) {
    if i == 0 { continue }
    if arg[0] != '-' {
      if ROOT_PATH == "" {
        ROOT_PATH = arg
      } else {
        os.Exit(1)
      }
      continue
    } else {
      flag := arg[1:]
      if flag == "hd" {
        IS_HD = true
      } else if flag == "ui" {
        IS_UI = true
      } else {
        log.Printf("Unrecognized flag.\n")
        os.Exit(1)
      }
    }
  }
  if ROOT_PATH == "" {
    log.Printf("No file path provided.\n")
    os.Exit(1)
  }
  if ROOT_PATH[0] != '/' {
    log.Printf("File path must be absolute.\n")
    os.Exit(1)
  }

  if IS_HD {
    log.Printf("Building an index...")
    FILE_INDEX = make(map[string][]string)
    q := make(map[string]bool)
    nextQ := make(map[string]bool)
    nextQ[ROOT_PATH] = true
    for {
      if len(nextQ) == 0 { break }
      q = nextQ
      nextQ = make(map[string]bool)
      for path, ok := range q {
        if !ok { break }
        FILE_INDEX[path] = []string{}
        children, err := ioutil.ReadDir(path)
        if err != nil { continue }
        for _, child := range children {
          childName := child.Name()
          if strings.HasPrefix(childName, ".") { continue }
          FILE_INDEX[path] = append(FILE_INDEX[path], childName)
          file, err := os.Open(path + "/" + childName)
          if err != nil { continue }
          fileInfo, err := file.Stat();
          if err != nil { continue }
          if fileInfo.Mode().IsDir() {
            nextQ[path + "/" + childName] = true
          }
          file.Close()
        }
      }
    }
  }
  log.Printf("Serving at http://localhost:8080/...")
  http.HandleFunc("/", handle)
  log.Printf("FATAL ERROR: %v", http.ListenAndServe(":8080", nil))
}

func childrenOfDir(path string) []string {
  if IS_HD {
    go func() {
      fakeFilePath := randomString(8)
      os.Open(fakeFilePath)
    }()
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
  path := ROOT_PATH + request.URL.Path
  log.Printf("handle %s", path)
  file, err := os.Open(path)
  if err != nil {
    SendError(writer, 404, "File Not Found [a]")
    return
  }
  defer file.Close()

  fileInfo, err := file.Stat();
  if err != nil {
    SendError(writer, 500, "Internal Server Error [b]")
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
  if err != nil {
    SendError(writer, 500, "Internal Server Error [c]")
    return
  }

  if IS_UI {
    writer.Header().Set("Content-type", "text/html")
    writer.WriteHeader(200)
    for _, childName := range children {
      if strings.HasPrefix(childName, ".") { continue }
      writer.Write([]byte("<a href=\"" + childName + "\">"))
      writer.Write([]byte(childName))
      writer.Write([]byte("</a><br/>"))
    }
    writer.Write([]byte(WEB_STRING))
  } else {
    writer.Header().Set("Content-type", "text/plain")
    writer.WriteHeader(200)
    for _, childName := range children {
      if strings.HasPrefix(childName, ".") { continue }
      writer.Write([]byte(childName + "\n"))
    }
  }
}

func SendError(writer http.ResponseWriter, errorCode int, format string, args ...interface{}) {
  errorMessage := "Error " + strconv.Itoa(errorCode) + ": " + fmt.Sprintf(format, args...)
  writer.Header().Set("Content-type", "text/html")
  writer.WriteHeader(errorCode)
  writer.Write([]byte(errorMessage))
  writer.Write([]byte(WEB_STRING))
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
