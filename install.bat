set CURDIR=%cd%
set GOPATH=%CURDIR%

go version
go build -x -o gearmand.exe .\\src\\gearman\\main.go

DEL /F /A /Q .\\bin\\gearmand.exe
move .\\gearmand.exe .\\bin\\

pause
