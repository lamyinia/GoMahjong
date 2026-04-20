@echo off
echo Installing protoc plugins if not already installed...
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo Setting up environment...
for /f "tokens=*" %%i in ('go env GOPATH') do set GOPATH=%%i
set PATH=%GOPATH%\bin;%PATH%

echo Generating Go files from protobuf...
protoc --go_out=../pb --go_opt=paths=source_relative --go-grpc_out=../pb --go-grpc_opt=paths=source_relative *.proto
if %errorlevel% equ 0 (
    echo Successfully generated Go files in ../pb directory
) else (
    echo Error: Failed to generate Go files
    exit /b 1
)
