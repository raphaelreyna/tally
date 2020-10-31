VERSION=`git describe --tags --abbrev=0`
VERSION_FLAG=main.Version=$(VERSION)
DATE=`date +"%d-%B-%Y"`
DATE_FLAG=main.Date="${DATE}"

tally:
	go build -ldflags "-X ${VERSION_FLAG} -X ${DATE_FLAG} -s -w" -o tally ./...
	upx --best ./tally

