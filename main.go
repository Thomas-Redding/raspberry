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
var SHOULD_SLEEP bool

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
.file {
  overflow-x: scroll;
  line-height: 1.4em;
}
.selected {
  background-color: hsl(240, 100%, 40%);
}
a {
  color: hsl(50, 100%, 50%);
}
</style>
<script>
let FILE_INDEX = <JSON_INDEX_DATA>;
</script>
<script>
function navigateToPath() {
  let td = document.createElement("TD");
  containingRow.append(td);
  let columnDiv = document.createElement("div");
  for (let i = 0; i < FILE_INDEX[""].length; ++i) {
    let fileName = FILE_INDEX[""][i];
    let file = document.createElement("div");
    file.classList.add("file");
    if (!(fileName in FILE_INDEX)) {
      let a = document.createElement("a");
      a.href = "/@/" + completePath;
      file.append(a);
      a.innerHTML = FILE_INDEX[fileName][i];
    } else {
      file.innerHTML = FILE_INDEX[fileName][i];
    }
    columnDiv.append(file);
    td.append(columnDiv);
  }
};

function selectionIndexPath() {
  let rtn = [];
  for (let i = 0; i < containingRow.children.length; ++i) {
    let cell = containingRow.children[i];
    for (let j = 0; j < cell.children[0].children.length; ++j) {
      if (cell.children[0].children[j].classList.contains("selected")) {
        rtn.push(j);
      }
    }
  }
  return rtn;
}

function filePathFromIndexPath(indexPath) {
  let rtn = [];
  for (let i = 0; i < indexPath.length; ++i) {
    let cell = containingRow.children[i];
    rtn.push(cell.children[0].children[indexPath[i]].innerText);
  }
  return rtn.join("/");
}

function fileDiv(parentPath, fileName) {
  let totalPath = parentPath + "/" + fileName;
  if (parentPath == "") totalPath = fileName;
  let file = document.createElement("DIV");
  file.classList.add("file");
  if (totalPath in FILE_INDEX) {
    file.innerHTML = fileName;
  } else {
    let a = document.createElement("A");
    a.href = "/@/" + totalPath;
    a.innerHTML = fileName;
    file.append(a);
  }
  return file;
}

onkeydown = (event) => {
  event.preventDefault();
  if (event.keyCode == 37) {
    // left
    let indexPath = selectionIndexPath()
    let filePath = filePathFromIndexPath(indexPath);
    if (filePath in FILE_INDEX) {
      // The leaf is a directory.
      let lastCell = containingRow.children[containingRow.children.length - 1];
      containingRow.removeChild(lastCell);
    }
    let lastCell = containingRow.children[containingRow.children.length - 1];
    lastCell.children[0].children[indexPath[indexPath.length - 1]].classList.remove("selected");
  } else if (event.keyCode == 39) {
    let lastCell = containingRow.children[containingRow.children.length - 1];
    lastCell.children[0].children[0].classList.add("selected");
    let s = selectionIndexPath();
    let filePath = filePathFromIndexPath(s);
    if (!(filePath in FILE_INDEX)) {
      // The leaf is a file
      return;
    }
    let children = FILE_INDEX[filePath];
    let td = document.createElement("TD");
    let div = document.createElement("DIV");
    for (let i = 0; i < children.length; ++i) {
      div.append(fileDiv(filePath, children[i]));
    }
    td.append(div);
    containingRow.append(td);
  } else if (event.keyCode == 38 || event.keyCode == 40) {
    // up or down
    let isUp = (event.keyCode == 38);
    let indexPath = selectionIndexPath()
    let filePath = filePathFromIndexPath(indexPath);
    let selectedCell = containingRow.children[indexPath.length - 1];
    let fileContainer = selectedCell.children[0];
    if (filePath in FILE_INDEX) {
      // The original leaf is a directory. Close it.
      let lastCell = containingRow.children[containingRow.children.length - 1];
      containingRow.removeChild(lastCell);
    }
    let selectedRow = indexPath[indexPath.length - 1];
    if (isUp) {
      if (selectedRow == 0) return;
      fileContainer.children[selectedRow-1].classList.add("selected");
    } else {
      if (selectedRow + 1 == fileContainer.children.length) return;
      fileContainer.children[selectedRow+1].classList.add("selected");
    }
    fileContainer.children[selectedRow].classList.remove("selected");
    fileContainer.scrollTo(0, fileContainer.children[selectedRow].offsetTop - 100);
    indexPath = selectionIndexPath()
    filePath = filePathFromIndexPath(indexPath);
    if (filePath in FILE_INDEX) {
      // The leaf is a directory. Open it.
      let td = document.createElement("TD");
      let div = document.createElement("DIV");
      let children = FILE_INDEX[filePath];
      for (let i = 0; i < children.length; ++i) {
        div.append(fileDiv(filePath, children[i]));
      }
      td.append(div);
      containingRow.append(td);
      return;
    }
  }
};

onload = () => {
  let td = document.createElement("TD");
  let div = document.createElement("DIV");
  let children = FILE_INDEX[""];
  for (let i = 0; i < children.length; ++i) {
    div.append(fileDiv("", children[i]));
  }
  td.append(div);
  containingRow.append(td);
};
</script>
<div id="container">
  <table>
    <tbody>
      <tr id="containingRow"></tr>
    </tbody>
  </table>
</div>
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
 go run main.go /path/to/file -hd -ui -sleep
 */
func main() {
  IS_HD = false
  IS_UI = false
  SHOULD_SLEEP = false
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
      } else if flag == "sleep" {
        SHOULD_SLEEP = true
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
    if SHOULD_SLEEP {
    	log.Printf("Sleeping for 60 seconds...")
    	time.Sleep(60 * time.Second)
    }
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
			slurp, _ := ioutil.ReadFile("/home/pi/raspberry/foo.html")
			indexHTML := string(slurp)
			// indexHTML := INDEX_HTML
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
