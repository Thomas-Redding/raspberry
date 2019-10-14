
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
  "cloud.google.com/go/storage"
  "fmt"
  "golang.org/x/net/context"
  "google.golang.org/api/iterator"
  "google.golang.org/appengine"
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
  pwd, err := os.Getwd()
  if err != nil {
    writer.WriteHeader(500)
    writer.Write([]byte("Error 500: Failed to determine if running in production or localhost."))
    return
  }
  var files *Files
  if strings.HasPrefix(pwd, "/app") {
    // Running in Prod
    ctx := appengine.NewContext(request)
    client, err := storage.NewClient(ctx)
    if err != nil {
      log.Printf("failed to create client: %v", err)
      writer.WriteHeader(500)
      writer.Write([]byte("Error 500: Failed to create client context."))
      return
    }
    defer client.Close()
    bucketPwd := bucketRoot(false)
    i := strings.Index(bucketPwd, ":")
    bucketName := bucketPwd[0:i]
    bucketRootDir := bucketPwd[i+1:]
    bucketHandle := client.Bucket(bucketName)
    files = &Files{_ctx: ctx, _client: client, _bucketHandle: bucketHandle,
        _rootDir: bucketRootDir, _isLocalHost: false}
  } else if strings.HasPrefix(pwd, "/Users") {
    // Running in localhost
    files = &Files{_ctx: nil, _client: nil, _bucketHandle: nil,
        _rootDir: bucketRoot(true), _isLocalHost: true}
  } else {
    writer.WriteHeader(500)
    writer.Write([]byte("Error 500: Not running in production or localhost: \"" + pwd + "\"."))
    return
  }
  system := &System{_writer: writer, files: files}
  handle(writer, request, system)
}

type Files struct {
  _ctx context.Context
  _client *storage.Client
  _bucketHandle *storage.BucketHandle
  _rootDir string
  _isLocalHost bool
}

func (files *Files) Write(key string, newValue []byte) error {
  return files.WriteFancy(key, newValue, "text/plain", make(map[string]string))
}

func (files *Files) WriteFancy(key string, newValue []byte, contentType string, metaData map[string]string) error {
  key = files.__System_CorrectFilePath(key)
  if files._isLocalHost {
    // Note: Since `contentType` and `metaData` are specific to the file system,
    // we just ignore on localhost.
    err := ioutil.WriteFile(key, newValue, 0644)
    return err
  } else {
    wc := files._bucketHandle.Object(key).NewWriter(files._ctx)
    wc.ContentType = contentType
    wc.Metadata = metaData
    if _, err := wc.Write(newValue); err != nil { return err }
    if err := wc.Close(); err != nil { return err }
    return nil
  }
}

func (files *Files) Read(key string) ([]byte, error) {
  key = files.__System_CorrectFilePath(key)
  if files._isLocalHost {
    return ioutil.ReadFile(key)
  } else {
    rc, err := files._bucketHandle.Object(key).NewReader(files._ctx)
    if err != nil { return nil, err }
    defer rc.Close()
    slurp, err := ioutil.ReadAll(rc)
    if err != nil { return nil, err }
    return slurp, nil
  }
}

func (files *Files) Exists(key string) bool {
  key = files.__System_CorrectFilePath(key)
  if files._isLocalHost {
    info, err := os.Stat(key);
    if os.IsNotExist(err) { return false }
    return !info.IsDir()
  } else {
    _, err := files._bucketHandle.Object(key).NewReader(files._ctx)
    return (err == nil)
  }
}

func (files *Files) Delete(key string) error {
  key = files.__System_CorrectFilePath(key)
  if files._isLocalHost {
    if files.Exists(key) {
      return os.Remove(key)
    } else {
      return fmt.Errorf("The item at \"%s\" does not exist.", key)
    }
  } else {
    err := files._bucketHandle.Object(key).Delete(files._ctx)
    return err
  }
}

func (files *Files) ContentType(key string) string {
  key = files.__System_CorrectFilePath(key)
  if (files._isLocalHost) {
    mimeType := mime.TypeByExtension(filepath.Ext(key))
    if mimeType != "" { return mimeType }
    f, err := os.Open(key)
    defer f.Close()
    buffer := make([]byte, 512)
    _, err = f.Read(buffer)
    if err != nil { return "" }
    contentType := http.DetectContentType(buffer)
    return contentType
  } else {
    attrs, err := files._bucketHandle.Object(key).Attrs(files._ctx)
    if err != nil { return "" }
    return attrs.ContentType
  }
}

func (files *Files) Size(key string) int64 {
  key = files.__System_CorrectFilePath(key)
  if files._isLocalHost {
    fi, err := os.Stat(key);
    if err != nil { return 0 }
    return fi.Size()
  } else {
    attrs, e := files._bucketHandle.Object(key).Attrs(files._ctx)
    if e == nil {
      return attrs.Size
    } else {
      return 0
    }
  }
}

func (files *Files) KeysWithPrefix(filePrefix string) ([]string, error) {
  filePrefix = files.__System_CorrectFilePath(filePrefix)
  if files._isLocalHost {
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
  } else {
    query := &storage.Query{Prefix: filePrefix}
    it := files._bucketHandle.Objects(files._ctx, query)
    rtn := []string{}
    for {
      obj, err := it.Next()
      if err == iterator.Done { break }
      if err != nil { return nil, err }
      rtn = append(rtn, obj.Name)
    }
    return rtn, nil
  }
}

func (files *Files) Children(dirPath string) ([]string, error) {
  dirPath = files.__System_CorrectFilePath(dirPath)
  if files._isLocalHost {
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
  } else {
    // We could use
    // ```
    // query := &storage.Query{Prefix: dirPath, Delimiter: "/"}
    // ```
    // to list all files in a directory, but this would skip all subdirs.
    if !strings.HasSuffix(dirPath, "/") {
      dirPath += "/"
    }
    query := &storage.Query{Prefix: dirPath}
    it := files._bucketHandle.Objects(files._ctx, query)
    set := make(map[string]bool)
    for {
      obj, err := it.Next()
      if err == iterator.Done { break }
      if err != nil { return nil, err }
      name := obj.Name[len(dirPath):]
      i := strings.Index(name, "/")
      if i == -1 {
        set[name] = true
      } else {
        set[name[0:i]] = true
      }
    }
    rtn := []string{}
    for key := range set {
      rtn = append(rtn, key)
    }
    return rtn, nil
  }
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
  if system.files._isLocalHost {
    children, err := ioutil.ReadDir(dirPath)
    if ! strings.HasSuffix(dirPath, "/") {
      dirPath += "/"
    }
    if err != nil { return false }
    return len(children) > 0
  } else {
    // We could use
    // ```
    // query := &storage.Query{Prefix: dirPath, Delimiter: "/"}
    // ```
    // to list all files in a directory, but this would skip all subdirs.
    if !strings.HasSuffix(dirPath, "/") {
      dirPath += "/"
    }
    query := &storage.Query{Prefix: dirPath}
    it := system.files._bucketHandle.Objects(system.files._ctx, query)
    for {
      obj, err := it.Next()
      if err == iterator.Done { break }
      if err != nil { return false }
      name := obj.Name[len(dirPath):]
      i := strings.Index(name, "/")
      if i == -1 {
        return true
      } else {
        return true
      }
    }
    return false
  }
}

func (system *System) __System_PathFromRequest(request *http.Request) string {
  parts := strings.Split(request.URL.Path, "?")
  return parts[0]
}
