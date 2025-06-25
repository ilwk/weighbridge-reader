# Weighbridge Reader

这是一个地磅读取服务，主要读取PC上串口连接的地磅信息，通过websocket转发出去

## 使用说明

### 如何使用
双击`main.exe`即可运行服务，config.json为配置文件，默认端口是8080, 如需修改端口，请修改配置文件中的`websocket_port`字段。
启动后，可通过`localhost:8080/ws`来获取地磅信息

### 开机启动

Windows下，可将`main.exe`复制到`C:\Program Files\WeighbridgeReader`目录下，并创建服务，命令如下：
```powershell
# 路径记得改为自己电脑上的路径
sc.exe create "WeighbridgeReaderSVC" binPath= "C:\Program Files\WeighbridgeReader\main.exe" start= auto 
```

### 取消开机启动

```powershell
sc.exe delete "WeighbridgeReaderSVC"
```


## 配置

```json5
{
  "serial_port": "COM1", // 串口号
  "baud_rate": 9600, // 波特率
  "websocket_port": 8080, // websocket端口
  "simulate": false // 是否模拟数据
}
```

## 开发

### 环境准备

- Go 1.16+

### 依赖安装

```bash
go mod tidy
```

### 运行

```bash
go run ./cmd/main.go
```

## 编译

```bash
go build -ldflags -H=windowsgui
```

## 其他说明

### 查看当前系统下的打印机

```powershell
# win7用这个
wmic printer get name

# win10或以上
Get-Printer | Select-Object Name
```

### 命令行打印