@echo off
REM ============================================================
REM 古代木拱桥结构力学仿真与数字化复原系统
REM Windows 启动脚本
REM ============================================================

echo ============================================================
echo 古代木拱桥结构力学仿真与数字化复原系统
echo ============================================================
echo.

echo [1] 启动后端服务 (Go)
echo [2] 启动前端静态服务
echo [3] 启动传感器模拟器
echo [4] 初始化数据库
echo [5] 启动全部服务
echo [6] 退出
echo.

set /p choice=请选择操作 (1-6): 

if "%choice%"=="1" goto start_backend
if "%choice%"=="2" goto start_frontend
if "%choice%"=="3" goto start_simulator
if "%choice%"=="4" goto init_db
if "%choice%"=="5" goto start_all
if "%choice%"=="6" goto end

:start_backend
echo.
echo 正在启动后端服务...
cd backend
if not exist .env copy .env.example .env
go run cmd/server/main.go
goto end

:start_frontend
echo.
echo 正在启动前端静态服务...
cd frontend
echo 请使用浏览器访问 index.html 或使用本地Web服务器
echo 推荐使用: python -m http.server 8000
pause
goto end

:start_simulator
echo.
echo 正在启动传感器模拟器...
cd sensor-simulator
python simulator.py --mode realtime --interval 60
goto end

:init_db
echo.
echo 正在初始化数据库...
echo 请确保 PostgreSQL + TimescaleDB 已安装并运行
echo 执行数据库脚本: psql -U postgres -f database/init.sql
echo 执行桥梁数据: psql -U postgres -d ancient_bridges -f database/bridge_data.sql
pause
goto end

:start_all
echo.
echo 正在启动全部服务...
echo 请确保 PostgreSQL + TimescaleDB 已运行
echo.

start "Bridge Backend" cmd /k "cd /d %~dp0backend && go run cmd/server/main.go"

timeout /t 3 /nobreak >nul

start "Bridge Frontend" cmd /k "cd /d %~dp0frontend && python -m http.server 8000"

timeout /t 2 /nobreak >nul

start "Sensor Simulator" cmd /k "cd /d %~dp0sensor-simulator && python simulator.py --mode realtime --interval 300"

echo.
echo 服务启动完成！
echo 后端API: http://localhost:8080
echo 前端页面: http://localhost:8000
echo.
pause

:end
