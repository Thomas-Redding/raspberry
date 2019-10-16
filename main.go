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
<script>
var xhttp = new XMLHttpRequest();
xhttp.open("POST", "_warm", true);
xhttp.send();
</script>
`

func main() {
  ROOT_PATH = os.Args[1]
  http.HandleFunc("/", handle)
  log.Printf("FATAL ERROR: %v", http.ListenAndServe(":8080", nil))
}

func handle(writer http.ResponseWriter, request *http.Request) {
  if request.URL.Path == "/_warm" {
    fakeFilePath := randomString(8)
    _, err := os.Open(fakeFilePath)
    SendError(writer, 200, "Warmed up disk with %s : %v", fakeFilePath, err)
    return
  }
  path := ROOT_PATH + request.URL.Path
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
  children, err := ioutil.ReadDir(path)
  if err != nil {
    SendError(writer, 500, "Internal Server Error [c]")
    return
  }
  writer.Header().Set("Content-type", "text/html")
  for _, child := range children {
    childName := child.Name()
    if strings.HasPrefix(childName, ".") { continue }

    file, err := os.Open(path + childName)
    if err != nil { continue }
    fileInfo, err = file.Stat();
    if err != nil { continue }
    isFile := !fileInfo.Mode().IsDir()
    if isFile {
      writer.Write([]byte("<a href=\"" + childName + "\">"))
    } else {
      writer.Write([]byte("<a href=\"" + childName + "/\">"))
    }
    writer.Write([]byte(childName))
    writer.Write([]byte("</a><br/>"))
  }
  writer.Write([]byte(WEB_STRING))
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
