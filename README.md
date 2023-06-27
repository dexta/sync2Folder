# sync2Folder
synchronize two folder with images from source to destination
Configuration with a file config.json searched at Working, Executeable or given Path.

```bash
folderSync -conf=/path/to/the/file/
```
or local development with config in same folder
```bahs
go run folderSync.go
```

# config.json
```json
{
	"sourceDir": "/home/user/sync2Folder/testSourceDir",
	"destinationDir": "/home/user/sync2Folder/testDestinationDir",
	"fileTypes": ["jpg","png"],
	"logEnable": true,
	"logPath": "/home/user/sync2Folder/testLogFile.log",
	"verbose": true,
	"delete": true,
	"dryRun": false
}
```
### fileTypes
a list image types not file extension we test the minetype here.
```go
var magicTable = map[string]string{
    "image/jpeg":    "jpg",
    "image/png":     "png",
    "image/bmp":     "bmp",
    "image/webp":    "webp",
    "image/svg+xml": "svg",
    "image/gif":     "gif"}
```

### logEnable
Write log to a file 

### verbose
Print everything 

### delete
Delete a file in the *destinationDir* if is not in *sourceDir*

### dryRun
Do not touch any files
