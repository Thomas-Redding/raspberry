
package main

/*

# Functions to implement
func bucketRoot(isLocalHost) string {
  if isLocalHost {
    pwd, err := os.Getwd()
    if err != nil { return "" }
    return pwd + "/data/default"
  } else {
    // Production
    return "some-bucket-name.appspot.com:default"
  }
}
func start() {}

class System:
  files *Files
  ServeFile(filePath string)
  FileServe(key string)
  WebServe(request *http.Request)
  WebServeOrWasIndex(request *http.Request) string
    Attempts to serves the requested file. If the requested file is a directory
    containing an "index.html" file, it will return the path to the "index.html"
    file and NOT serve. Otherwise, it will serve (or throw a 404) and return "".
  SendError(errorCode int, format string, ...)

class Files:
  Write(key string, newValue []byte) error
  WriteFancy(key string, newValue []byte, contentType string, metaData map[string]string) error
  Read(key string) ([]byte, error)
  Exists(key string) bool
  Delete(key string) error
  ContentType(key string) string
  Size(key string) int64
  KeysWithPrefix(filePrefix string) ([]string, error)
  Children(dirPath string) ([]string, error)

# Reserved function names
1. main()
2. Anything in the global namespace starting with `__System_`.
*/

import (
  "fmt"
  "io/ioutil"
  "log"
  "mime"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
)

func main() {
  http.HandleFunc("/", __System_handle)
  http.HandleFunc("/_ah/health", __System_healthHandler)
  start()
  log.Printf("FATAL ERROR: %v", http.ListenAndServe(":8080", nil))
}

func __System_healthHandler(writer http.ResponseWriter, request *http.Request) {
  fmt.Fprint(writer, "ok")
}





/********** Files **********/

func __System_handle(writer http.ResponseWriter, request *http.Request) {
  files := &Files{_rootDir: bucketRoot(true)}
  system := &System{_writer: writer, files: files}
  handle(writer, request, system)
}

type Files struct {
  _rootDir string
}

func (files *Files) Write(key string, newValue []byte) error {
  return files.WriteFancy(key, newValue, "text/plain", make(map[string]string))
}

func (files *Files) WriteFancy(key string, newValue []byte, contentType string, metaData map[string]string) error {
  key = files.__System_CorrectFilePath(key)
  // Note: Since `contentType` and `metaData` are specific to the file system,
  // we just ignore them.
  err := ioutil.WriteFile(key, newValue, 0644)
  return err
}

func (files *Files) Read(key string) ([]byte, error) {
  key = files.__System_CorrectFilePath(key)
  return ioutil.ReadFile(key)
}

func (files *Files) Exists(key string) bool {
  key = files.__System_CorrectFilePath(key)
  info, err := os.Stat(key);
  if os.IsNotExist(err) { return false }
  return !info.IsDir()
}

func (files *Files) Delete(key string) error {
  key = files.__System_CorrectFilePath(key)
  if files.Exists(key) {
    return os.Remove(key)
  } else {
    return fmt.Errorf("The item at \"%s\" does not exist.", key)
  }
}

func (files *Files) ContentType(key string) string {
  key = files.__System_CorrectFilePath(key)
  mimeType := mime.TypeByExtension(filepath.Ext(key))
  if mimeType != "" { return mimeType }
  f, err := os.Open(key)
  defer f.Close()
  buffer := make([]byte, 512)
  _, err = f.Read(buffer)
  if err != nil { return "" }
  contentType := http.DetectContentType(buffer)
  return contentType
}

func (files *Files) Size(key string) int64 {
  key = files.__System_CorrectFilePath(key)
  fi, err := os.Stat(key);
  if err != nil { return 0 }
  return fi.Size()
}

func (files *Files) KeysWithPrefix(filePrefix string) ([]string, error) {
  filePrefix = files.__System_CorrectFilePath(filePrefix)
  // TODO: Optimize.
  arr := strings.Split(filePrefix, "/")
  arr = arr[0:len(arr)-1]
  dirToWalk := strings.Join(arr, "/")
  rtn := []string{}
  err := filepath.Walk(dirToWalk,
    func(path string, info os.FileInfo, err error) error {
    if err != nil { return err }
    if strings.HasPrefix(path, filePrefix) {
      rtn = append(rtn, path)
    }
    return nil
  })
  if err != nil { return nil, err }
  return rtn, nil
}

