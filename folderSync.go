package main

import(
  "crypto/md5"
  "encoding/json"
  "encoding/hex"
  "io/ioutil"
  "net/http"
  "strings"
  "path/filepath"
  "path"
  "flag"
  "fmt"
  "log"
  "io"
  "os"
)

type appConfig struct {
	SourceDir				string		`json:"sourceDir"`
	DestinationDir	string		`json:"destinationDir"`
	FileTypes				[]string	`json:"fileTypes"`
	LogEnable				bool			`json:"logEnable"`
	LogPath					string		`json:"logPath"`
	DryRun					bool			`json:"dryRun"`
	Verbose					bool			`json:"verbose"`
	Delete					bool			`json:"delete"`
}

var conf appConfig

type diffLog struct {
	Added						[]string
	Modif						[]string
	Remov						[]string
}

var dlog diffLog

var fileLogger *log.Logger

var magicTable = map[string]string{
    "image/jpeg":   	"jpg",
    "image/png": 			"png",
    "image/bmp": 			"bmp",
    "image/webp": 		"webp",
    "image/svg+xml": 	"svg",
    "image/gif": 			"gif"}

func doesFileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func loadConfig(altConfigPath string) {
  realConfigPath := ""
  if altConfigPath!="" {
  	realConfigPath = altConfigPath
  } else {
	  exPath, errEx := os.Executable()
		if errEx != nil {
			fmt.Println("no path to exec")
		}
		woPath, errWo := os.Getwd()
		if errWo != nil {
			fmt.Println("no path to working Dir")
		}
		if doesFileExists( path.Join(woPath, "config.json") ) {
			realConfigPath = woPath
		} else if doesFileExists( path.Join(exPath, "config.json") ) {
			realConfigPath = filepath.Dir(exPath)
		} else {
			panic("No Config File found")
		}
	}

	jsonPathFile := path.Join(realConfigPath, "config.json")
  jsonFile, err := os.Open(jsonPathFile)
  if err != nil {
   fmt.Println("Error occured while reading config "+jsonPathFile)
   return
  }

  byteValue, _ := ioutil.ReadAll(jsonFile)
  defer jsonFile.Close()

  json.Unmarshal(byteValue, &conf)
}

func verboseLog(logType string, tailValue string) {
	if conf.Verbose {
		switch logType {
			case "ln":
				fmt.Println(tailValue)
			case "print":
				fmt.Print(tailValue)
		}
	}

	if conf.LogEnable {
		fileLogger.Print(tailValue)
	}
}

func copyFile(src, dst string) (bool, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file", src)
	}

  input, err := ioutil.ReadFile(src)
  if err != nil {
    fmt.Println(err)
    return false, fmt.Errorf("%s",err)
  }

  err = ioutil.WriteFile(dst, input, 0644)
  if err != nil {
    verboseLog("ln", "Error creating " + dst)
    log.Println(err)
    return false, fmt.Errorf("%s",err)
  }

  return true, nil
}

func mkMd5Sum(filePath string) (string, error) {
	var reMD5Sum string

	file, err := os.Open(filePath)
	if err != nil {
		return reMD5Sum, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return reMD5Sum, err
	} else {
		hashInBytes := hash.Sum(nil)[:16]
		reMD5Sum = hex.EncodeToString(hashInBytes)
		return reMD5Sum, nil
	}
}

func checkImageType(fileNamePath string) bool {
	file, err := os.Open(fileNamePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512)

	_, ferr := file.Read(buffer)
	if ferr != nil {
		return false
	}
	mineType := http.DetectContentType(buffer)
	confImagName := magicTable[mineType]

	for _, a := range conf.FileTypes {
		if a==confImagName {
			return true
		}
	}

  return false
}

func getDirList(dirPath string) (map[string]string, error) {
	dirList := make(map[string]string)

	dirFiles, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return dirList, err
	}
	
	for _, f := range dirFiles {
		chFilePath := path.Join(dirPath, f.Name())
		md5Sum, err := mkMd5Sum(chFilePath)
		if err == nil && checkImageType(chFilePath) {
			dirList[f.Name()] = md5Sum	
		}
	}

	return dirList, nil
}

