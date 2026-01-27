@echo off
cd /d E:\PYPROJECT\EYESON-API\eyeson-go\eyeson-go-server
set "EYESON_API_BASE_URL=http://127.0.0.1:8888"
set "EYESON_API_USERNAME=samsonixapi"
set "EYESON_API_PASSWORD=p@KuQf!4mm!7=?G*£\X?4i5&0"
go run cmd/server/main.go
pause