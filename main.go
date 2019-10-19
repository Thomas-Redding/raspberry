package main

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

var ROOT_PATH string
var IS_UI bool
var IS_HD bool

// FILE_INDEX["/foo/bar"] = ["baz", "qux"]
var FILE_INDEX map[string][]string
var INDEX_STRING = "{}"

var INDEX_HTML = `
<meta charset="utf-8">
<style>
html {
  color: white;
  overflow: hidden;
  background-color: #222;
}
#container {
  position: absolute;
  left: 0;
  top: 0;
  width: 100vw;
  height: 100vh;
  overflow-x: scroll;
}
table {
  display: block;
  overflow: hidden;
  height: 100vh;
  border-collapse: collapse;
}
tbody {
  display: block;
  height: 100vh;
}
tr {
  display: block;
  height: 100vh;
}
td {
  width: 12em;
  vertical-align: top;
}
td > div {
  height: 100vh;
  overflow-y: scroll;
}
.file:before {
  content:"• ";
}
.file {
  overflow-x: scroll;
  line-height: 1.4em;
}
a {
  color: hsl(50, 100%, 50%);
}
</style>
<script>
let gPath = [];
let FILE_INDEX = <JSON_INDEX_DATA>;
function navigateToPath() {
  window.history.replaceState({}, "", window.location.origin + "/" + gPath.join("/"));
  let container = document.getElementById("container");
  container.innerHTML = "";
  let table = document.createElement("table");
  let tbody = document.createElement("tbody");
  let tr = document.createElement("tr");
  let columnCount = 0;
  for (let i = 0; i < gPath.length + 1; ++i) {
    let subPath = gPath.slice(0, i).join("/");
    let column = document.createElement("td");
    let columnDiv = document.createElement("div");
    if (!(subPath in FILE_INDEX)) break;
    for (let j = 0; j < FILE_INDEX[subPath].length; ++j) {
      let completePath = (subPath == "" ? "" : subPath + "/") + FILE_INDEX[subPath][j];
      let file = document.createElement("div");
      file.classList.add("file");
      if (!(completePath in FILE_INDEX)) {
        let a = document.createElement("a");
        a.href = "/@/" + completePath;
        file.append(a);
        a.innerHTML = FILE_INDEX[subPath][j];
      } else {
        file.innerHTML = FILE_INDEX[subPath][j];
      }
      file.innerHTML
      if (gPath[i] == FILE_INDEX[subPath][j]) {
        file.style.backgroundColor = "hsl(240, 100%, 50%)";
      }
      columnDiv.append(file);
      column.append(columnDiv);
    }
    tr.append(column);
    ++columnCount;
  }
  tbody.append(tr);
  table.append(tbody);
  table.style.width = (12*columnCount) + "em";
  container.append(table);
};

onkeydown = (event) => {
  if (event.keyCode == 37) {
    // left
    gPath.pop();
    navigateToPath();
  } else if (event.keyCode == 39) {
    // right
    if (gPath.join("/") in FILE_INDEX) {
      gPath.push(FILE_INDEX[gPath.join("/")][0]);
      navigateToPath();
    }
  } else if (event.keyCode == 38) {
    // up
    let leaf = gPath[gPath.length - 1];
    let parent = gPath.slice(0, gPath.length - 1);
    let siblings = FILE_INDEX[parent.join("/")];
    let index = siblings.indexOf(leaf);
    if (index == 0) return;
    gPath[gPath.length - 1] = siblings[index-1];
    navigateToPath();
  } else if (event.keyCode == 40) {
    // down
    let leaf = gPath[gPath.length - 1];
    let parent = gPath.slice(0, gPath.length - 1);
    let siblings = FILE_INDEX[parent.join("/")];
    let index = siblings.indexOf(leaf);
    if (index + 1 == siblings.length) return;
    gPath[gPath.length - 1] = siblings[index+1];
    navigateToPath();
  }
};

onload = () => {
  let path = decodeURI(window.location.pathname);
  if (path[0] == "/") {
    path = path.substr(1);
  }
  if (path[path.length - 1] == "/") {
    path = path.substr(0, path.length - 1);
  }
  gPath = path.split("/");
  if (gPath.length == 1 && gPath == "") {
    gPath = [];
  }
  navigateToPath();
};
</script>
<div id="container"></div>
`

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

/*
 go run main.go /path/to/file -hd -ui
 */
func main() {
  IS_HD = false
  IS_UI = false
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
    log.Printf("Sleeping for 60 seconds...")
    time.Sleep(60 * time.Second)
    log.Printf("Building an index...")
    reconstructIndex(ROOT_PATH)
  }
  log.Printf("Serving at http://localhost:8080/...")
  http.HandleFunc("/", handle)
  log.Printf("FATAL ERROR: %v", http.ListenAndServe(":8080", nil))
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
	if IS_UI {
		if strings.HasPrefix(requestPath, "/@/") {
			requestPath = requestPath[2:]
		} else {
			warmUpDisk()
			writer.Header().Set("Content-type", "text/html")
			indexHTML := strings.Replace(INDEX_HTML, "<JSON_INDEX_DATA>", INDEX_STRING, -1)
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