func compareDirLists( srcList map[string]string, dstList map[string]string) (diffLog, error) {
	for imgName, md5Sum := range srcList {
		if dstList[imgName]!="" {
			if dstList[imgName] == md5Sum {
				// fmt.Println("image are the same")
			} else {
				// fmt.Println("image are not the same")
				dlog.Modif = append(dlog.Modif, imgName)
			}
		} else {
			// fmt.Println("image not in dest folder")
			dlog.Added = append(dlog.Added, imgName)
		}
	}

	for imgName, _ := range dstList {
		if srcList[imgName]=="" {
			// fmt.Println("image not in source list")
			dlog.Remov = append(dlog.Remov, imgName)
		}
	}
	return dlog, nil
}

func copyAllFiles( aFiles diffLog) {
	countAdd := 0
	countMod := 0
	for _, a := range aFiles.Added {
		srcFile := path.Join(conf.SourceDir, a)
		dstFile := path.Join(conf.DestinationDir, a)
		done, err := copyFile(srcFile, dstFile)
		if !done {
			verboseLog("ln", "addFileFail: " + a)
			fmt.Print(err)
		}
		countAdd++
	}

	for _, m := range aFiles.Modif {
		srcFile := path.Join(conf.SourceDir, m)
		dstFile := path.Join(conf.DestinationDir, m)
		done, err := copyFile(srcFile, dstFile)
		if !done {
			verboseLog("ln", "modFileFail: " + m)
			fmt.Print(err)
		}
		countMod++
	}

	verboseLog("print", fmt.Sprintf("Copy files: Added(%d) Modif(%d)\n", countAdd, countMod))
}

func deleteAllFiles( dFiles diffLog) {
	countDel := 0
	for _, d := range dFiles.Remov {
		delFile := path.Join(conf.DestinationDir, d)
		err := os.Remove(delFile)
		if err != nil {
			verboseLog("ln", "removeFileFail: " + d)
		} else {
			verboseLog("ln", d + "File Removed")
			countDel++
		}
	}
	verboseLog("print", fmt.Sprintf("Delete files: Removed(%d)\n", countDel))
}

func init() {
	nConfPath := flag.String("conf", "", "change the config path")
  flag.Parse()

  loadConfig(*nConfPath)

	if conf.LogEnable {
		file, err := os.OpenFile(conf.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    if err != nil {
        log.Fatal(err)
    }

  	fileLogger = log.New(file, "", log.Ldate|log.Ltime)
  	fileLogger.Println("")
  	fileLogger.Println("*** programm starts running ***")
	}
}

func main() {
  sList, err := getDirList(conf.SourceDir)
  if err != nil {
      log.Fatal(err)
  }
  if len(sList) == 0 {
  	verboseLog("print", "Empty Source Dir")
  	return
  }

	dList, err := getDirList(conf.DestinationDir)
  if err != nil {
      log.Fatal(err)
  }  

  cObj, err := compareDirLists(sList, dList)
  if( err == nil) {
  	if conf.Verbose && (len(cObj.Added)>0||len(cObj.Modif)>0||len(cObj.Remov)>0) {
	  	formOut := fmt.Sprintf("\nAdded images (%d):\n\t%s\nModif images (%d):\n\t%s\nRemoved images (%d):\n\t%s\n", 
	  			len(cObj.Added), strings.Join(cObj.Added, "\n\t"), 
	  			len(cObj.Modif), strings.Join(cObj.Modif, "\n\t"),
	  			len(cObj.Remov), strings.Join(cObj.Remov, "\n\t") )
	  	verboseLog("print", formOut)
  	}
  } else {
  	log.Fatal(err)
  }

  if !conf.DryRun {
  	copyAllFiles(cObj)
  	if(conf.Delete) {
  		deleteAllFiles(cObj)
  	}
  }
}