func (files *Files) Children(dirPath string) ([]string, error) {
  children, err := ioutil.ReadDir(dirPath)
  if ! strings.HasSuffix(dirPath, "/") {
    dirPath += "/"
  }
  rtn := []string{}
  if err != nil { return rtn, err }
  for _, child := range children {
    rtn = append(rtn, child.Name())
  }
  return rtn, nil
}

func (files *Files) __System_CorrectFilePath(key string) string {
  if strings.HasPrefix(key, "/") {
    return files._rootDir + key
  } else {
    return files._rootDir + "/" + key
  }
}





/********** System **********/

type System struct {
  _writer http.ResponseWriter
  files *Files
}

func (system *System) ServeFile(filePath string) {
  data, err := system.files.Read(filePath)
  if err != nil {
    system.SendError(500, "Internal Server Error: %v", err)
    return
  }
  system._writer.Header().Set("Content-type", system.files.ContentType(filePath))
  system._writer.Write(data)
}

func (system *System) FileServe(request *http.Request) {
  key := system.__System_PathFromRequest(request)
  if system.files.Exists(key) {
    // File
    system._writer.Header().Set("Content-type", system.files.ContentType(key))
    data, err := system.files.Read(key)
    if err == nil {
      system._writer.Write(data)
    } else {
      system.SendError(500, "Internal Server Error: %v", err)
    }
  } else if system.__System_IsNonEmptyDir(key) {
    // Dir
    if !strings.HasSuffix(key, "/") {
      http.Redirect(system._writer, request, key + "/", http.StatusSeeOther)
      return
    }
    children, err := system.files.Children(key)
    if err != nil {
      system.SendError(500, "Internal Server Error: %v", err)
    } else {
      for _, child := range children {
        if strings.HasPrefix(child, ".") { continue }
        system._writer.Write([]byte("<a href=\"" + child + "\">"))
        system._writer.Write([]byte(child))
        system._writer.Write([]byte("</a><br/>"))
      }
    }
  } else {
    system.SendError(404, "File Not Found")
    return
  }
}

func (system *System) WebServe(request *http.Request) {
  indexPath := system.WebServeOrWasIndex(request)
  if indexPath == "" { return }
  system._writer.Header().Set("Content-type", system.files.ContentType(indexPath))
  data, err := system.files.Read(indexPath)
  if err == nil {
    system._writer.Write(data)
  } else {
    system.SendError(500, "Internal Server Error: %v", err)
  }
}

func (system *System) WebServeOrWasIndex(request *http.Request) string {
  key := system.__System_PathFromRequest(request)
  if system.files.Exists(key) {
    // File
    if strings.HasSuffix(key, "/index.html") {
      http.Redirect(system._writer, request, key[0:len(key)-10], http.StatusSeeOther)
      return ""
    }
    system._writer.Header().Set("Content-type", system.files.ContentType(key))
    data, err := system.files.Read(key)
    if err == nil {
      system._writer.Write(data)
    } else {
      system.SendError(500, "Internal Server Error: %v", err)
    }
    return ""
  } else if system.__System_IsNonEmptyDir(key) {
    // Dir
    if !strings.HasSuffix(key, "/") {
      http.Redirect(system._writer, request, key + "/", http.StatusSeeOther)
      return ""
    }
    if system.files.Exists(key + "index.html") {
      return key + "index.html"
    } else {
      return ""
    }
  } else {
    system.SendError(404, "File Not Found")
    return ""
  }
}

func (system *System) SendError(errorCode int, format string, args ...interface{}) {
  errorMessage := "Error " + strconv.Itoa(errorCode) + ": " + fmt.Sprintf(format, args...)
  system._writer.WriteHeader(errorCode)
  system._writer.Write([]byte(errorMessage))
  log.Printf(errorMessage)
}

func (system *System) __System_IsNonEmptyDir(dirPath string) bool {
  dirPath = system.files.__System_CorrectFilePath(dirPath)
  children, err := ioutil.ReadDir(dirPath)
  if ! strings.HasSuffix(dirPath, "/") {
    dirPath += "/"
  }
  if err != nil { return false }
  return len(children) > 0
}

func (system *System) __System_PathFromRequest(request *http.Request) string {
  parts := strings.Split(request.URL.Path, "?")
  return parts[0]
}
